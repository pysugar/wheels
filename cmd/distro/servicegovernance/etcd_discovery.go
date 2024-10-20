package servicegovernance

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type (
	etcdDiscoverer struct {
		cli *clientv3.Client
		rev int64
	}
)

func DiscoverETCD(etcdEndpoints []string, envName, serviceName, group string, watchEnabled bool) ([]*Endpoint, error) {
	client, err := newEtcdClient(etcdEndpoints)
	if err != nil {
		log.Printf("unexpected err: %v\n", err)
		return nil, err
	}

	serviceWithEnv := fmt.Sprintf("/%s/%s", envName, serviceName)
	serviceDiscoverKey := fmt.Sprintf("%s/", serviceWithEnv)
	if group != DefaultGroup && group != "" {
		serviceDiscoverKey = fmt.Sprintf("%s:%s/", serviceWithEnv, group)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	discoverer := NewEtcdDiscoverer(client)
	if watchEnabled {
		endpoints, watcher, er := discoverer.Watch(ctx, serviceDiscoverKey)
		if er != nil {
			log.Printf("discover watch err: %v\n", er)
			return nil, err
		}
		log.Printf("[%s] updateState\n", watcher.Service())
		for _, ep := range endpoints {
			log.Printf("\t[%s] endpoint (%s - %s)\n", watcher.Service(), ep.Address, ep.Group)
		}

		go watching(ctx, watcher)

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
		sig := <-sigCh

		if cerr := watcher.Close(); cerr != nil {
			log.Printf("close wathcer error:%v \n", cerr)
		}
		log.Printf("[%s] discover watch received signal: %v, exiting...\n", watcher.Service(), sig)
		return endpoints, nil
	} else {
		endpoints, er := discoverer.Get(ctx, serviceDiscoverKey)
		if er != nil {
			log.Printf("discover get err: %v\n", er)
			return nil, er
		}
		return endpoints, nil
	}
}

func watching(ctx context.Context, watcher Watcher) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("discoverer watching done")
			return
		default:
		}

		endpoints, err := watcher.Next()
		if err != nil {
			time.Sleep(time.Second)
			log.Printf("discoverer watching error: %v\n", err)
			continue
		}
		log.Printf("[%s] updateState\n", watcher.Service())
		for _, ep := range endpoints {
			log.Printf("\t[%s] endpoint (%s - %s)\n", watcher.Service(), ep.Address, ep.Group)
		}
	}
}

func NewEtcdDiscoverer(cli *clientv3.Client) Discoverer {
	return &etcdDiscoverer{
		cli: cli,
	}
}

func (d *etcdDiscoverer) Get(ctx context.Context, serviceDiscoverKey string) ([]*Endpoint, error) {
	serviceRegisterKey, group := parseServiceGroup(serviceDiscoverKey)

	resp, err := d.cli.Get(ctx, serviceRegisterKey, clientv3.WithPrefix())
	if err != nil {
		log.Printf("get service %s failure: %v\n", serviceDiscoverKey, err)
		return nil, err
	}

	if resp.Header != nil {
		d.rev = resp.Header.GetRevision()
	}

	endpoints := etcdInstancesToEndpoints(resp.Kvs)
	return FilterOrDefault(endpoints, group), nil
}

func (d *etcdDiscoverer) Watch(ctx context.Context, serviceDiscoverKey string) ([]*Endpoint, Watcher, error) {
	endpoints, err := d.Get(ctx, serviceDiscoverKey)
	if err != nil {
		return nil, nil, err
	}

	w := newWatcher(d.cli, serviceDiscoverKey, endpoints, d.rev)
	return endpoints, w, err
}

func parseServiceGroup(serviceKeyPrefix string) (string, string) {
	colonIndex := strings.LastIndex(serviceKeyPrefix, ":")

	if colonIndex == -1 {
		return serviceKeyPrefix, DefaultGroup
	}

	basePath := serviceKeyPrefix[:colonIndex]
	feature := serviceKeyPrefix[colonIndex+1:]

	feature = strings.TrimSuffix(feature, "/")

	return basePath + "/", feature
}
