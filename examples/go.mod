module github.com/pysugar/wheels/examples

go 1.23.1

replace github.com/pysugar/wheels => ../

require (
	github.com/pysugar/wheels v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/pires/go-proxyproto v0.7.0
	github.com/prometheus/client_golang v1.20.5
	google.golang.org/grpc v1.67.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240814211410-ddb44dafa142 // indirect
)
