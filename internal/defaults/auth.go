package defaults

import (
	"context"

	"github.com/sylvester-francis/watchdog/internal/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/registry"
)

const moduleAuthProvider = "auth_provider"

var (
	_ registry.Module   = (*authModule)(nil)
	_ ports.AuthProvider = (*authModule)(nil)
)

// authModule wraps the existing AuthService which already satisfies AuthProvider.
type authModule struct {
	ports.AuthProvider
}

func newAuthModule(auth ports.AuthProvider) *authModule {
	return &authModule{AuthProvider: auth}
}

func (m *authModule) Name() string                    { return moduleAuthProvider }
func (m *authModule) Init(_ context.Context) error    { return nil }
func (m *authModule) Health(_ context.Context) error   { return nil }
func (m *authModule) Shutdown(_ context.Context) error { return nil }
