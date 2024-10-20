package servicegovernance

import "context"

type (
	Discoverer interface {
		Get(ctx context.Context, serviceKeyPrefix string) ([]*Endpoint, error)
		//Watch(ctx context.Context, serviceKeyPrefix string) ([]*Endpoint, Watcher, error)
	}

	Watcher interface {
		Next() ([]*Endpoint, error)
		Close() error
	}

	DiscoverGetNamingService func(etcdEndpoints []string, envName, serviceName, group string) ([]*Endpoint, error)
)
