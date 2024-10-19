package distro

import "github.com/pysugar/wheels/cmd/base"

func init() {
	base.AddSubCommands(fileServerCmd)
}
