package registry

import "context"

// Module is a pluggable component with lifecycle management.
// Modules are registered at startup and initialized in registration order.
type Module interface {
	// Name returns a unique identifier for this module.
	Name() string

	// Init is called at startup to initialize the module.
	Init(ctx context.Context) error

	// Health returns nil if the module is healthy.
	Health(ctx context.Context) error

	// Shutdown is called during graceful shutdown.
	Shutdown(ctx context.Context) error
}
