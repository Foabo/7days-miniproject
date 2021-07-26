package geerpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int        // MagicNumber marks this's a geerpc request
	CodecType   codec.Type // client may choose different Codec to encode body
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

// Server represents an RPC Server.
// Server 只有一个属性，serviceMap，代表了能提供的服务列表
type Server struct {
	serviceMap sync.Map
}

// NewServer returns a new Server.
func NewServer() *Server {
	return &Server{}
}

// DefaultServer is the default instance of *Server.
var DefaultServer = NewServer()

// RegistService 为Server注册服务
func (server *Server) RegisterService(recvr interface{}) error {
	s := newService(recvr)
	fmt.Printf("RegisterService: s的name为%s \n", s.name)
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc: service already defined: " + s.name)
	}
	return nil
}

// Register publishes the receiver's methods in the DefaultServer.
func RegisterService(recvr interface{}) error {
	return DefaultServer.RegisterService(recvr)
}

func (server *Server) findService(serviceMethod string) (svc *service, mtype *methodType, err error) {
	dot := strings.LastIndex(serviceMethod, ".") // Service.Method
	if dot < 0 {
		err = errors.New("rpc server: service/method request ill-formed: " + serviceMethod)
	}
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	// 从serviceMap找到service的实例
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service " + serviceName)
		return
	}
	svc = svci.(*service) //接口强转成service
	// 从service的实例中找到方法
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc server: can't find method \" + methodName")
	}
	return
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
// lis代表监听套接字
// go 中socket变成只需要使用 Listen + Accept的形式就可以了
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept() // 返回三次握手后，用于连接的套接字
		if err != nil {
			log.Println("rpc server : accept error:", err)
			return
		}
		go server.ServerConn(conn)
	}
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

// ServeConn 的实现就和之前讨论的通信过程紧密相关了，
// 首先使用 json.NewDecoder 反序列化得到 Option 实例，
// 检查 MagicNumber 和 CodeType 的值是否正确。
// 然后根据 CodeType 得到对应的消息编解码器，接下来的处理交给 serverCodec。
// 参数本来应该是net.Conn才对，这里为什么可以是io.ReadWriteCloser，这是因为net.Conn实现了Read,Write,Close方法
// 本质上他们都是接口的定义，具体的实例才是在内存里面实际的东西
func (server *Server) ServerConn(conn io.ReadWriteCloser) {
	defer func() { conn.Close() }()
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}
	f := codec.NewCodecFuncMap[opt.CodecType] // 这是个func,返回实现了Codec接口的结构，这里是GobCodec
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}
	server.serverCodec(f(conn))
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

func (server *Server) serverCodec(cc codec.Codec) {
	sending := new(sync.Mutex) // make sure to send a complete response
	wg := new(sync.WaitGroup)  // wait until all request are handled
	for {
		// 不停的接收请求
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break // it's not possible to recover, so close the connection
			}
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending) // 发送body为空
			continue
		}
		wg.Add(1)
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()

}

// request stores all information of a call
type request struct {
	h            *codec.Header // header of request
	argv, replyv reflect.Value // argv and replyv of request
	mtype        *methodType   // rpc的调用方法的结构
	svc          *service      // rpc调用服务名
}

// readRequestHeader 根据输入的编解码的结构，解析Header到返回值 h 中
func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

// readRequest 读取客户端发来的请求
// 读取Header 和 body到返回值 自定义的request结构中
func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	// server从注册的方法中找到对应的service和方法
	req.svc, req.mtype, err = server.findService(h.ServiceMethod)
	if err != nil {
		return req, err
	}
	// 得先看懂methodType的定义和下面这两个函数
	// 这两个都是返回一个空的结构，大概就是根据参数和返回的类型初始化一个新的
	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	// 确保argvi是一个接口，ReadBody需要传入的是一个接口interface{}
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}
	// conn的数据读到argvi中
	if err = cc.ReadBody(argvi); err != nil {
		log.Println("rpc server : read body err:", err)
		return req, err
	}

	return req, nil
}

// sendResponse 向客户端处理结果
func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil { // h,body encode到 cc.buf 结构
		log.Println("rpc server: write response error:", err)
	}
}

// handleRequest负责调用rpc方法
func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {

	defer wg.Done()
	log.Println("handle request:", req.h, req.argv.Type())
	// 调用call，
	err := req.svc.call(req.mtype, req.argv, req.replyv)
	if err != nil {
		req.h.Error = err.Error()
		server.sendResponse(cc, req.h, invalidRequest, sending)
		return
	}
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}
