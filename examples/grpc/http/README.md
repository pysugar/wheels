


```bash
$ go run main.go

$ curl -i -k http://127.0.0.1:8080/health

$ curl --http2 -k -v -d '{}' http://127.0.0.1:8080/grpc/grpc.health.v1.Health/Check -H 'Content-Type: application/grpc'
$ netool grpc --context-path=grpc --plaintext 127.0.0.1:8080 grpc.health.v1.Health/Check

$ curl -i -k http://127.0.0.1:8080/metrics
```