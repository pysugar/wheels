package client

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/singleflight"
)

type connPool struct {
	rw      sync.RWMutex
	conns   map[string]*clientConn
	g       *singleflight.Group
	verbose uint32
}

func newConnPool() *connPool {
	return &connPool{
		conns: make(map[string]*clientConn),
		g:     new(singleflight.Group),
	}
}

func (p *connPool) Close() (err error) {
	p.rw.Lock()
	defer p.rw.Unlock()
	for _, conn := range p.conns {
		err = conn.Close()
	}
	p.conns = nil
	return err
}

func (cp *connPool) getConn(ctx context.Context, target string, opts ...DialOption) (*clientConn, error) {
	cp.rw.RLock()
	cc := cp.conns[target]
	cp.rw.RUnlock()

	verboseFromContext := VerboseFromContext(ctx)
	if verboseFromContext {
		atomic.CompareAndSwapUint32(&cp.verbose, 0, 1)
	} else {
		atomic.CompareAndSwapUint32(&cp.verbose, 1, 0)
	}

	if cc != nil {
		if cc.isValid(ctx) {
			cp.printf("[connPool] get conn success from cache, target: %s", target)
			return cc, nil
		}
		fmt.Printf("[connPool] get conn fail from cache, target: %s\n", target)
		cc.Close()
	}

	v, err, shared := cp.g.Do(target, func() (interface{}, error) {
		if verboseFromContext {
			opts = append(opts, withVerbose())
		}

		cp.printf("[connPool] connect to target: %s", target)
		return dialContext(ctx, target, opts...)
	})

	if err != nil {
		cp.printf("[connPool] get conn err, target: %s, shared: %v, err: %v", target, shared, err)
		return nil, err
	}

	cc = v.(*clientConn)
	cp.printf("[connPool] get conn-%05d success, target: %s, shared: %v", cc.id, target, shared)

	cp.rw.Lock()
	defer cp.rw.Unlock()
	cp.conns[target] = cc

	return cc, nil
}

func (cp *connPool) printf(format string, v ...interface{}) {
	if atomic.LoadUint32(&cp.verbose) == 1 {
		log.Printf(format, v...)
	}
}
