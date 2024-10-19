package distro

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	ttl           int64 = 10
	retryInterval       = 5 * time.Second
	key                 = "/notes-etcd/keepalive"
	val                 = `{"address":"127.0.0.1:8848","group":"default"}`
)

func register(client *clientv3.Client, parentCtx context.Context) error {
	ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
	defer cancel()

	lgr, err := client.Grant(ctx, ttl)
	if err != nil {
		log.Println(err)
		return err
	}

	pr, err := client.Put(ctx, key, val, clientv3.WithLease(lgr.ID))
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println("PutResponse:", pr)

	lkaCh, err := client.KeepAlive(context.TODO(), lgr.ID)
	if err != nil {
		log.Println(err)
		return err
	}

	keepalive(client, parentCtx, lkaCh)

	return nil
}

func keepalive(client *clientv3.Client, parentCtx context.Context, lkaCh <-chan *clientv3.LeaseKeepAliveResponse) {
	done := false
	for !done {
		select {
		case r, ok := <-lkaCh:
			if !ok || r == nil {
				fmt.Println("Keepalive closed or lost heartbeat", r, ok)
				retry(client, parentCtx)
			} else {
				fmt.Println("Keepalive ping", r, ok)
			}
		case v, ok := <-parentCtx.Done():
			fmt.Println("Context done", v, ok)
			done = true
		}
	}
}

func retry(client *clientv3.Client, parentCtx context.Context) {
	retryCh := time.Tick(retryInterval)
loop:
	for {
		select {
		case t, ok := <-retryCh:
			err := register(client, parentCtx)
			if err != nil {
				fmt.Println("RETRY FAILURE TRY AGAIN:", t, ok)
			} else {
				fmt.Println("RETRY SUCCESS:", t, ok)
				break loop
			}
		case v, ok := <-parentCtx.Done():
			fmt.Println("Context done", v, ok)
			break loop
		}
	}
}
