import socket

def main():
    # 创建 TCP/IP 套接字
    server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)

    # 绑定套接字到地址 ('', 9876)，'' 表示所有网络接口
    server_address = ('', 9876)
    server_socket.bind(server_address)
    print("服务器正在监听端口 9876...")

    # 开始监听连接，参数为允许的最大连接数
    server_socket.listen(5)

    while True:
        # 等待客户端连接（此方法会阻塞，直到有客户端连接）
        client_socket, client_address = server_socket.accept()
        print(f"收到来自 {client_address} 的连接")

        try:
            data = b""
            while True:
                chunk = client_socket.recv(1024)
                if not chunk:
                    break
                data += chunk

            received_message = data.decode('utf-8')
            print(f"从客户端接收到的消息: {received_message}")

            response = received_message.upper()
            client_socket.send(f"服务器收到您的消息: {response}, client: {str(client_address)}".encode('utf-8'))
        except Exception as e:
            print(f"处理客户端数据时发生错误: {e}")
        finally:
            client_socket.close()
            print(f"与 {client_address} 的连接已关闭")


if __name__ == '__main__':
    main()
