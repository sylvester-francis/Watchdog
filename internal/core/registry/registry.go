package registry

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Registry manages module lifecycle and dependency resolution.
// Modules are initialized in registration order and shut down in reverse order.
type Registry struct {
	mu      sync.RWMutex
	modules map[string]Module
	order   []string
	logger  *slog.Logger
}

// New creates a new module registry.
func New(logger *slog.Logger) *Registry {
	return &Registry{
		modules: make(map[string]Module),
		logger:  logger,
	}
}

// Register adds or replaces a module by name.
// If a module with the same name already exists, it is replaced
// but retains its position in the initialization order.
func (r *Registry) Register(m Module) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := m.Name()
	if _, exists := r.modules[name]; exists {
		r.modules[name] = m
		return
	}
	r.modules[name] = m
	r.order = append(r.order, name)
}

// Get returns a module by name.
func (r *Registry) Get(name string) (Module, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.modules[name]
	return m, ok
}

// MustGet returns a module by name or panics if not found.
// A panic indicates a startup misconfiguration.
func (r *Registry) MustGet(name string) Module {
	m, ok := r.Get(name)
	if !ok {
		panic(fmt.Sprintf("registry: module %q not registered", name))
	}
	return m
}

// InitAll initializes all modules in registration order.
func (r *Registry) InitAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, name := range r.order {
		m := r.modules[name]
		r.logger.Info("initializing module", slog.String("module", name))
		if err := m.Init(ctx); err != nil {
			return fmt.Errorf("init module %q: %w", name, err)
		}
	}
	return nil
}

// ShutdownAll shuts down all modules in reverse registration order.
// Returns the first error encountered but attempts to shut down all modules.
func (r *Registry) ShutdownAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var firstErr error
	for i := len(r.order) - 1; i >= 0; i-- {
		name := r.order[i]
		m := r.modules[name]
		r.logger.Info("shutting down module", slog.String("module", name))
		if err := m.Shutdown(ctx); err != nil {
			r.logger.Error("module shutdown failed",
				slog.String("module", name),
				slog.String("error", err.Error()),
			)
			if firstErr == nil {
				firstErr = fmt.Errorf("shutdown module %q: %w", name, err)
			}
		}
	}
	return firstErr
}

// HealthAll checks the health of all registered modules.
func (r *Registry) HealthAll(ctx context.Context) map[string]error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[string]error, len(r.modules))
	for _, name := range r.order {
		results[name] = r.modules[name].Health(ctx)
	}
	return results
}
