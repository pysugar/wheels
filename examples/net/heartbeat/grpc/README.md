





```bash
$ go get -u google.golang.org/grpc
$ go get -u github.com/golang/protobuf/protoc-gen-go

$ protoc --go_out=./ --go_opt=paths=source_relative \
       --go-grpc_out=./ --go-grpc_opt=paths=source_relative \
       heartbeat/heartbeat.proto
```