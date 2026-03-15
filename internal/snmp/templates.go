package snmp

import "strings"

// OIDEntry defines a single OID to poll from a network device.
type OIDEntry struct {
	OID       string `json:"oid"`
	Name      string `json:"name"`
	Unit      string `json:"unit,omitempty"`
	Category  string `json:"category"`
	IsCounter bool   `json:"is_counter,omitempty"`
}

// DeviceTemplate defines a set of OIDs for a class of network device.
type DeviceTemplate struct {
	ID              string     `json:"id"`
	Vendor          string     `json:"vendor"`
	Model           string     `json:"model"`
	Description     string     `json:"description"`
	SysObjectIDs    []string   `json:"sys_object_ids"`
	OIDs            []OIDEntry `json:"oids"`
	DefaultInterval int        `json:"default_interval"`
}

// templates is the compiled-in list of device templates.
var templates = []DeviceTemplate{
	{
		ID:          "cisco-ios",
		Vendor:      "Cisco",
		Model:       "IOS Switch/Router",
		Description: "Cisco IOS devices — switches, routers, L3 switches",
		SysObjectIDs: []string{
			"1.3.6.1.4.1.9.1.*",
		},
		DefaultInterval: 60,
		OIDs: []OIDEntry{
			// System
			{OID: "1.3.6.1.2.1.1.1.0", Name: "System Description", Category: "system"},
			{OID: "1.3.6.1.2.1.1.3.0", Name: "System Uptime", Category: "system"},
			{OID: "1.3.6.1.2.1.1.5.0", Name: "System Name", Category: "system"},
			// CPU (CISCO-PROCESS-MIB)
			{OID: "1.3.6.1.4.1.9.9.109.1.1.1.1.7.1", Name: "CPU Busy (5s)", Unit: "%", Category: "cpu"},
			{OID: "1.3.6.1.4.1.9.9.109.1.1.1.1.8.1", Name: "CPU Busy (1m)", Unit: "%", Category: "cpu"},
			{OID: "1.3.6.1.4.1.9.9.109.1.1.1.1.5.1", Name: "CPU Busy (5m)", Unit: "%", Category: "cpu"},
			// Memory (CISCO-MEMORY-POOL-MIB)
			{OID: "1.3.6.1.4.1.9.9.48.1.1.1.5.1", Name: "Memory Used", Unit: "bytes", Category: "memory"},
			{OID: "1.3.6.1.4.1.9.9.48.1.1.1.6.1", Name: "Memory Free", Unit: "bytes", Category: "memory"},
			// Interfaces (IF-MIB walk)
			{OID: "1.3.6.1.2.1.2.2.1.2", Name: "Interface Description", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.8", Name: "Interface Oper Status", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.10", Name: "Interface In Octets", Unit: "bytes", Category: "interface", IsCounter: true},
			{OID: "1.3.6.1.2.1.2.2.1.16", Name: "Interface Out Octets", Unit: "bytes", Category: "interface", IsCounter: true},
			{OID: "1.3.6.1.2.1.2.2.1.14", Name: "Interface In Errors", Unit: "errors", Category: "interface", IsCounter: true},
			{OID: "1.3.6.1.2.1.2.2.1.20", Name: "Interface Out Errors", Unit: "errors", Category: "interface", IsCounter: true},
		},
	},
	{
		ID:          "hp-procurve",
		Vendor:      "HP",
		Model:       "ProCurve Switch",
		Description: "HP ProCurve and Aruba managed switches",
		SysObjectIDs: []string{
			"1.3.6.1.4.1.11.2.3.7.11.*",
			"1.3.6.1.4.1.11.2.3.7.8.*",
		},
		DefaultInterval: 60,
		OIDs: []OIDEntry{
			{OID: "1.3.6.1.2.1.1.1.0", Name: "System Description", Category: "system"},
			{OID: "1.3.6.1.2.1.1.3.0", Name: "System Uptime", Category: "system"},
			{OID: "1.3.6.1.2.1.1.5.0", Name: "System Name", Category: "system"},
			// HP-ICF-MIB
			{OID: "1.3.6.1.4.1.11.2.14.11.5.1.9.6.1.0", Name: "CPU Utilization", Unit: "%", Category: "cpu"},
			{OID: "1.3.6.1.4.1.11.2.14.11.5.1.1.2.1.1.1.7.1", Name: "Memory Used", Unit: "bytes", Category: "memory"},
			// Interfaces
			{OID: "1.3.6.1.2.1.2.2.1.2", Name: "Interface Description", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.8", Name: "Interface Oper Status", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.10", Name: "Interface In Octets", Unit: "bytes", Category: "interface", IsCounter: true},
			{OID: "1.3.6.1.2.1.2.2.1.16", Name: "Interface Out Octets", Unit: "bytes", Category: "interface", IsCounter: true},
		},
	},
	{
		ID:          "mikrotik-routeros",
		Vendor:      "MikroTik",
		Model:       "RouterOS",
		Description: "MikroTik routers and switches running RouterOS",
		SysObjectIDs: []string{
			"1.3.6.1.4.1.14988.1.*",
		},
		DefaultInterval: 60,
		OIDs: []OIDEntry{
			{OID: "1.3.6.1.2.1.1.1.0", Name: "System Description", Category: "system"},
			{OID: "1.3.6.1.2.1.1.3.0", Name: "System Uptime", Category: "system"},
			{OID: "1.3.6.1.2.1.1.5.0", Name: "System Name", Category: "system"},
			// MIKROTIK-MIB
			{OID: "1.3.6.1.2.1.25.3.3.1.2.1", Name: "CPU Load", Unit: "%", Category: "cpu"},
			{OID: "1.3.6.1.2.1.25.2.3.1.6.65536", Name: "Memory Used", Unit: "units", Category: "memory"},
			{OID: "1.3.6.1.2.1.25.2.3.1.5.65536", Name: "Memory Total", Unit: "units", Category: "memory"},
			// Interfaces
			{OID: "1.3.6.1.2.1.2.2.1.2", Name: "Interface Description", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.8", Name: "Interface Oper Status", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.10", Name: "Interface In Octets", Unit: "bytes", Category: "interface", IsCounter: true},
			{OID: "1.3.6.1.2.1.2.2.1.16", Name: "Interface Out Octets", Unit: "bytes", Category: "interface", IsCounter: true},
			// Disk
			{OID: "1.3.6.1.2.1.25.2.3.1.6.131072", Name: "Disk Used", Unit: "units", Category: "storage"},
			{OID: "1.3.6.1.2.1.25.2.3.1.5.131072", Name: "Disk Total", Unit: "units", Category: "storage"},
		},
	},
	{
		ID:          "ubiquiti-edgerouter",
		Vendor:      "Ubiquiti",
		Model:       "EdgeRouter / UniFi",
		Description: "Ubiquiti EdgeRouter and UniFi networking devices",
		SysObjectIDs: []string{
			"1.3.6.1.4.1.41112.*",
		},
		DefaultInterval: 60,
		OIDs: []OIDEntry{
			{OID: "1.3.6.1.2.1.1.1.0", Name: "System Description", Category: "system"},
			{OID: "1.3.6.1.2.1.1.3.0", Name: "System Uptime", Category: "system"},
			{OID: "1.3.6.1.2.1.1.5.0", Name: "System Name", Category: "system"},
			// HOST-RESOURCES (EdgeOS is Linux-based)
			{OID: "1.3.6.1.2.1.25.3.3.1.2.1", Name: "CPU Load", Unit: "%", Category: "cpu"},
			{OID: "1.3.6.1.2.1.25.2.3.1.5.1", Name: "Memory Total", Unit: "units", Category: "memory"},
			{OID: "1.3.6.1.2.1.25.2.3.1.6.1", Name: "Memory Used", Unit: "units", Category: "memory"},
			// Interfaces
			{OID: "1.3.6.1.2.1.2.2.1.2", Name: "Interface Description", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.8", Name: "Interface Oper Status", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.10", Name: "Interface In Octets", Unit: "bytes", Category: "interface", IsCounter: true},
			{OID: "1.3.6.1.2.1.2.2.1.16", Name: "Interface Out Octets", Unit: "bytes", Category: "interface", IsCounter: true},
		},
	},
	{
		ID:          "apc-ups",
		Vendor:      "APC",
		Model:       "UPS",
		Description: "APC Uninterruptible Power Supplies (Smart-UPS, Back-UPS)",
		SysObjectIDs: []string{
			"1.3.6.1.4.1.318.1.3.*",
		},
		DefaultInterval: 30,
		OIDs: []OIDEntry{
			{OID: "1.3.6.1.2.1.1.1.0", Name: "System Description", Category: "system"},
			{OID: "1.3.6.1.2.1.1.3.0", Name: "System Uptime", Category: "system"},
			// PowerNet-MIB
			{OID: "1.3.6.1.4.1.318.1.1.1.2.2.1.0", Name: "Battery Capacity", Unit: "%", Category: "battery"},
			{OID: "1.3.6.1.4.1.318.1.1.1.2.2.3.0", Name: "Battery Runtime Remaining", Unit: "timeticks", Category: "battery"},
			{OID: "1.3.6.1.4.1.318.1.1.1.2.1.1.0", Name: "Battery Status", Category: "battery"},
			{OID: "1.3.6.1.4.1.318.1.1.1.2.2.2.0", Name: "Battery Temperature", Unit: "°C", Category: "battery"},
			{OID: "1.3.6.1.4.1.318.1.1.1.4.2.1.0", Name: "Output Load", Unit: "%", Category: "output"},
			{OID: "1.3.6.1.4.1.318.1.1.1.4.2.3.0", Name: "Output Current", Unit: "A", Category: "output"},
			{OID: "1.3.6.1.4.1.318.1.1.1.3.2.1.0", Name: "Input Voltage", Unit: "V", Category: "input"},
			{OID: "1.3.6.1.4.1.318.1.1.1.4.2.1.0", Name: "Output Voltage", Unit: "V", Category: "output"},
			{OID: "1.3.6.1.4.1.318.1.1.1.3.2.4.0", Name: "Input Frequency", Unit: "Hz", Category: "input"},
		},
	},
	{
		ID:          "generic-linux",
		Vendor:      "Generic",
		Model:       "Linux Server",
		Description: "Linux servers running NET-SNMP (snmpd)",
		SysObjectIDs: []string{
			"1.3.6.1.4.1.8072.3.2.*",
		},
		DefaultInterval: 60,
		OIDs: []OIDEntry{
			{OID: "1.3.6.1.2.1.1.1.0", Name: "System Description", Category: "system"},
			{OID: "1.3.6.1.2.1.1.3.0", Name: "System Uptime", Category: "system"},
			{OID: "1.3.6.1.2.1.1.5.0", Name: "System Name", Category: "system"},
			// NET-SNMP CPU
			{OID: "1.3.6.1.4.1.2021.11.9.0", Name: "CPU User %", Unit: "%", Category: "cpu"},
			{OID: "1.3.6.1.4.1.2021.11.10.0", Name: "CPU System %", Unit: "%", Category: "cpu"},
			{OID: "1.3.6.1.4.1.2021.11.11.0", Name: "CPU Idle %", Unit: "%", Category: "cpu"},
			// NET-SNMP Memory
			{OID: "1.3.6.1.4.1.2021.4.5.0", Name: "Total RAM", Unit: "KB", Category: "memory"},
			{OID: "1.3.6.1.4.1.2021.4.6.0", Name: "Available RAM", Unit: "KB", Category: "memory"},
			{OID: "1.3.6.1.4.1.2021.4.11.0", Name: "Total Free Memory", Unit: "KB", Category: "memory"},
			{OID: "1.3.6.1.4.1.2021.4.14.0", Name: "Buffered Memory", Unit: "KB", Category: "memory"},
			{OID: "1.3.6.1.4.1.2021.4.15.0", Name: "Cached Memory", Unit: "KB", Category: "memory"},
			// Load
			{OID: "1.3.6.1.4.1.2021.10.1.3.1", Name: "Load Avg (1m)", Category: "cpu"},
			{OID: "1.3.6.1.4.1.2021.10.1.3.2", Name: "Load Avg (5m)", Category: "cpu"},
			{OID: "1.3.6.1.4.1.2021.10.1.3.3", Name: "Load Avg (15m)", Category: "cpu"},
			// Interfaces
			{OID: "1.3.6.1.2.1.2.2.1.2", Name: "Interface Description", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.8", Name: "Interface Oper Status", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.10", Name: "Interface In Octets", Unit: "bytes", Category: "interface", IsCounter: true},
			{OID: "1.3.6.1.2.1.2.2.1.16", Name: "Interface Out Octets", Unit: "bytes", Category: "interface", IsCounter: true},
			// Disk (UCD-SNMP)
			{OID: "1.3.6.1.4.1.2021.9.1.2", Name: "Disk Path", Category: "storage"},
			{OID: "1.3.6.1.4.1.2021.9.1.6", Name: "Disk Total", Unit: "KB", Category: "storage"},
			{OID: "1.3.6.1.4.1.2021.9.1.8", Name: "Disk Used", Unit: "KB", Category: "storage"},
			{OID: "1.3.6.1.4.1.2021.9.1.9", Name: "Disk % Used", Unit: "%", Category: "storage"},
		},
	},
	{
		ID:          "generic-windows",
		Vendor:      "Generic",
		Model:       "Windows Server",
		Description: "Windows servers with SNMP service enabled (HOST-RESOURCES-MIB)",
		SysObjectIDs: []string{
			"1.3.6.1.4.1.311.1.1.*",
		},
		DefaultInterval: 60,
		OIDs: []OIDEntry{
			{OID: "1.3.6.1.2.1.1.1.0", Name: "System Description", Category: "system"},
			{OID: "1.3.6.1.2.1.1.3.0", Name: "System Uptime", Category: "system"},
			{OID: "1.3.6.1.2.1.1.5.0", Name: "System Name", Category: "system"},
			// HOST-RESOURCES-MIB
			{OID: "1.3.6.1.2.1.25.3.3.1.2", Name: "CPU Load Per Processor", Unit: "%", Category: "cpu"},
			{OID: "1.3.6.1.2.1.25.2.3.1.3", Name: "Storage Description", Category: "storage"},
			{OID: "1.3.6.1.2.1.25.2.3.1.5", Name: "Storage Size", Unit: "units", Category: "storage"},
			{OID: "1.3.6.1.2.1.25.2.3.1.6", Name: "Storage Used", Unit: "units", Category: "storage"},
			{OID: "1.3.6.1.2.1.25.2.2.0", Name: "Total Memory", Unit: "KB", Category: "memory"},
			// Interfaces
			{OID: "1.3.6.1.2.1.2.2.1.2", Name: "Interface Description", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.8", Name: "Interface Oper Status", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.10", Name: "Interface In Octets", Unit: "bytes", Category: "interface", IsCounter: true},
			{OID: "1.3.6.1.2.1.2.2.1.16", Name: "Interface Out Octets", Unit: "bytes", Category: "interface", IsCounter: true},
		},
	},
	{
		ID:          "generic-network",
		Vendor:      "Generic",
		Model:       "Network Device",
		Description: "Any SNMP-capable network device — basic IF-MIB monitoring",
		SysObjectIDs: []string{
			"*",
		},
		DefaultInterval: 60,
		OIDs: []OIDEntry{
			{OID: "1.3.6.1.2.1.1.1.0", Name: "System Description", Category: "system"},
			{OID: "1.3.6.1.2.1.1.3.0", Name: "System Uptime", Category: "system"},
			{OID: "1.3.6.1.2.1.1.5.0", Name: "System Name", Category: "system"},
			{OID: "1.3.6.1.2.1.2.1.0", Name: "Interface Count", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.2", Name: "Interface Description", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.8", Name: "Interface Oper Status", Category: "interface"},
			{OID: "1.3.6.1.2.1.2.2.1.10", Name: "Interface In Octets", Unit: "bytes", Category: "interface", IsCounter: true},
			{OID: "1.3.6.1.2.1.2.2.1.16", Name: "Interface Out Octets", Unit: "bytes", Category: "interface", IsCounter: true},
			{OID: "1.3.6.1.2.1.2.2.1.14", Name: "Interface In Errors", Unit: "errors", Category: "interface", IsCounter: true},
			{OID: "1.3.6.1.2.1.2.2.1.20", Name: "Interface Out Errors", Unit: "errors", Category: "interface", IsCounter: true},
		},
	},
}

// templateIndex is a lookup map keyed by template ID.
var templateIndex map[string]*DeviceTemplate

func init() {
	templateIndex = make(map[string]*DeviceTemplate, len(templates))
	for i := range templates {
		templateIndex[templates[i].ID] = &templates[i]
	}
}

// GetAllTemplates returns all compiled-in device templates.
func GetAllTemplates() []DeviceTemplate {
	result := make([]DeviceTemplate, len(templates))
	copy(result, templates)
	return result
}

// GetByID returns a device template by ID, or nil if not found.
func GetByID(id string) *DeviceTemplate {
	t, ok := templateIndex[id]
	if !ok {
		return nil
	}
	return t
}

// MatchBySysObjectID finds the best-matching template for a sysObjectID.
// It prefers vendor-specific matches over the generic-network fallback.
func MatchBySysObjectID(sysOID string) *DeviceTemplate {
	var fallback *DeviceTemplate

	for i := range templates {
		t := &templates[i]
		for _, pattern := range t.SysObjectIDs {
			if pattern == "*" {
				fallback = t
				continue
			}
			if matchOIDPattern(pattern, sysOID) {
				return t
			}
		}
	}

	return fallback
}

// matchOIDPattern matches an OID against a glob pattern.
// Supports trailing "*" wildcard (e.g., "1.3.6.1.4.1.9.1.*" matches "1.3.6.1.4.1.9.1.1045").
func matchOIDPattern(pattern, oid string) bool {
	if !strings.HasSuffix(pattern, ".*") {
		return pattern == oid
	}
	prefix := strings.TrimSuffix(pattern, "*")
	return strings.HasPrefix(oid, prefix)
}
