package distro

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const (
	DefaultGroup = "default"
)

var (
	NamingServices = map[string]RegisterNamingService{
		"etcd": RegisterETCD,
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

type (
	RegisterNamingService func(endpoints []string, env, service, address string) error

	Endpoint struct {
		Address  string              `json:"address"`
		Group    string              `json:"group"`
		Metadata map[string][]string `json:"metadata"`
	}

	Instance struct {
		Env         string
		ServiceName string
		Endpoint    Endpoint
	}

	Registrar interface {
		Register(ctx context.Context, instance *Instance) error
		Deregister(ctx context.Context) error
	}
)

func RegisterETCD(endpoints []string, envName, serviceName, address string) error {
	client, err := newEtcdClient(endpoints)
	if err != nil {
		log.Printf("unexpected err: %v\n", err)
		return err
	}

	appCtx := context.Background()
	registrar := NewEtcdRegistry(client)
	err = registrar.Register(appCtx, &Instance{
		ServiceName: serviceName,
		Env:         envName,
		Endpoint:    Endpoint{Address: address, Group: DefaultGroup},
	})

	if err != nil {
		log.Printf("register err: %v\n", err)
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGQUIT, syscall.SIGSTOP)
	sig := <-sigCh
	if er := registrar.Deregister(appCtx); er != nil {
		log.Printf("Deregister failure: \n", er)
	}
	fmt.Printf("\nReceived signal: %v. Exiting...\n", sig)

	return nil
}

func (i *Instance) ServiceWithEnv() string {
	return fmt.Sprintf("/%s/%s", i.Env, i.ServiceName)
}

func (i *Instance) Key() string {
	serviceWithEnv := i.ServiceWithEnv()
	serviceKeyPrefix := fmt.Sprintf("%s/", serviceWithEnv)
	if i.Endpoint.Group != DefaultGroup {
		serviceKeyPrefix = fmt.Sprintf("%s:%s/", serviceWithEnv, i.Endpoint.Group)
	}
	return fmt.Sprintf("%s%s", serviceKeyPrefix, i.Endpoint.Address)
}

func (e *Endpoint) Encode() string {
	b, _ := json.Marshal(e)
	return string(b)
}

func (e *Endpoint) Decode(value []byte) error {
	return json.Unmarshal(value, e)
}
