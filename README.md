```bash
$ go install -v google.golang.org/protobuf/cmd/protoc-gen-go@latest
$ go install -v google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# /usr/bin/env protoc --version
$ /usr/local/bin/protoc --version
$ protoc --go_out=. --go_opt=paths=source_relative  serial/typed_message.proto
```