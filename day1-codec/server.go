package geerpc

import (
	"encoding/json"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

// Option 结构体用于存储客户端传来的选项信息
type Option struct {
	MagicNumber int        // MagicNumber 标记这是一个grpc请求
	CodecType   codec.Type // 客户端可以选择不同的编解码器
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

// Server 结构体表示一个 RPC 服务端
type Server struct{}

// NewServer 函数用于创建一个新的服务端
func NewServer() *Server {
	return &Server{}
}

// DefaultServer 是一个默认的服务端实例
var DefaultServer = NewServer()

// ServeConn 方法用于在单个连接上运行服务端
// 服务端先读取客户端传来的选项信息，然后根据客户端选择的编解码器类型创建一个编解码器
// 调用 serveCodec 方法处理客户端的请求
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}
	server.serveCodec(f(conn))
}

// invalidRequest 是发生错误时响应argv的占位符
var invalidRequest = struct{}{}

// serveCodec 方法用于处理客户端的请求
// 服务端使用 readRequest 方法读取客户端的请求
// readRequest 方法首先调用 readRequestHeader 方法读取请求头，然后读取请求参数并返回一个 request 结构体。
// request 结构体包含了请求头信息、请求参数和回复参数的值
// 当服务端读取到客户端的请求之后，会开启一个新的 goroutine 来处理这个请求，使用 handleRequest 方法进行处理。
func (server *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex) // 确保发送完整的响应
	wg := new(sync.WaitGroup)  // 等待所有请求得到处理
	for {
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break // it's not possible to recover, so close the connection
			}
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

// request 存储通话的所有信息
type request struct {
	h            *codec.Header // 请求头
	argv, replyv reflect.Value // 请求的参数和响应
}

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

// readRequest 用于读取客户端的请求
func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	// TODO: now we don't know the type of request argv
	// day 1, just suppose it's string
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}

// sendResponse 发送响应
func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

// handleRequest 方法首先获取请求中指定的服务和方法名称，然后调用相应的服务方法。
// 如果调用成功，则使用 sendResponse 方法向客户端发送响应。如果调用失败，则将错误信息记录在请求头中，并向客户端发送一个错误响应。
// sendResponse 方法使用传入的编解码器将响应头和响应参数写入到连接中，并返回给客户端。
func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("geerpc resp %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		go server.ServeConn(conn)
	}
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func Accept(lis net.Listener) { DefaultServer.Accept(lis) }
