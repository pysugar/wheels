package distro

import "github.com/pysugar/wheels/cmd/base"

func init() {
	base.AddSubCommands(fileServerCmd)
	base.AddSubCommands(httpProxyCmd)
	base.AddSubCommands(registryCmd)
	base.AddSubCommands(discoveryCmd)
	base.AddSubCommands(devtoolCmd)
}
