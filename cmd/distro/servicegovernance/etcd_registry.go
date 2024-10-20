package servicegovernance

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
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
	signal.Notify(sigCh, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	sig := <-sigCh
	if er := registrar.Deregister(appCtx); er != nil {
		log.Printf("deregister failure: %v\n", er)
	}
	fmt.Printf("[%s] registrar received signal: %v, exiting...\n", serviceName, sig)
	return nil
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

	log.Printf("register success\n\tinfo: (%s - %s), \n\tresponse: %v \n\tlease: %v\n", instanceKey, value, pr, lgr)

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
				log.Printf("etcd keepalive channel closed, attempting to retry registration...\n")
				go r.retry()
				return
			} else if resp == nil {
				log.Printf("etcd keepalive response is nil, retrying registration...\n")
				go r.retry()
				return
			} else {
				log.Printf("keepalive successful: %v\n", resp)
			}
		case <-r.ctx.Done():
			log.Printf("keepalive context done, exiting keepalive loop\n")
			return
		}
	}
}

func (r *etcdRegistry) retry() {
	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := r.Register(context.Background(), r.instance)
			if err == nil {
				log.Printf("etcd register retry success\n")
				return
			}
			log.Printf("retry register error: %v\n", err)
		case <-r.ctx.Done():
			log.Printf("retry context done, exiting retry loop\n")
			return
		}
	}
}
