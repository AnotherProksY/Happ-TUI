package builder

import (
	"encoding/json"
	"fmt"
	"slices"
)

const apiTag = "happ-tui-api"

// notProxyProtocols are outbounds whose traffic doesn't go through the VPN.
var notProxyProtocols = []string{"freedom", "blackhole", "dns", "loopback"}

// AddStats lets the core read traffic counters: it enables Xray's stats with
// per outbound counters and the StatsService API on 127.0.0.1:apiPort. Only
// StatsService is enabled, so the API can't change inbounds or outbounds.
// It runs after Sanitize, which drops a profile's own api and stats. The
// returned tags are the outbounds that carry VPN traffic.
func AddStats(raw []byte, apiPort int) ([]byte, []string, error) {
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, nil, fmt.Errorf("parse Xray config: %w", err)
	}
	outbounds, err := getOutbounds(config)
	if err != nil {
		return nil, nil, err
	}

	var proxyTags []string
	for _, value := range outbounds {
		outbound, ok := value.(map[string]any)
		if !ok || slices.Contains(notProxyProtocols, stringValue(outbound["protocol"])) {
			continue
		}
		if tag := stringValue(outbound["tag"]); tag != "" {
			proxyTags = append(proxyTags, tag)
		}
	}

	config["stats"] = map[string]any{}
	config["api"] = map[string]any{
		"tag":      apiTag,
		"services": []string{"StatsService"},
	}
	policy := getOrCreateObject(config, "policy")
	system := getOrCreateObject(policy, "system")
	system["statsOutboundUplink"] = true
	system["statsOutboundDownlink"] = true

	inbounds, _ := config["inbounds"].([]any)
	config["inbounds"] = append(inbounds, map[string]any{
		"tag":      apiTag,
		"listen":   "127.0.0.1",
		"port":     apiPort,
		"protocol": "dokodemo-door",
		"settings": map[string]any{"address": "127.0.0.1"},
	})

	routing := getOrCreateObject(config, "routing")
	rules := getOrCreateArray(routing, "rules")
	routing["rules"] = append([]any{map[string]any{
		"type":        "field",
		"inboundTag":  []string{apiTag},
		"outboundTag": apiTag,
	}}, rules...)

	out, err := json.Marshal(config)
	return out, proxyTags, err
}
