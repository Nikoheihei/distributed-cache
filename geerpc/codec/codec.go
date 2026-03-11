package codec

import "io"

type Header struct {
	ServiceMethod string // format "Service.Method"
	Seq           uint64 //客户端发出请求的序列号
	Error         string
}

// 抽象出对消息编解码的接口，支持多种编解码方式
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json" //还没实现
)

var NewCodecFuncMap map[Type]NewCodecFunc

// 抽象出codec的构造函数，根据不同的编解码方式创建不同的Codec实例
// 和工厂模式类似，但返回的不是实例而是构造函数
func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
