package servicegovernance

import (
	"log"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func newEtcdClient(endpoints []string) (*clientv3.Client, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		return nil, err
	}
	return client, err
}

func etcdInstancesToEndpoints(kvs []*mvccpb.KeyValue) []*Endpoint {
	endpoints := make([]*Endpoint, 0, len(kvs))
	for _, kv := range kvs {
		ep := new(Endpoint)
		if err := ep.Decode(kv.Value); err != nil {
			log.Printf("invalid endpoint info (), error: %v\n", string(kv.Value), err)
			continue
		}
		endpoints = append(endpoints, ep)
	}
	return endpoints
}
