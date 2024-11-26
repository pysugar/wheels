package subcmds

import (
	"context"
	"fmt"
	"github.com/pysugar/wheels/cmd/base"
	"github.com/spf13/cobra"
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"strings"
	"time"
)

var etcdGetCmd = &cobra.Command{
	Use:   `etcdget [--endpoints=127.0.0.1:2379] --key=key`,
	Short: "Get etcd values according to give key",
	Long: `
Get etcd values according to give key.
`,
	Run: func(cmd *cobra.Command, args []string) {
		endpoints, _ := cmd.Flags().GetString("endpoints")
		client, err := clientv3.New(clientv3.Config{
			Endpoints: strings.Split(endpoints, ","),
		})
		if err != nil {
			log.Fatal(err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		key, _ := cmd.Flags().GetString("key")
		limit, _ := cmd.Flags().GetInt64("limit")
		prefix, _ := cmd.Flags().GetBool("prefix")

		options := []clientv3.OpOption{clientv3.WithLimit(limit)}
		if prefix {
			options = append(options, clientv3.WithPrefix())
		}
		resp, err := client.Get(ctx, key, options...)
		if err != nil {
			log.Fatal(err)
			return
		}

		for _, kv := range resp.Kvs {
			fmt.Printf("%s : %s\n", kv.Key, string(kv.Value))
		}
	},
}

func init() {
	etcdGetCmd.Flags().StringP("endpoints", "p", "127.0.0.1:2379", "etcd server address")
	etcdGetCmd.Flags().StringP("key", "K", "", "search key")
	etcdGetCmd.Flags().Int64P("limit", "L", 100, "WithLimit")
	etcdGetCmd.Flags().BoolP("prefix", "P", false, "WithPrefix")
	etcdGetCmd.Flags().BoolP("verbose", "V", false, "Verbose mode")
	base.AddSubCommands(etcdGetCmd)
}
