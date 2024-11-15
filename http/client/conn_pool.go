package client

import (
	"context"
	"log"
	"sync"

	"golang.org/x/sync/singleflight"
)

type connPool struct {
	sync.RWMutex
	conns   map[string]*clientConn
	g       *singleflight.Group
	verbose bool
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
		err = conn.Close()
	}
	p.conns = nil
	return err
}

func (cp *connPool) getConn(ctx context.Context, target string, opts ...DialOption) (*clientConn, error) {
	cp.RLock()
	cc := cp.conns[target]
	cp.RUnlock()

	if cc != nil {
		if cc.isValid() {
			cp.printf("[connPool] get conn success from cache, target: %s", target)
			return cc, nil
		}
		cc.Close()
	}

	v, err, shared := cp.g.Do(target, func() (interface{}, error) {
		if cp.verbose {
			// opts = append(opts, WithVerbose())
		}

		cp.printf("[connPool] connect to target: %s", target)
		return dialContext(ctx, target, opts...)
	})

	if err != nil {
		return nil, err
	}

	cc = v.(*clientConn)
	cp.printf("[connPool] get conn-%05d success, target: %s, shared: %v", cc.id, target, shared)
	
	cp.Lock()
	defer cp.Unlock()
	cp.conns[target] = cc

	return cc, nil
}

func (cp *connPool) printf(format string, v ...interface{}) {
	if cp.verbose {
		log.Printf(format, v...)
	}
}
