package codec

import "io"

/*
	ServiceMethod 是服务名和方法名，通常与 Go 语言中的结构体和方法相映射。
	Seq 是请求的序号，也可以认为是某个请求的 ID，用来区分不同的请求。
	Error 是错误信息，客户端置为空，服务端如果如果发生错误，将错误信息置于 Error 中。
*/
type Header struct {
	ServiceMethod string // format "Service.Method"
	Seq           uint64 // sequence number chosen by client
	Error         string
}

// Codec 对消息体进行编解码的接口
// 抽象出接口是为了实现不同的 Codec 实例：
type Codec interface {
	io.Closer
	ReadHeader(header *Header) error  // 解码header
	ReadBody(interface{}) error       // 解码body
	Write(*Header, interface{}) error // 编码Header和body
}

// NewCodecFunc 是 Codec 的构造函数
// io.ReadWriteCloser 以组合方式实现了 Read Write Closer接口
type NewCodecFunc func(closer io.ReadWriteCloser)Codec

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json" // not implemented
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init(){
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
