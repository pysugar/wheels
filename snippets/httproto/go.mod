module github.com/pysugar/wheels/snippets/httproto

go 1.21

replace github.com/pysugar/wheels => ../../

require (
	github.com/pysugar/wheels v0.0.0-00010101000000-000000000000
	golang.org/x/net v0.31.0
)

require golang.org/x/text v0.20.0 // indirect
