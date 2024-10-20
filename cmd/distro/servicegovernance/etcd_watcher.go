package servicegovernance

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type etcdWatcher struct {
	cli                *clientv3.Client
	serviceRegisterKey string
	group              string
	endpoints          map[string]*Endpoint
	wch                clientv3.WatchChan
	ctx                context.Context
	stop               context.CancelFunc
}

func newWatcher(cli *clientv3.Client, serviceDiscoverKey string, endpoints []*Endpoint, rev int64) Watcher {
	serviceRegisterKey, group := parseServiceGroup(serviceDiscoverKey)

	ctx, cancel := context.WithCancel(context.Background())
	w := &etcdWatcher{
		cli:                cli,
		serviceRegisterKey: serviceRegisterKey,
		group:              group,
		endpoints:          make(map[string]*Endpoint),
		wch:                make(chan clientv3.WatchResponse, 1),
		ctx:                ctx,
		stop:               cancel,
	}

	for _, ep := range endpoints {
		w.endpoints[ep.Address] = ep
	}

	if rev > 0 {
		w.wch = w.cli.Watch(ctx, serviceRegisterKey, clientv3.WithPrefix(), clientv3.WithRev(rev+1))
	} else {
		w.wch = w.cli.Watch(ctx, serviceRegisterKey, clientv3.WithPrefix())
	}

	return w
}

func (w *etcdWatcher) Service() string {
	return w.serviceRegisterKey
}

func (w *etcdWatcher) Next() ([]*Endpoint, error) {
	for {
		select {
		case resp, ok := <-w.wch:
			if !ok {
				log.Printf("etcd watch %s encounter channel closed\n", w.serviceRegisterKey)
				w.wch = w.cli.Watch(w.ctx, w.serviceRegisterKey, clientv3.WithPrefix())
				return nil, errors.New("channel is closed")
			}
			log.Printf("%s %v next: %v\n", w.serviceRegisterKey, ok, resp.Header)

			changed := false
			for _, ev := range resp.Events {
				switch ev.Type {
				case mvccpb.PUT:
					endpoint := new(Endpoint)
					err := endpoint.Decode(ev.Kv.Value)
					if err != nil {
						log.Printf("decode %s value error: %v\n", string(ev.Kv.Value), err)
						continue
					}

					if !strings.HasPrefix(string(ev.Kv.Key), w.serviceRegisterKey) {
						log.Printf("[ERROR] %s etcd next, put another service instance %s, skip", w.serviceRegisterKey, string(ev.Kv.Key))
						continue
					}

					w.endpoints[endpoint.Address] = endpoint
					changed = true
				case mvccpb.DELETE:
					address, err := extractAddressFromInstanceKey(string(ev.Kv.Key))
					if err != nil {
						log.Printf("extract %s address error: %v\n", string(ev.Kv.Key), err)
						continue
					}

					if !strings.HasPrefix(string(ev.Kv.Key), w.serviceRegisterKey) {
						log.Printf("[ERROR] %s etcd next, delete another service instance %s, skip", w.serviceRegisterKey, string(ev.Kv.Key))
						continue
					}

					delete(w.endpoints, address)
					changed = true
				}
			}

			if changed {
				return w.getEndpoints(), nil
			}
		case <-w.ctx.Done():
			log.Printf("etcd watcher context done\n")
			return nil, w.ctx.Err()
		}
	}
}

func (w *etcdWatcher) Close() error {
	w.stop()
	return nil
}

func (w *etcdWatcher) getEndpoints() []*Endpoint {
	var endpoints []*Endpoint
	for _, ep := range w.endpoints {
		endpoints = append(endpoints, ep)
	}
	return FilterOrDefault(endpoints, w.group)
}

func extractAddressFromInstanceKey(key string) (string, error) {
	seg := strings.Split(key, "/")
	if len(seg) < 3 {
		return "", fmt.Errorf("invalid key %s", key)
	} else {
		return seg[len(seg)-1], nil
	}
}
