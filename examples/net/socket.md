## socket的核心概念

在 UNIX 和类似 UNIX 的操作系统（如 Linux）中，套接字是一种通信端点，用于在不同进程之间传输数据，可以在同一台机器上，也可以跨网络。
套接字是操作系统提供的抽象，用于处理网络通信的底层细节。

### **创建套接字**

无论是服务器还是客户端，都通过系统调用 `socket()` 创建一个套接字。这一步创建了一个套接字文件描述符，代表一个通信端点。

```c
int socket(int domain, int type, int protocol);
```

- **domain**：协议族，如 `AF_INET`（IPv4）、`AF_INET6`（IPv6）。
- **type**：套接字类型，如 `SOCK_STREAM`（流套接字，用于 TCP）、`SOCK_DGRAM`（数据报套接字，用于 UDP）。
- **protocol**：协议编号，通常为 0，由系统自动选择。

### **套接字的后续操作**

- **服务器套接字（监听套接字）**：
    - **绑定（bind）**：将套接字绑定到特定的本地地址和端口。
    - **监听（listen）**：将套接字置于被动监听模式，等待传入的连接请求。
    - **接受（accept）**：接受传入的连接，返回一个新的套接字，代表与客户端的连接。

- **客户端套接字**：
    - **连接（connect）**：主动发起与服务器的连接请求。


## **示例对比：**

### **Go 语言**

```go
// 服务器套接字
listener, err := net.Listen("tcp", ":9876")
if err != nil {
    // 处理错误
}
defer listener.Close()

// 客户端套接字
conn, err := net.Dial("tcp", "server_address:9876")
if err != nil {
    // 处理错误
}
defer conn.Close()
```

### **Java**

```java
// 服务器套接字
ServerSocket serverSocket = new ServerSocket(9876);
serverSocket.close();

// 客户端套接字
Socket clientSocket = new Socket("server_address", 9876);
clientSocket.close();
```

### **Python**

```python
# 服务器套接字
server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
server_socket.bind(('', 9876))
server_socket.listen()
server_socket.close()

# 客户端套接字
client_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
client_socket.connect(('server_address', 9876))
client_socket.close()
```
