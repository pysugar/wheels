package servicegovernance

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type (
	etcdDiscoverer struct {
		cli *clientv3.Client
		rev int64
	}
)

func DiscoverGetETCD(etcdEndpoints []string, envName, serviceName, group string) ([]*Endpoint, error) {
	client, err := newEtcdClient(etcdEndpoints)
	if err != nil {
		log.Printf("unexpected err: %v\n", err)
		return nil, err
	}

	serviceWithEnv := fmt.Sprintf("/%s/%s", envName, serviceName)
	serviceKeyPrefix := fmt.Sprintf("%s/", serviceWithEnv)
	if group != DefaultGroup && group != "" {
		serviceKeyPrefix = fmt.Sprintf("%s:%s/", serviceWithEnv, group)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	discoverer := NewEtcdDiscoverer(client)
	endpoints, err := discoverer.Get(ctx, serviceKeyPrefix)
	if err != nil {
		log.Printf("unexpected err: %v\n", err)
		return nil, err
	}
	return endpoints, nil
}

func NewEtcdDiscoverer(cli *clientv3.Client) Discoverer {
	return &etcdDiscoverer{
		cli: cli,
	}
}

func (d *etcdDiscoverer) Get(ctx context.Context, serviceKeyPrefix string) ([]*Endpoint, error) {
	serviceDiscoveryKey, feature := parseServiceGroup(serviceKeyPrefix)

	resp, err := d.cli.Get(ctx, serviceDiscoveryKey, clientv3.WithPrefix())
	if err != nil {
		log.Printf("get service %s failure: %v\n", serviceKeyPrefix, err)
		return nil, err
	}

	if resp.Header != nil {
		d.rev = resp.Header.GetRevision()
	}

	endpoints := etcdInstancesToEndpoints(resp.Kvs)
	targetEndpoints := make([]*Endpoint, 0)
	log.Printf("discoverer get %s with fature %s\n", serviceDiscoveryKey, feature)
	for _, ep := range endpoints {
		log.Printf("\tendpoint: %v\n", ep)
		if ep.Group == feature {
			targetEndpoints = append(targetEndpoints, ep)
		}
	}
	return targetEndpoints, nil
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
