package client

import (
	"context"
	"golang.org/x/sync/singleflight"
	"log"
	"sync"
)

type connPool struct {
	sync.RWMutex
	conns map[string]*clientConn
	g     *singleflight.Group
}

func newConnPool() *connPool {
	return &connPool{
		conns: make(map[string]*clientConn),
		g:     new(singleflight.Group),
	}
}

func (p *connPool) Close() (err error) {
	p.Lock()
	defer p.Unlock()
	for _, conn := range p.conns {
		err = conn.close()
	}
	p.conns = nil
	return err
}

func (cp *connPool) getConn(ctx context.Context, target string, opts ...DialOption) (*clientConn, error) {
	cp.RLock()
	cc := cp.conns[target]
	cp.RUnlock()

	if cc != nil && cc.isValid() {
		return cc, nil
	}

	v, err, shared := cp.g.Do(target, func() (interface{}, error) {
		return dialContext(ctx, target, opts...)
	})

	log.Printf("get conn success, target: %s, shard: %v", target, shared)

	if err != nil {
		return nil, err
	}

	cc = v.(*clientConn)

	cp.Lock()
	defer cp.Unlock()
	cp.conns[target] = cc

	return cc, nil
}
