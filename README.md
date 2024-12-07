# Common Useful Tools by Golang

```bash
$ netool help
A simple CLI for Net tool

Usage:
  netool [flags]
  netool [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  devtool     Start a DevTool for HTTP
  discovery   Discovery Service from NamingService
  echoservice Start a gRPC echo service
  fetch       fetch http2 response from url
  fileserver  Start a File Server
  grpc        call grpc service
  help        Help about any command
  httpproxy   Start a Transparent HTTP Proxy
  rand        Generate Rand String
  read-proto  Read proto binary file
  registry    Register Service to NamingService
  signature   Signature Commands
  uuid        Generate UUIDv4 or UUIDv5
  version     Show current version of Netool
  wg          Generate key pair for wireguard key exchange
  x25519      Generate key pair for x25519 key exchange

Flags:
  -h, --help   help for netool

Use "netool [command] --help" for more information about a command.

$ netool help fetch                                                                                                                                      2:29:05 

fetch http2 response from url

fetch http2 response from url: netool fetch https://www.google.com
call grpc service: netool fetch --grpc https://localhost:8443/grpc.health.v1.Health/Check
call grpc via context path: netool fetch --grpc http://localhost:8080/grpc/grpc.health.v1.Health/Check
call grpc service: netool fetch --grpc https://localhost:8443/grpc.health.v1.Health/Check --proto-path=health.proto -d'{"service": ""}'

Usage:
  netool fetch https://www.google.com [flags]

Flags:
  -d, --data string         request data (default "{}")
  -G, --grpc                Is GRPC Request Or Not
  -H, --header strings      HTTP Header
  -h, --help                help for fetch
      --http1               Is HTTP1 Request Or Not
      --http2               Is HTTP2 Request Or Not
  -i, --insecure            Skip server certificate and domain verification (skip TLS)
  -M, --method string       HTTP Method (default "GET")
  -P, --proto-path string   Proto Path
  -U, --upgrade             try http upgrade
  -A, --user-agent string   User Agent
  -V, --verbose             Verbose mode
  -W, --websocket           Is WebSocket Request Or Not

$ netool help grpc                                                                                                                                              
call grpc service

Send an empty request:                     netool grpc grpc.server.com:443 my.custom.server.Service/Method
Send a request with a header and a body:   netool grpc -H "Authorization: Bearer $token" -d '{"foo": "bar"}' grpc.server.com:443 my.custom.server.Service/Method
List all services exposed by a server:     netool grpc grpc.server.com:443 list
List all methods in a particular service:  netool grpc grpc.server.com:443 list my.custom.server.Service

Usage:
  netool grpc -d '{}' 127.0.0.1:50051 grpc.health.v1.Health/Check [flags]

Flags:
  -c, --context-path string   context path
  -d, --data string           request data (default "{}")
  -H, --header stringArray    Extra header to include in information sent
  -h, --help                  help for grpc
  -i, --insecure              Skip server certificate and domain verification (skip TLS)
  -p, --plaintext             Use plain-text HTTP/2 when connecting to server (no TLS)
  
  
$ netool echoservice --port=50051

$ netool fetch --grpc http://localhost:50051/proto.EchoService/Echo \
  --proto-path=echo.proto -d'{"message": "netool"}'
$ netool fetch --grpc http://localhost:50051/grpc.health.v1.Health/Check \
  --proto-path=health.proto -d'{"service": "echoservice"}'
```

