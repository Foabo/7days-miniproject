package geerpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"sync"
)

// Call 结构体代表了一次远程过程调用
// 首先是序号，然后是调用的方法，方法的参数
// Done 是是为了支持异步调用，将完成的一次RPC发到chan
type Call struct {
	Seq           uint64
	ServiceMethod string      // format "<service>.<method>"
	Args          interface{} // arguments to the function
	Reply         interface{} // reply from the function
	Error         error       // if error occurs, it will be set
	Done          chan *Call  // Strobes when call is complete.

}

// 当调用结束时，会调用 call.done() 通知调用方。
func (call *Call) done() {
	call.Done <- call
}

// 核心Client
// Client的作用肯定是与服务端通信，将数据写入conn,获取服务端的数据
// Client represents an RPC Client.
// There may be multiple outstanding Calls associated
// with a single Client, and a Client may be used by
// multiple goroutines simultaneously.
type Client struct {
	cc       codec.Codec      // Codec接口，实际上是GobCodec，主要是操作conn io流
	opt      *Option          // 记录了数据流的Option结构，包含编解码方法
	sending  sync.Mutex       // 发送数据时候要加锁,保证请求有序发到conn的缓冲区
	header   codec.Header     // 发送数据的头部，只在发送时候需要，每个客户端只有一个
	mu       sync.Mutex       // 关闭连接时候要加锁
	seq      uint64           // 给发送的请求编号，每个请求拥有唯一编号。和call的seq对应
	pending  map[uint64]*Call // 存储未处理完的请求，键是编号，值是 Call 实例。
	closing  bool             // 客户端关闭了连接，用户主动的
	shutdown bool             // 服务端关闭了连接，用户被动的，一般有错误发生
}

// 编译时检查是否实现了io.Closer接口
var _ io.Closer = (*Client)(nil)

var ErrShutDown = errors.New("connection is shut down")

// Close 关闭与服务端的套接字连接
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	// 并发状况下可能有多个client关闭连接
	// 如果这时候已经被关闭了
	if client.closing {
		return ErrShutDown
	}
	client.closing = true
	// 最终是关闭套接字连接
	return client.cc.Close()
}

// IsAvailable return true if the client does work
func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing
}

// 给call的Seq赋值，同时将call加入到client的pending中，按序调用call
func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown {
		return 0, ErrShutDown
	}
	call.Seq = client.seq
	client.pending[client.seq] = call
	client.seq++
	return call.Seq, nil
}

// 调用完call就的删掉了
// 根据 seq，从 client.pending 中移除对应的 call，并返回。
func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

// 提前终止client与服务端连接，一般是发生了错误
// 服务端或客户端发生错误时调用，将 shutdown 设置为 true，且将错误信息通知所有 pending 状态的 call。
func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}

}

// 客户端接收服务端的消息
func (client *Client) receive() {
	var err error
	// for循环持续接收服务端消息
	for err == nil {
		var h codec.Header
		// 读取服务器的response，先读取header到h中
		if err = client.cc.ReadHeader(&h); err != nil {
			break
		}
		call := client.removeCall(h.Seq)
		switch {
		case call == nil:
			// call == nil 通常意味着可能是conn发送请求也就是Write失败
			// 或者是call已经被移除了
			err = client.cc.ReadBody(nil)

		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			err = client.cc.ReadBody(nil)
			call.done()

		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()

		}
	}
	// error occurs, so terminateCalls pending calls
	// 错误发生时候，客户端所有没调用的call都被终止
	client.terminateCalls(err)
}

// 创建 Client 实例时，首先需要完成一开始的协议交换，
// 即发送 Option 信息给服务端。协商好消息的编解码方式之后，
// 再创建一个子协程调用 receive() 接收响应。
func NewClient(conn net.Conn, opt *Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	}
	// send options with server
	// opt 编码到conn中，也就是相当于写到套接字的写缓冲
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client: options error: ", err)
		_ = conn.Close()
		return nil, err
	}
	return newClientCodec(f(conn), opt), nil
}

// new一个Client
// 事实上，在这里是先有了套接字连接conn，才有了传进来的Codec
// 所以这时候client可以负责conn的receive操作
func newClientCodec(cc codec.Codec, opt *Option) *Client {
	client := &Client{
		seq:     1, // seq starts with 1, 0 means invalid call
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}

// parseOptions的作用是可以让Option使用一个默认的DefaultOption
// 这样就不用每次都构造了，但无论如何，Option一定要有，且只有一个
func parseOptions(opts ...*Option) (*Option, error) {
	if len(opts) == 0 || opts[0] == nil {
		return DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of options is more than 1")
	}
	opt := opts[0]
	opt.MagicNumber = DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = DefaultOption.CodecType
	}
	return opt, nil

}

// Dial connects to an RPC server at the specified network address
func Dial(network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	// close the connection if client is nil
	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()
	return NewClient(conn, opt)
}

// 实现发送请求的功能
func (client *Client) send(call *Call) {
	// 保证一次请求的数据是连续发到套接字缓冲区
	client.sending.Lock()
	defer client.sending.Unlock()

	// 为client注册这个call
	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	// 构造请求的头部信息
	client.header.Seq = seq
	client.header.ServiceMethod = call.ServiceMethod
	client.header.Error = ""

	if err := client.cc.Write(&client.header, call.Args); err != nil {
		call = client.removeCall(seq)
		// call may be nil, it usually means that Write partially failed,
		// client has received the response and handled
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

// Go invokes the function asynchronously.
// It returns the Call structure representing the invocation.
// Go的作用是构造一个call，发送请求给服务端啊
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}
	// 构造发送请求的call
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	client.send(call)
	return call
}

// Call invokes the named function, waits for it to complete,
// and returns its error status.
// Call的作用是调用Go，让客户端发送请求给服务端，请求完成后，会在recive()函数调用call.done()
// 将完成的call放到Done这个通道中
func (client *Client) Call(serviceMethod string, args, reply interface{}) error {
	call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
	return call.Error

}
