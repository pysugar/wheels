
```bash
$ openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes
```

```bash
$ GODEBUG=http2debug=2 go run server.go

$ curl -i -k https://127.0.0.1:8443/health
$ curl -i -k https://127.0.0.1:8443/metrics
```

```bash
$ netool fetch --verbose https://127.0.0.1:8443/health
$ netool fetch --verbose https://127.0.0.1:8443/metrics

$ netool fetch --websocket --verbose https://127.0.0.1:8443/websocket

$ netool fetch --grpc --verbose https://localhost:8443/grpc/proto.EchoService/Echo --proto-path=../grpc/proto/echo.proto -d'{"message": "netool"}'
```

```bash
$ netool grpc --insecure 127.0.0.1:8443 --context-path=grpc grpc.health.v1.Health/Check
$ netool grpc --insecure 127.0.0.1:8443 --context-path=grpc proto.EchoService/Echo -d'{"message": "netool"}'
```
