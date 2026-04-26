package handlers

import "strings"

// parseTagsQuery turns repeated `tag=key:value` query string values into a
// map. Empty input or malformed entries (no colon, empty key, empty value)
// are silently dropped — handlers treat the result as an opt-in filter.
//
// Examples:
//
//	["env:prod"]             -> {"env": "prod"}
//	["env:prod","tier:web"]  -> {"env": "prod", "tier": "web"}
//	["env:prod:eu"]          -> {"env": "prod:eu"}     // colons in values OK
//	["malformed", ":val", "k:"] -> {}                  // all dropped
func parseTagsQuery(values []string) map[string]string {
	tags := make(map[string]string, len(values))
	for _, v := range values {
		idx := strings.Index(v, ":")
		if idx <= 0 || idx == len(v)-1 {
			continue
		}
		tags[v[:idx]] = v[idx+1:]
	}
	return tags
}
