#### http://127.0.0.1:50051

```bash
$ cd examples/grpc/heartbeat/server
$ go run main.go
```

#### http://127.0.0.1:8080

```bash
$ cd examples/grpc/httproxy
$ go run main.go
```

#### https://127.0.0.1:8443

```bash
$ cd examples/grpc/tls/server
$ go run main.go

$ cd examples/grpc/https
$ go run main.go

$ cd examples/grpc/https2
$ go run main.go
```




```bash
$ go run -race main.go --url=http://127.0.0.1:50051 --concurrency=10

$ go run -race main.go --url=http://127.0.0.1:8080 --concurrency=10

$ go run -race main.go --url=https://127.0.0.1:8443 --concurrency=100
```