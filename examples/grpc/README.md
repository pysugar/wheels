# gRPC 相关概念


### grpc-go

#### **ClientConn 和 Server**：

- **ClientConn**：位于`clientconn.go`，是客户端的核心结构，管理着底层的连接、负载均衡和名称解析。它负责建立连接、发送RPC请求以及处理响应。
- **Server**：位于`server.go`，是服务器端的核心，负责监听传入的连接、处理RPC请求并发送响应。

#### **传输层（Transport）**：

- 位于`transport`包，封装了底层的网络传输细节，主要实现了HTTP/2协议。`http2_client.go`和`http2_server.go`分别处理客户端和服务器端的HTTP/2逻辑。
- 传输层负责管理流（Stream）、窗口大小、帧的读写等。

#### **流（Stream）**：

- 每个RPC调用都对应一个`Stream`，位于`stream.go`。它管理着单个RPC调用的状态、元数据、读写操作等。
- 流的创建、生命周期管理以及错误处理都是通过`Stream`对象完成的。

#### **编码解码（Codec）和消息序列化**：

- `Codec`接口定义在`rpc_util.go`，用于消息的序列化和反序列化。
- 默认使用Protocol Buffers进行序列化，但grpc-go也支持自定义的编码方式。

#### **拦截器（Interceptors）**：

- 拦截器允许在RPC的调用链中插入自定义逻辑，类似于中间件。
- 分为Unary和Stream两种拦截器，定义在`interceptor.go`。

#### **名称解析和负载均衡**：

- 名称解析器（Resolver）位于`resolver`包，负责将逻辑服务名解析为物理地址。
- 负载均衡器（Balancer）位于`balancer`包，管理多个后端地址的选择策略。

#### **连接管理**：

- grpc-go实现了连接的健康检查、Keepalive、重试机制等，确保了长连接的稳定性。
- 连接的建立和关闭逻辑主要在`clientconn.go`和`transport`包中处理。

#### **错误处理**：

- 定义了一套丰富的错误码和状态，位于`codes.go`和`status`包，用于表示RPC调用的结果。

#### **元数据（Metadata）**：

- 元数据用于在客户端和服务器之间传递额外的信息，定义在`metadata`包。
- 可以用于传递认证信息、请求标识等。

#### **安全传输（Credentials）**：

- 位于`credentials`包，支持TLS、OAuth等多种认证方式。
- 可以在创建`ClientConn`和`Server`时配置。


