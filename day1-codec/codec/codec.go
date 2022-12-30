package codec // Package codec 消息编解码相关代码

import (
	"io"
)

// Header 客户端发送请求，除了参数和返回值的剩余信息
type Header struct {
	ServiceMethod string // 服务名和方法名
	Seq           uint64 // 某个请求的 ID 用来区分不同的请求
	Error         string //	如果服务端发生错误，将信息置于 Error 中
}

// Codec 抽象出消息编解码的接口
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
	JsonType Type = "application/json" // not implemented
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
