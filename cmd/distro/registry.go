package distro

import (
	"github.com/pysugar/wheels/cmd/distro/servicegovernance"
	"github.com/spf13/cobra"
	"log"
	"strings"
)

var (
	NamingServices = map[string]servicegovernance.RegisterNamingService{
		"etcd": servicegovernance.RegisterETCD,
	}

	registryCmd = &cobra.Command{
		Use:   `registry [--naming-type=etcd] [--endpoints=127.0.0.1:2379] [--env-name=live] --service=service-name --address=192.168.1.5:8080`,
		Short: "Register Service to Registrar",
		Long: `
Register Service to Registrar.

Register a Service: netool registry --endpoints=127.0.0.1:2379 --env-name=live --service=service-name --address=192.168.1.5:8080

ETCDCTL_API=3 etcdctl get '/live/service-name' --endpoints=127.0.0.1:2379 --prefix
`,
		Run: func(cmd *cobra.Command, args []string) {
			namingType, _ := cmd.Flags().GetString("naming_type")
			if fn, has := NamingServices[namingType]; has {
				serviceName, _ := cmd.Flags().GetString("service")
				endpoints, _ := cmd.Flags().GetString("endpoints")
				address, _ := cmd.Flags().GetString("address")
				envName, _ := cmd.Flags().GetString("env")

				if err := fn(strings.Split(endpoints, ","), envName, serviceName, address); err != nil {
					log.Printf("register to %s failure: %v\n", namingType, err)
				}
				return
			}
			log.Printf("Unsupported naming type: %s\n", namingType)
		},
	}
)

func init() {
	registryCmd.Flags().StringP("endpoints", "p", "127.0.0.1:2379", "naming service addresses")
	registryCmd.Flags().StringP("naming_type", "t", "etcd", "naming service type")
	registryCmd.Flags().StringP("env", "e", "live", "env name")
	registryCmd.Flags().StringP("service", "s", "", "your service")
	registryCmd.Flags().StringP("address", "a", "", "your service address")
}
