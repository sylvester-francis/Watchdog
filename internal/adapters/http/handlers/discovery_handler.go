package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/internal/core/services"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// DiscoveryHandler handles network discovery API endpoints.
type DiscoveryHandler struct {
	discoverySvc *services.DiscoveryService
	agentRepo    ports.AgentRepository
}

// NewDiscoveryHandler creates a new DiscoveryHandler.
func NewDiscoveryHandler(discoverySvc *services.DiscoveryService, agentRepo ports.AgentRepository) *DiscoveryHandler {
	return &DiscoveryHandler{
		discoverySvc: discoverySvc,
		agentRepo:    agentRepo,
	}
}

type startScanRequest struct {
	AgentID     string `json:"agent_id"`
	Subnet      string `json:"subnet"`
	Community   string `json:"community"`
	SNMPVersion string `json:"snmp_version"`
}

// StartScan starts a network discovery scan.
// POST /api/v1/discovery
func (h *DiscoveryHandler) StartScan(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	var req startScanRequest
	if err := c.Bind(&req); err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid request body")
	}

	if req.AgentID == "" || req.Subnet == "" {
		return errJSON(c, http.StatusBadRequest, "agent_id and subnet are required")
	}

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid agent_id")
	}

	scan, err := h.discoverySvc.StartScan(c.Request().Context(), userID, agentID, req.Subnet, req.Community, req.SNMPVersion)
	if err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"data": map[string]any{
			"id":      scan.ID.String(),
			"subnet":  scan.Subnet,
			"status":  scan.Status,
			"agent_id": scan.AgentID.String(),
		},
	})
}

// ListScans returns all discovery scans for the user.
// GET /api/v1/discovery
func (h *DiscoveryHandler) ListScans(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	scans, err := h.discoverySvc.ListScans(c.Request().Context(), userID)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to fetch scans")
	}

	result := make([]map[string]any, 0, len(scans))
	for _, s := range scans {
		entry := map[string]any{
			"id":         s.ID.String(),
			"agent_id":   s.AgentID.String(),
			"subnet":     s.Subnet,
			"status":     s.Status,
			"host_count": s.HostCount,
			"created_at": s.CreatedAt,
		}
		if s.StartedAt != nil {
			entry["started_at"] = s.StartedAt
		}
		if s.CompletedAt != nil {
			entry["completed_at"] = s.CompletedAt
		}
		if s.ErrorMessage != "" {
			entry["error_message"] = s.ErrorMessage
		}
		result = append(result, entry)
	}

	return c.JSON(http.StatusOK, map[string]any{"data": result})
}

// GetScan returns a scan with its discovered devices.
// GET /api/v1/discovery/:id
func (h *DiscoveryHandler) GetScan(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return errJSON(c, http.StatusUnauthorized, "unauthorized")
	}

	scanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid scan ID")
	}

	scan, devices, err := h.discoverySvc.GetScan(c.Request().Context(), scanID)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, "failed to fetch scan")
	}
	if scan == nil || scan.UserID != userID {
		return errJSON(c, http.StatusNotFound, "scan not found")
	}

	deviceList := make([]map[string]any, 0, len(devices))
	for _, d := range devices {
		deviceList = append(deviceList, map[string]any{
			"id":                    d.ID.String(),
			"ip":                    d.IP,
			"hostname":              d.Hostname,
			"sys_descr":             d.SysDescr,
			"sys_object_id":         d.SysObjectID,
			"sys_name":              d.SysName,
			"snmp_reachable":        d.SNMPReachable,
			"ping_reachable":        d.PingReachable,
			"suggested_template_id": d.SuggestedTemplateID,
			"monitor_created":       d.MonitorCreated,
			"discovered_at":         d.DiscoveredAt,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": map[string]any{
			"scan": map[string]any{
				"id":            scan.ID.String(),
				"agent_id":      scan.AgentID.String(),
				"subnet":        scan.Subnet,
				"status":        scan.Status,
				"host_count":    scan.HostCount,
				"started_at":    scan.StartedAt,
				"completed_at":  scan.CompletedAt,
				"error_message": scan.ErrorMessage,
				"created_at":    scan.CreatedAt,
			},
			"devices": deviceList,
		},
	})
}

