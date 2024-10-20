package servicegovernance

import (
	"context"
)

type (
	Discoverer interface {
		Get(ctx context.Context, serviceKeyPrefix string) ([]*Endpoint, error)
		Watch(ctx context.Context, serviceKeyPrefix string) ([]*Endpoint, Watcher, error)
	}

	Watcher interface {
		Service() string
		Next() ([]*Endpoint, error)
		Close() error
	}

	DiscoverNamingService func(etcdEndpoints []string, envName, serviceName, group string, watchEnabled bool) ([]*Endpoint, error)
)

func FilterOrDefault(endpoints []*Endpoint, group string) []*Endpoint {
	groupEndpoints := make([]*Endpoint, 0)
	defaultEndpoints := make([]*Endpoint, 0)
	for _, ep := range endpoints {
		if ep.Group == DefaultGroup || ep.Group == "" {
			defaultEndpoints = append(defaultEndpoints, ep)
		} else if ep.Group == group {
			groupEndpoints = append(groupEndpoints, ep)
		}
	}
	if len(groupEndpoints) != 0 {
		return groupEndpoints
	}
	return defaultEndpoints
}
