package defaults

import (
	"context"
	"os"

	"github.com/sylvester-francis/watchdog/internal/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/registry"
)

const moduleClusterCoordinator = "cluster_coordinator"

var (
	_ registry.Module          = (*clusterModule)(nil)
	_ ports.ClusterCoordinator = (*clusterModule)(nil)
)

// clusterModule is a standalone cluster coordinator.
// In standalone mode, this node is always the leader.
type clusterModule struct {
	nodeID string
}

func newClusterModule() *clusterModule {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "hub-standalone"
	}
	return &clusterModule{nodeID: hostname}
}

func (m *clusterModule) Name() string                    { return moduleClusterCoordinator }
func (m *clusterModule) Init(_ context.Context) error    { return nil }
func (m *clusterModule) Health(_ context.Context) error   { return nil }
func (m *clusterModule) Shutdown(_ context.Context) error { return nil }
func (m *clusterModule) IsLeader() bool                   { return true }
func (m *clusterModule) NodeID() string                   { return m.nodeID }
