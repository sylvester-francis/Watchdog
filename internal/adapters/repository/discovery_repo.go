package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// DiscoveryRepository implements ports.DiscoveryRepository using PostgreSQL.
type DiscoveryRepository struct {
	db *DB
}

// NewDiscoveryRepository creates a new DiscoveryRepository.
func NewDiscoveryRepository(db *DB) *DiscoveryRepository {
	return &DiscoveryRepository{db: db}
}

// CreateScan inserts a new discovery scan.
func (r *DiscoveryRepository) CreateScan(ctx context.Context, scan *domain.DiscoveryScan) error {
	q := r.db.Querier(ctx)
	if scan.ID == uuid.Nil {
		scan.ID = uuid.New()
	}
	_, err := q.Exec(ctx,
		`INSERT INTO discovery_scans (id, user_id, agent_id, subnet, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())`,
		scan.ID, scan.UserID, scan.AgentID, scan.Subnet, scan.Status,
	)
	return err
}

// GetScanByID returns a scan by ID.
func (r *DiscoveryRepository) GetScanByID(ctx context.Context, id uuid.UUID) (*domain.DiscoveryScan, error) {
	q := r.db.Querier(ctx)
	row := q.QueryRow(ctx,
		`SELECT id, user_id, agent_id, subnet, status, started_at, completed_at, host_count, error_message, created_at
		 FROM discovery_scans WHERE id = $1`, id,
	)
	return scanDiscoveryScan(row)
}

// GetScansByUserID returns all scans for a user, newest first.
func (r *DiscoveryRepository) GetScansByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.DiscoveryScan, error) {
	q := r.db.Querier(ctx)
	rows, err := q.Query(ctx,
		`SELECT id, user_id, agent_id, subnet, status, started_at, completed_at, host_count, error_message, created_at
		 FROM discovery_scans WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scans []*domain.DiscoveryScan
	for rows.Next() {
		s, err := scanDiscoveryScan(rows)
		if err != nil {
			return nil, err
		}
		scans = append(scans, s)
	}
	return scans, rows.Err()
}

// GetActiveScansByAgentID returns pending/running scans for a given agent.
func (r *DiscoveryRepository) GetActiveScansByAgentID(ctx context.Context, agentID uuid.UUID) ([]*domain.DiscoveryScan, error) {
	q := r.db.Querier(ctx)
	rows, err := q.Query(ctx,
		`SELECT id, user_id, agent_id, subnet, status, started_at, completed_at, host_count, error_message, created_at
		 FROM discovery_scans WHERE agent_id = $1 AND status IN ('pending', 'running') ORDER BY created_at DESC`, agentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scans []*domain.DiscoveryScan
	for rows.Next() {
		s, err := scanDiscoveryScan(rows)
		if err != nil {
			return nil, err
		}
		scans = append(scans, s)
	}
	return scans, rows.Err()
}

// UpdateScan updates scan fields.
func (r *DiscoveryRepository) UpdateScan(ctx context.Context, scan *domain.DiscoveryScan) error {
	q := r.db.Querier(ctx)
	_, err := q.Exec(ctx,
		`UPDATE discovery_scans SET status = $2, started_at = $3, completed_at = $4, host_count = $5, error_message = $6
		 WHERE id = $1`,
		scan.ID, scan.Status, scan.StartedAt, scan.CompletedAt, scan.HostCount, scan.ErrorMessage,
	)
	return err
}

// CreateDevice inserts a discovered device.
func (r *DiscoveryRepository) CreateDevice(ctx context.Context, device *domain.DiscoveredDevice) error {
	q := r.db.Querier(ctx)
	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}
	_, err := q.Exec(ctx,
		`INSERT INTO discovered_devices (id, scan_id, user_id, ip, hostname, sys_descr, sys_object_id, sys_name, snmp_reachable, ping_reachable, suggested_template_id, discovered_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())`,
		device.ID, device.ScanID, device.UserID, device.IP, device.Hostname,
		device.SysDescr, device.SysObjectID, device.SysName,
		device.SNMPReachable, device.PingReachable, device.SuggestedTemplateID,
	)
	return err
}

// GetDevicesByScanID returns all devices from a scan.
func (r *DiscoveryRepository) GetDevicesByScanID(ctx context.Context, scanID uuid.UUID) ([]*domain.DiscoveredDevice, error) {
	q := r.db.Querier(ctx)
	rows, err := q.Query(ctx,
		`SELECT id, scan_id, user_id, ip, hostname, sys_descr, sys_object_id, sys_name, snmp_reachable, ping_reachable, suggested_template_id, monitor_created, discovered_at
		 FROM discovered_devices WHERE scan_id = $1 ORDER BY ip`, scanID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*domain.DiscoveredDevice
	for rows.Next() {
		d, err := scanDiscoveredDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

// GetDeviceByID returns a single discovered device.
func (r *DiscoveryRepository) GetDeviceByID(ctx context.Context, id uuid.UUID) (*domain.DiscoveredDevice, error) {
	q := r.db.Querier(ctx)
	row := q.QueryRow(ctx,
		`SELECT id, scan_id, user_id, ip, hostname, sys_descr, sys_object_id, sys_name, snmp_reachable, ping_reachable, suggested_template_id, monitor_created, discovered_at
		 FROM discovered_devices WHERE id = $1`, id,
	)
	return scanDiscoveredDevice(row)
}

// MarkDeviceMonitorCreated sets monitor_created = true for a device.
func (r *DiscoveryRepository) MarkDeviceMonitorCreated(ctx context.Context, deviceID uuid.UUID) error {
	q := r.db.Querier(ctx)
	_, err := q.Exec(ctx,
		`UPDATE discovered_devices SET monitor_created = true WHERE id = $1`, deviceID,
	)
	return err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanDiscoveryScan(s scannable) (*domain.DiscoveryScan, error) {
	scan := &domain.DiscoveryScan{}
	var errMsg *string
	err := s.Scan(
		&scan.ID, &scan.UserID, &scan.AgentID, &scan.Subnet, &scan.Status,
		&scan.StartedAt, &scan.CompletedAt, &scan.HostCount, &errMsg, &scan.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if errMsg != nil {
		scan.ErrorMessage = *errMsg
	}
	return scan, nil
}

func scanDiscoveredDevice(s scannable) (*domain.DiscoveredDevice, error) {
	d := &domain.DiscoveredDevice{}
	var hostname, sysDescr, sysObjectID, sysName, templateID *string
	err := s.Scan(
		&d.ID, &d.ScanID, &d.UserID, &d.IP, &hostname,
		&sysDescr, &sysObjectID, &sysName,
		&d.SNMPReachable, &d.PingReachable, &templateID, &d.MonitorCreated, &d.DiscoveredAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if hostname != nil {
		d.Hostname = *hostname
	}
	if sysDescr != nil {
		d.SysDescr = *sysDescr
	}
	if sysObjectID != nil {
		d.SysObjectID = *sysObjectID
	}
	if sysName != nil {
		d.SysName = *sysName
	}
	if templateID != nil {
		d.SuggestedTemplateID = *templateID
	}
	_ = time.Now // ensure time import
	return d, nil
}
