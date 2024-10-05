package platform

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	PluginLocation  = "wheels.location.plugin"
	ConfigLocation  = "wheels.location.config"
	ConfdirLocation = "wheels.location.confdir"
	ToolLocation    = "wheels.location.tool"
	AssetLocation   = "wheels.location.asset"

	UseReadV         = "wheels.buf.readv"
	UseFreedomSplice = "wheels.buf.splice"
	UseVmessPadding  = "wheels.vmess.padding"
	UseCone          = "wheels.cone.disabled"

	BufferSize           = "wheels.ray.buffer.size"
	BrowserDialerAddress = "wheels.browser.dialer"
	XUDPLog              = "wheels.xudp.show"
	XUDPBaseKey          = "wheels.xudp.basekey"
)

type EnvFlag struct {
	Name    string
	AltName string
}

func NewEnvFlag(name string) EnvFlag {
	return EnvFlag{
		Name:    name,
		AltName: NormalizeEnvName(name),
	}
}

func (f EnvFlag) GetValue(defaultValue func() string) string {
	if v, found := os.LookupEnv(f.Name); found {
		return v
	}
	if len(f.AltName) > 0 {
		if v, found := os.LookupEnv(f.AltName); found {
			return v
		}
	}

	return defaultValue()
}

func (f EnvFlag) GetValueAsInt(defaultValue int) int {
	useDefaultValue := false
	s := f.GetValue(func() string {
		useDefaultValue = true
		return ""
	})
	if useDefaultValue {
		return defaultValue
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return defaultValue
	}
	return int(v)
}

func NormalizeEnvName(name string) string {
	return strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(name)), ".", "_")
}

func getExecutableDir() string {
	exec, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exec)
}

func getExecutableSubDir(dir string) func() string {
	return func() string {
		return filepath.Join(getExecutableDir(), dir)
	}
}

func GetPluginDirectory() string {
	pluginDir := NewEnvFlag(PluginLocation).GetValue(getExecutableSubDir("plugins"))
	return pluginDir
}

func GetConfigurationPath() string {
	configPath := NewEnvFlag(ConfigLocation).GetValue(getExecutableDir)
	return filepath.Join(configPath, "config.json")
}

// GetConfDirPath reads "xray.location.confdir"
func GetConfDirPath() string {
	configPath := NewEnvFlag(ConfdirLocation).GetValue(func() string { return "" })
	return configPath
}
