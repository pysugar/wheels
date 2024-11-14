package client

import (
	"golang.org/x/sync/singleflight"
	"net/url"
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

func (cp *connPool) getConn(url *url.URL) (*clientConn, error) {
	serviceKey := url.Host
	cp.RLock()
	cc := cp.conns[serviceKey]
	cp.RUnlock()

	if cc != nil && cc.isValid() {
		return cc, nil
	}

	v, err, _ := cp.g.Do(serviceKey, func() (interface{}, error) {
		return newClientConn(url)
	})

	if err != nil {
		return nil, err
	}

	cc = v.(*clientConn)

	cp.Lock()
	defer cp.Unlock()
	cp.conns[serviceKey] = cc

	return cc, nil
}
