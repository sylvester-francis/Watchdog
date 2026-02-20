package defaults

import (
	"context"

	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/core/registry"
)

const moduleStatusPageProvider = "status_page_provider"

var (
	_ registry.Module          = (*statusModule)(nil)
	_ ports.StatusPageProvider = (*statusModule)(nil)
)

// statusModule wraps the existing StatusPageRepository.
type statusModule struct {
	ports.StatusPageRepository
}

func newStatusModule(repo ports.StatusPageRepository) *statusModule {
	return &statusModule{StatusPageRepository: repo}
}

func (m *statusModule) Name() string                    { return moduleStatusPageProvider }
func (m *statusModule) Init(_ context.Context) error    { return nil }
func (m *statusModule) Health(_ context.Context) error   { return nil }
func (m *statusModule) Shutdown(_ context.Context) error { return nil }
