
## protoc

```bash
$ go install -v google.golang.org/protobuf/cmd/protoc-gen-go@latest
$ go install -v google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# /usr/bin/env protoc --version
$ /usr/local/bin/protoc --version
$ protoc --go_out=. --go_opt=paths=source_relative serial/typed_message.proto

$ protoc --go_out=. --go_opt=paths=source_relative net/address.proto
$ protoc --go_out=. --go_opt=paths=source_relative net/network.proto
$ protoc --go_out=. --go_opt=paths=source_relative net/port.proto
$ protoc --go_out=. --go_opt=paths=source_relative net/destination.proto

$ protoc --go_out=. --go_opt=paths=source_relative transport/internet/config.proto
```

## generate

### 安装`mockgen`（可选）

> `go run github.com/golang/mock/mockgen` 直接运行 mockgen 工具，无需提前安装。Go 会下载并运行该包

```bash
$ go get github.com/golang/mock/gomock
$ go install github.com/golang/mock/mockgen@latest
```

```bash
$ go generate ./...
```


