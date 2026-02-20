package defaults

import (
	"context"

	"github.com/sylvester-francis/watchdog/internal/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/registry"
)

const moduleStorageBackend = "storage_backend"

var (
	_ registry.Module    = (*storageModule)(nil)
	_ ports.StorageBackend = (*storageModule)(nil)
)

// storageModule wraps the existing database connection.
// WithTenantScope returns ctx unchanged in the default implementation.
type storageModule struct {
	db ports.Transactor
}

func newStorageModule(db ports.Transactor) *storageModule {
	return &storageModule{db: db}
}

func (m *storageModule) Name() string                    { return moduleStorageBackend }
func (m *storageModule) Init(_ context.Context) error    { return nil }
func (m *storageModule) Health(_ context.Context) error   { return nil }
func (m *storageModule) Shutdown(_ context.Context) error { return nil }

func (m *storageModule) WithTenantScope(ctx context.Context, _ string) context.Context {
	return ctx
}
