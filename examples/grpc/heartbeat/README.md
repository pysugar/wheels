

```bash
$ GOOS=linux GOARCH=amd64 go build -o hbserver ./server
$ GOOS=linux GOARCH=amd64 go build -o hbclient ./client

$ scp hbserver ubuntu-pn51.local:/opt/bin
$ scp hbclient ubuntu-pn51.local:/opt/bin
```

```bash
$ nohup /opt/bin/hbserver > /var/log/heartbeat/hbserver.log 2>&1 &
$ nohup /opt/bin/hbclient > /var/log/heartbeat/hbclient.log 2>&1 &

$ jobs  
```

```bash
$ grpcurl --plaintext localhost:50051 list
grpc.channelz.v1.Channelz
grpc.health.v1.Health
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection

$ grpcurl --plaintext localhost:50051 list grpc.health.v1.Health
grpc.health.v1.Health.Check
grpc.health.v1.Health.Watch

$ grpcurl --plaintext localhost:50051 list grpc.reflection.v1.ServerReflection
grpc.reflection.v1.ServerReflection.ServerReflectionInfo

$ grpcurl --plaintext localhost:50051 list grpc.channelz.v1.Channelz
grpc.channelz.v1.Channelz.GetChannel
grpc.channelz.v1.Channelz.GetServer
grpc.channelz.v1.Channelz.GetServerSockets
grpc.channelz.v1.Channelz.GetServers
grpc.channelz.v1.Channelz.GetSocket
grpc.channelz.v1.Channelz.GetSubchannel
grpc.channelz.v1.Channelz.GetTopChannels

$ grpcurl --plaintext -d '{"service": "my_service"}' localhost:50051 grpc.health.v1.Health/Check

$ grpcurl --plaintext -d '{}' localhost:50051 grpc.channelz.v1.Channelz/GetTopChannels
$ grpcurl --plaintext -d '{}' localhost:50051 grpc.channelz.v1.Channelz/GetServers
```


```bash
$ curl http://localhost:9092/metrics
```