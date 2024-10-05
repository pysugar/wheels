module github.com/pysugar/wheels/examples

go 1.23.1

replace github.com/pysugar/wheels => ../

require (
	github.com/pysugar/wheels v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.34.2
)

require github.com/pires/go-proxyproto v0.7.0
