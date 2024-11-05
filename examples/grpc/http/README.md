
```bash
$ go run main.go

$ curl -i -k https://127.0.0.1:8443/health

$ grpcurl --insecure 127.0.0.1:8443 grpc.health.v1.Health/Check
$ netool grpc --insecure 127.0.0.1:8443 grpc.health.v1.Health/Check

$ curl -i -k https://127.0.0.1:8443/metrics
```