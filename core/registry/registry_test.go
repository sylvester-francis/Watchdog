package registry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockModule is a test module that records lifecycle calls.
type mockModule struct {
	name        string
	initCalled  bool
	initErr     error
	shutdownErr error
	healthErr   error
	order       *[]string // shared slice to track call order
}

func (m *mockModule) Name() string { return m.name }

func (m *mockModule) Init(_ context.Context) error {
	m.initCalled = true
	if m.order != nil {
		*m.order = append(*m.order, fmt.Sprintf("init:%s", m.name))
	}
	return m.initErr
}

func (m *mockModule) Health(_ context.Context) error {
	return m.healthErr
}

func (m *mockModule) Shutdown(_ context.Context) error {
	if m.order != nil {
		*m.order = append(*m.order, fmt.Sprintf("shutdown:%s", m.name))
	}
	return m.shutdownErr
}

func newTestRegistry() *Registry {
	return New(slog.Default())
}

func TestRegisterAndGet(t *testing.T) {
	reg := newTestRegistry()
	m := &mockModule{name: "test"}

	reg.Register(m)

	got, ok := reg.Get("test")
	assert.True(t, ok)
	assert.Equal(t, m, got)

	_, ok = reg.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegisterOverride(t *testing.T) {
	reg := newTestRegistry()
	m1 := &mockModule{name: "test"}
	m2 := &mockModule{name: "test"}

	reg.Register(m1)
	reg.Register(m2)

	got, ok := reg.Get("test")
	assert.True(t, ok)
	assert.Equal(t, m2, got, "second registration should replace first")
	assert.Len(t, reg.order, 1, "override should not add duplicate order entry")
}

func TestMustGet(t *testing.T) {
	reg := newTestRegistry()
	m := &mockModule{name: "test"}
	reg.Register(m)

	got := reg.MustGet("test")
	assert.Equal(t, m, got)
}

func TestMustGetPanics(t *testing.T) {
	reg := newTestRegistry()

	assert.PanicsWithValue(t,
		`registry: module "missing" not registered`,
		func() { reg.MustGet("missing") },
	)
}

func TestInitAllOrder(t *testing.T) {
	reg := newTestRegistry()
	order := &[]string{}

	reg.Register(&mockModule{name: "a", order: order})
	reg.Register(&mockModule{name: "b", order: order})
	reg.Register(&mockModule{name: "c", order: order})

	err := reg.InitAll(context.Background())
	require.NoError(t, err)

	assert.Equal(t, []string{"init:a", "init:b", "init:c"}, *order)
}

func TestInitAllStopsOnError(t *testing.T) {
	reg := newTestRegistry()
	order := &[]string{}
	initErr := errors.New("init failed")

	reg.Register(&mockModule{name: "a", order: order})
	reg.Register(&mockModule{name: "b", order: order, initErr: initErr})
	reg.Register(&mockModule{name: "c", order: order})

	err := reg.InitAll(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "init module \"b\"")
	assert.Equal(t, []string{"init:a", "init:b"}, *order, "should stop after failing module")
}

func TestShutdownAllReverseOrder(t *testing.T) {
	reg := newTestRegistry()
	order := &[]string{}

	reg.Register(&mockModule{name: "a", order: order})
	reg.Register(&mockModule{name: "b", order: order})
	reg.Register(&mockModule{name: "c", order: order})

	err := reg.ShutdownAll(context.Background())
	require.NoError(t, err)

	assert.Equal(t, []string{"shutdown:c", "shutdown:b", "shutdown:a"}, *order)
}

func TestShutdownAllContinuesOnError(t *testing.T) {
	reg := newTestRegistry()
	order := &[]string{}
	shutdownErr := errors.New("shutdown failed")

	reg.Register(&mockModule{name: "a", order: order})
	reg.Register(&mockModule{name: "b", order: order, shutdownErr: shutdownErr})
	reg.Register(&mockModule{name: "c", order: order})

	err := reg.ShutdownAll(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shutdown module \"b\"")
	assert.Equal(t, []string{"shutdown:c", "shutdown:b", "shutdown:a"}, *order,
		"should attempt all shutdowns even after error")
}

func TestHealthAll(t *testing.T) {
	reg := newTestRegistry()
	healthErr := errors.New("unhealthy")

	reg.Register(&mockModule{name: "healthy"})
	reg.Register(&mockModule{name: "unhealthy", healthErr: healthErr})

	results := reg.HealthAll(context.Background())

	assert.NoError(t, results["healthy"])
	assert.Equal(t, healthErr, results["unhealthy"])
	assert.Len(t, results, 2)
}
