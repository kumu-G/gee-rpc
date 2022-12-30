package main

import (
	"encoding/json"
	"fmt"
	"geerpc"
	"geerpc/codec"
	"log"
	"net"
	"time"
)

// 程序开始定义一个 startServer 函数，该函数在一个随机的空闲端口上侦听传入连接。
// 侦听套接字的地址通过提供的 addr 通道发送。函数然后调用 geerpc.Accept 函数来开始接受连接并处理 RPC 请求。
func startServer(addr chan string) {
	// pick a free port
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	geerpc.Accept(l)
}

// 程序启动一个 goroutine 来运行 startServer 函数。
// 然后，它等待侦听套接字的地址通过 addr 通道发送。一旦收到地址，main 函数使用 net.Dial 对服务器进行拨号。
// 在建立连接后，main 函数使用 JSON 编码将选项发送到服务器。然后，它创建一个新的 codec.GobCodec，用于在连接上编码和解码数据。
func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)

	// in fact, following code is like a simple geerpc client
	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)
	// send options
	_ = json.NewEncoder(conn).Encode(geerpc.DefaultOption)
	cc := codec.NewGobCodec(conn)
	// send request & receive response
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		_ = cc.Write(h, fmt.Sprintf("geerpc req %d", h.Seq))
		_ = cc.ReadHeader(h)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}
