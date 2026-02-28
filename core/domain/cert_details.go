package domain

import (
	"time"

	"github.com/google/uuid"
)

// CertDetails holds TLS certificate metadata for a monitor.
type CertDetails struct {
	MonitorID     uuid.UUID
	TenantID      uuid.UUID
	LastCheckedAt time.Time
	ExpiryDays    *int
	Issuer        string
	SANs          []string
	Algorithm     string
	KeySize       int
	SerialNumber  string
	ChainValid    bool
}
