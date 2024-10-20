package servicegovernance

import "context"

type (
	Registrar interface {
		Register(ctx context.Context, instance *Instance) error
		Deregister(ctx context.Context) error
	}

	RegisterNamingService func(endpoints []string, env, service, address string) error
)
