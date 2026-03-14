package services

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog-proto/protocol"
	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/core/realtime"
	"github.com/sylvester-francis/watchdog/internal/snmp"
)

// DiscoveryService orchestrates network discovery operations.
type DiscoveryService struct {
	discoveryRepo ports.DiscoveryRepository
	agentRepo     ports.AgentRepository
	monitorSvc    ports.MonitorService
	hub           *realtime.Hub
	logger        *slog.Logger
}

// NewDiscoveryService creates a new DiscoveryService.
func NewDiscoveryService(
	discoveryRepo ports.DiscoveryRepository,
	agentRepo ports.AgentRepository,
	monitorSvc ports.MonitorService,
	hub *realtime.Hub,
	logger *slog.Logger,
) *DiscoveryService {
	return &DiscoveryService{
		discoveryRepo: discoveryRepo,
		agentRepo:     agentRepo,
		monitorSvc:    monitorSvc,
		hub:           hub,
		logger:        logger,
	}
}

// StartScan validates input, creates a scan record, and dispatches to an agent.
func (s *DiscoveryService) StartScan(ctx context.Context, userID, agentID uuid.UUID, subnet, community, snmpVersion string) (*domain.DiscoveryScan, error) {
	// Validate CIDR
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR notation: %w", err)
	}
	ones, _ := ipNet.Mask.Size()
	if ones < 20 {
		return nil, fmt.Errorf("subnet too large: /%d (max /20 = 4096 hosts)", ones)
	}
	if !isPrivateNetwork(ipNet) {
		return nil, fmt.Errorf("only private network ranges are allowed (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)")
	}

	// Validate agent ownership
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil || agent == nil || agent.UserID != userID {
		return nil, fmt.Errorf("agent not found")
	}

	// Check if agent is connected
	if !s.hub.IsConnected(agentID) {
		return nil, fmt.Errorf("agent is not connected")
	}

	// Default SNMP version
	if snmpVersion == "" {
		snmpVersion = "2c"
	}
	if community == "" {
		community = "public"
	}

	// Create scan record
	scan := &domain.DiscoveryScan{
		ID:      uuid.New(),
		UserID:  userID,
		AgentID: agentID,
		Subnet:  subnet,
		Status:  domain.DiscoveryStatusPending,
	}
	if err := s.discoveryRepo.CreateScan(ctx, scan); err != nil {
		return nil, fmt.Errorf("failed to create scan: %w", err)
	}

	// Send discovery task to agent
	taskMsg := protocol.NewDiscoveryTaskMessage(scan.ID.String(), subnet, community, snmpVersion, 300)
	s.hub.SendToAgent(agentID, taskMsg)

	s.logger.Info("discovery scan started",
		slog.String("scan_id", scan.ID.String()),
		slog.String("subnet", subnet),
		slog.String("agent_id", agentID.String()),
	)

	return scan, nil
}

// ProcessResult handles a discovery result from an agent.
func (s *DiscoveryService) ProcessResult(ctx context.Context, result *protocol.DiscoveryResultPayload) error {
	scanID, err := uuid.Parse(result.TaskID)
	if err != nil {
		return fmt.Errorf("invalid scan ID: %w", err)
	}

	scan, err := s.discoveryRepo.GetScanByID(ctx, scanID)
	if err != nil || scan == nil {
		return fmt.Errorf("scan not found: %s", result.TaskID)
	}

	switch result.Status {
	case "running":
		now := time.Now()
		scan.Status = domain.DiscoveryStatusRunning
		if scan.StartedAt == nil {
			scan.StartedAt = &now
		}
		scan.HostCount = len(result.Devices)

	case "complete":
		now := time.Now()
		scan.Status = domain.DiscoveryStatusComplete
		scan.CompletedAt = &now
		scan.HostCount = len(result.Devices)

	case "error":
		now := time.Now()
		scan.Status = domain.DiscoveryStatusError
		scan.CompletedAt = &now
		scan.ErrorMessage = result.Error
	}

	if err := s.discoveryRepo.UpdateScan(ctx, scan); err != nil {
		s.logger.Error("failed to update scan", slog.String("error", err.Error()))
	}

	// Store discovered devices
	for _, d := range result.Devices {
		// Match template by sysObjectID
		templateID := d.TemplateID
		if templateID == "" && d.SysObjectID != "" {
			if t := snmp.MatchBySysObjectID(d.SysObjectID); t != nil {
				templateID = t.ID
			}
		}

		device := &domain.DiscoveredDevice{
			ScanID:              scanID,
			UserID:              scan.UserID,
			IP:                  d.IP,
			Hostname:            d.Hostname,
			SysDescr:            d.SysDescr,
			SysObjectID:         d.SysObjectID,
			SysName:             d.SysName,
			SNMPReachable:       d.SNMPReachable,
			PingReachable:       d.PingReachable,
			SuggestedTemplateID: templateID,
		}
		if err := s.discoveryRepo.CreateDevice(ctx, device); err != nil {
			s.logger.Error("failed to store discovered device",
				slog.String("ip", d.IP),
				slog.String("error", err.Error()),
			)
		}
	}

	return nil
}

// GetScan returns a scan with its discovered devices.
func (s *DiscoveryService) GetScan(ctx context.Context, scanID uuid.UUID) (*domain.DiscoveryScan, []*domain.DiscoveredDevice, error) {
	scan, err := s.discoveryRepo.GetScanByID(ctx, scanID)
	if err != nil {
		return nil, nil, err
	}
	if scan == nil {
		return nil, nil, nil
	}
	devices, err := s.discoveryRepo.GetDevicesByScanID(ctx, scanID)
	if err != nil {
		return scan, nil, err
	}
	return scan, devices, nil
}

// ListScans returns all scans for a user.
func (s *DiscoveryService) ListScans(ctx context.Context, userID uuid.UUID) ([]*domain.DiscoveryScan, error) {
	return s.discoveryRepo.GetScansByUserID(ctx, userID)
}

func isPrivateNetwork(ipNet *net.IPNet) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7",
	}
	for _, cidr := range privateRanges {
		_, privateNet, _ := net.ParseCIDR(cidr)
		if privateNet.Contains(ipNet.IP) {
			return true
		}
	}
	return false
}
