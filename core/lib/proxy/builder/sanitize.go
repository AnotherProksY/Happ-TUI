package builder

import (
	"encoding/json"
	"fmt"
)

// allowedSections are the top level parts of a profile's Xray config that
// are kept. Everything else (api, stats, reverse, metrics, unknown keys) is
// dropped: profiles come from subscriptions and pasted text, and in a root
// core such sections could expose a control API or reach the file system.
// The core adds its own api and stats sections after this filter.
var allowedSections = map[string]bool{
	"log":              true,
	"dns":              true,
	"routing":          true,
	"inbounds":         true, // replaced by the core's own inbounds
	"outbounds":        true,
	"policy":           true,
	"observatory":      true,
	"burstObservatory": true,
	"fakedns":          true,
	"transport":        true,
}

// allowedLogKeys keeps log settings that don't name files: access and error
// log paths would let a profile write wherever the xray process may.
var allowedLogKeys = map[string]bool{
	"loglevel":    true,
	"dnsLog":      true,
	"maskAddress": true,
}

// Sanitize removes the parts of a profile's Xray config listed above.
func Sanitize(raw []byte) ([]byte, error) {
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("parse Xray config: %w", err)
	}
	if config == nil {
		return nil, ErrCoreIsNill
	}

	for key := range config {
		if !allowedSections[key] {
			delete(config, key)
		}
	}
	if logConfig, ok := config["log"].(map[string]any); ok {
		for key := range logConfig {
			if !allowedLogKeys[key] {
				delete(logConfig, key)
			}
		}
	} else {
		delete(config, "log")
	}

	return json.Marshal(config)
}
