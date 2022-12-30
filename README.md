# 动手写RPC框架 - GeeRPC

## Day1 服务端与消息编码

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

它启动了一个 goroutine 来运行 `startServer` 函数来处理 RPC 请求。在 `startServer` 中使用了信道 `addr`，确保服务端端口监听成功，客户端再发起请求。

main 函数使用 `net.Dial` 对服务器进行拨号，并使用 JSON 编码将选项发送到服务器。

然后，main 函数进入一个循环，在该循环中，它向服务器发送请求并读取响应。请求头指定要调用的 RPC 方法的名称和序列号，请求体是一个字符串。

使用 codec 的 `cc.ReadHeader` 和 `cc.ReadBody` 方法对响应头和体进行解码，然后将解码后的响应写入日志。

## Day2 高性能客户端

### client.go

该代码定义了一个 `Client` 结构体，表示 RPC 客户端。在 `Client` 结构体中，定义了一个 `Call` 结构体，表示一个 RPC 调用。该代码实现了对应的方法，包括向服务端发送请求、处理服务端的响应、关闭连接等。

- `Call`
  -  `done` 方法，该方法将调用的状态设置为已完成，并通过通道发送信号。
  -  `registerCall` 方法，该方法将参数 call 添加到 `client.pending` 中，并更新` client.seq`
  -  `removeCall` 方法，该方法根据 seq，从` client.pending `中移除对应的 call，并返回。
  -  `terminateCalls` 方法，该方法服务端或客户端发生错误时调用，将 shutdown 设置为 true，且将错误信息通知所有 pending 状态的 call。
- `Client`
  -  `Close` 方法，该方法用于关闭客户端连接。
  -  `IsAvailable` 方法，该方法用于检查客户端是否可用。
  -  `send` 方法，该方法用于向服务端发送一个 RPC 调用请求。
  -  `Call` 方法是一种同步方法，它包装方法并阻止，直到收到响应或发生错误。如果在 RPC 调用期间发生错误，它将返回错误。
  -  `receive` 方法
  - call 不存在，可能是请求没有发送完整，或者因为其他原因被取消，但是服务端仍旧处理了。
  - call 存在，但服务端处理出错，即 h.Error 不为空。
  - call 存在，服务端处理正常，那么需要从 body 中读取 Reply 的值。

`Go` 是一种异步方法，用于发送 RPC 请求并返回表示 RPC 调用的结构。该结构包含字段，例如要调用的服务方法、要传递的参数以及用于接收响应的通道。结构中的通道用于在收到响应或发生错误时发出信号

`NewClient`创建 Client 实例时，首先需要完成一开始的协议交换，即发送 `Option` 信息给服务端。协商好消息的编解码方式之后，再创建一个子协程调用 `receive()` 接收响应。

`Dial`  还需要实现 `Dial` 函数，便于用户传入服务端地址，创建 Client 实例。为了简化用户调用，通过 `...*Option` 将 Option 实现为可选参数。

### main.go

首先，在 main 函数中会启动一个 goroutine，该 goroutine 会启动一个 RPC 服务端。该服务端会侦听本地任意可用的端口，并将所侦听的地址传递给客户端。

然后，main 函数会使用 geerpc 包中的 Dial 函数创建一个 RPC 客户端，并连接到服务端。

接下来，main 函数会使用 sync 包中的 WaitGroup 类型创建一个 WaitGroup 实例，并启动五个 goroutine。每个 goroutine 会执行一个 RPC 调用，并使用 client.Call 函数发送请求。

每个 goroutine 执行完后，会调用 WaitGroup 实例的 Done 方法，将计数器减 1。

最后，main 函数会调用 WaitGroup 实例的 Wait 方法，等待所有 goroutine 执行完毕。