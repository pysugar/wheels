package base

import (
	"fmt"
	"github.com/pysugar/wheels/serial"
	"runtime"
)

var (
	Version_x byte = 0
	Version_y byte = 0
	Version_z byte = 1
)

var (
	build    = "Custom"
	codename = "Netool, Penetrates Everything."
	intro    = "Net tools for everything."
)

func Version() string {
	return fmt.Sprintf("%v.%v.%v", Version_x, Version_y, Version_z)
}

// VersionStatement returns a list of strings representing the full version info.
func VersionStatement() []string {
	return []string{
		serial.Concat("Netool ", Version(), " (", codename, ") ", build, " (", runtime.Version(), " ", runtime.GOOS, "/", runtime.GOARCH, ")"),
		intro,
	}
}