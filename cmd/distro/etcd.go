package distro

import (
	"context"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	ttl           int64 = 10
	retryInterval       = 5 * time.Second
)

type (
	etcdRegistry struct {
		client      *clientv3.Client
		ctx         context.Context
		stop        context.CancelFunc
		instance    *Instance
		lease       clientv3.LeaseID
		keepaliveCh <-chan *clientv3.LeaseKeepAliveResponse
	}
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

func NewEtcdRegistry(cli *clientv3.Client) Registrar {
	ctx, cancel := context.WithCancel(context.Background())
	return &etcdRegistry{
		client: cli,
		ctx:    ctx,
		stop:   cancel,
	}
}

func (r *etcdRegistry) Register(appCtx context.Context, instance *Instance) error {
	ctx, cancel := context.WithTimeout(appCtx, 3*time.Second)
	defer cancel()

	lgr, err := r.client.Grant(ctx, ttl)
	if err != nil {
		log.Printf("register grant fail, err: %v\n", err)
		return err
	}

	instanceKey := instance.Key()
	value := instance.Endpoint.Encode()

	pr, err := r.client.Put(ctx, instanceKey, value, clientv3.WithLease(lgr.ID))
	if err != nil {
		log.Printf("register put fail, err: %v\n", err)
		return err
	}

	r.keepaliveCh, err = r.client.KeepAlive(context.Background(), lgr.ID)
	if err != nil {
		log.Printf("register keepalive fail, err: %v\n", err)
		return err
	}
	r.lease = lgr.ID
	r.instance = instance

	log.Printf("register success, \n\tresponse: %v \n\tlease: %v\n", pr, lgr)

	go r.keepalive()
	return nil
}

func (r *etcdRegistry) Deregister(ctx context.Context) error {
	r.stop()

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	lrr, err := r.client.Revoke(ctx, r.lease)
	log.Printf("deregister with revoked (%v, %v)", lrr, err)
	return err
}

func (r *etcdRegistry) keepalive() {
	for {
		select {
		case resp, ok := <-r.keepaliveCh:
			if !ok {
				log.Printf("etcd keepalive error with resp: %v\n", resp)
				go r.retry()
				return
			} else {
				log.Printf("keepalive %v(%v)\n", resp, ok)
			}
		case <-r.ctx.Done():
			log.Printf("keepalive done\n")
			return
		}
	}
}

func (r *etcdRegistry) retry() {
	ticker := time.Tick(retryInterval)
	for {
		select {
		case <-ticker:
			err := r.Register(context.Background(), r.instance)
			if err == nil {
				log.Printf("etcd register success\n")
				return
			}
			log.Printf("retry register error: %v\n", err)
		case <-r.ctx.Done():
			log.Printf("retry while context done\n")
			return
		}
	}
}
