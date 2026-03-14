package engine

import "github.com/sylvester-francis/watchdog/internal/snmp"

// TemplateOID describes an SNMP OID to add to a device template.
type TemplateOID struct {
	OID       string
	Name      string
	Unit      string
	Category  string
	IsCounter bool
}

// AppendTemplateOIDs adds OIDs to an existing device template by ID.
func AppendTemplateOIDs(templateID string, oids []TemplateOID) {
	t := snmp.GetByID(templateID)
	if t == nil {
		return
	}
	for _, o := range oids {
		t.OIDs = append(t.OIDs, snmp.OIDEntry{
			OID:       o.OID,
			Name:      o.Name,
			Unit:      o.Unit,
			Category:  o.Category,
			IsCounter: o.IsCounter,
		})
	}
}
