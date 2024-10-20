package servicegovernance

import clientv3 "go.etcd.io/etcd/client/v3"

func newEtcdClient(endpoints []string) (*clientv3.Client, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		return nil, err
	}
	return client, err
}
