# GRPC

## day1 服务端与消息编码

### codec包

- coder.go  抽象出编解码的接口
- gob.go 消息编解码器的实例

### server.go

```
NewServer -> DefaultServer -> ServeConn -> serveCodec -> readRequest -> handleRequest -> sendResponse
```

- `NewServer` 函数用于创建一个新的服务端。
- `DefaultServer` 是一个默认的服务端实例。
- `ServeConn` 读取客户端传来的选项信息，然后根据客户端选择的编解码器类型创建一个编解码器
- `serveCodec` 方法用于处理客户端的请求。
- `readRequest` 方法读取请求头，然后读取请求参数并返回一个 `request` 结构体。
- `handleRequest` 当服务端读取到客户端的请求之后，会开启一个新的 goroutine 来处理这个请求，使用 `handleRequest` 方法进行处理。
- `sendResponse` 方法使用传入的编解码器将响应头和响应参数写入到连接中，并返回给客户端。

处理请求是并发的，但是回复请求的报文必须是逐个发送的，并发容易导致多个回复报文交织在一起，客户端无法解析。在这里使用锁(sending)保证

### main.go

它启动了一个 goroutine 来运行 `startServer` 函数来处理 RPC 请求。在 `startServer` 中使用了信道 `addr`，确保服务端端口监听成功，客户端再发起请求。main 函数使用 `net.Dial` 对服务器进行拨号，并使用 JSON 编码将选项发送到服务器。然后，main 函数进入一个循环，在该循环中，它向服务器发送请求并读取响应。请求头指定要调用的 RPC 方法的名称和序列号，请求体是一个字符串。使用 codec 的 `cc.ReadHeader` 和 `cc.ReadBody` 方法对响应头和体进行解码，然后将解码后的响应写入日志。



