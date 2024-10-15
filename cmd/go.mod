module github.com/pysugar/wheels/cmd

go 1.23.2

replace github.com/pysugar/wheels => ../

require (
	github.com/pysugar/wheels v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.8.1
	golang.org/x/crypto v0.28.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)
