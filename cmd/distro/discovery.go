package distro

import (
	"log"
	"strings"

	"github.com/pysugar/wheels/cmd/distro/servicegovernance"
	"github.com/spf13/cobra"
)

var (
	NamingDiscoverGetServices = map[string]servicegovernance.DiscoverNamingService{
		"etcd": servicegovernance.DiscoverETCD,
	}

	discoveryCmd = &cobra.Command{
		Use:   `discovery [--naming-type=etcd] [--endpoints=127.0.0.1:2379] [--env-name=live] --service=service-name --watch`,
		Short: "Discovery Service from NamingService",
		Long: `
Discovery Service from NamingService.

Register a Service: netool discovery --endpoints=127.0.0.1:2379 --env-name=live --service=service-name --watch
`,
		Run: func(cmd *cobra.Command, args []string) {
			namingType, _ := cmd.Flags().GetString("naming-type")
			if fn, has := NamingDiscoverGetServices[namingType]; has {
				serviceName, _ := cmd.Flags().GetString("service")
				endpoints, _ := cmd.Flags().GetString("endpoints")
				envName, _ := cmd.Flags().GetString("env-name")
				group, _ := cmd.Flags().GetString("group")
				watchEnabled, _ := cmd.Flags().GetBool("watch")

				if eps, err := fn(strings.Split(endpoints, ","), envName, serviceName, group, watchEnabled); err != nil {
					log.Printf("discover to %s failure: %v\n", namingType, err)
				} else {
					log.Printf("discover (watch: %v) /%s/%s:%s/:\n", watchEnabled, envName, serviceName, group)
					for _, ep := range eps {
						log.Printf("\t%s - %s\n", ep.Address, ep.Group)
					}
				}
				return
			}
			log.Printf("Unsupported naming type: %s\n", namingType)
		},
	}
)

func init() {
	discoveryCmd.Flags().StringP("endpoints", "p", "127.0.0.1:2379", "naming service addresses")
	discoveryCmd.Flags().StringP("naming-type", "t", "etcd", "naming service type")
	discoveryCmd.Flags().StringP("env-name", "e", "live", "env name")
	discoveryCmd.Flags().StringP("service", "s", "", "your service")
	discoveryCmd.Flags().StringP("group", "g", "default", "group")
	discoveryCmd.Flags().BoolP("watch", "w", false, "watch enabled")
}
