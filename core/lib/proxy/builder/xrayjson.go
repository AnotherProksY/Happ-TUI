package builder

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/AnotherProksY/Happ-TUI/core/structs"
	"github.com/AnotherProksY/Happ-TUI/core/utils"
)

const tproxyInboundTag = "tproxy-in"

// IsXrayJSON reports whether a profile stores a full Xray config instead of a share link.
func IsXrayJSON(uri string) bool {
	trimmed := bytes.TrimSpace([]byte(uri))
	return len(trimmed) > 0 && trimmed[0] == '{'
}

// ProfileConfig builds the Xray config for a profile, whether it is a share
// link or a full Xray config from a JSON subscription. Pass -1 to omit the
// http or tproxy inbound.
func ProfileConfig(profile structs.Profile, socksPort int, httpPort int, tproxyPort int) ([]byte, error) {
	if IsXrayJSON(profile.Uri) {
		return FromXrayJSON([]byte(profile.Uri), socksPort, httpPort, tproxyPort)
	}
	return utils.ParseUri(profile.Uri, socksPort, httpPort, tproxyPort)
}

// FromXrayJSON keeps the provider's outbounds, routing and dns but replaces
// the inbounds with Happ-TUI's own, reusing the provider's inbound tags so
// that routing rules referring to them keep working.
func FromXrayJSON(raw []byte, socksPort int, httpPort int, tproxyPort int) ([]byte, error) {
	var coreConfig map[string]any
	if err := json.Unmarshal(raw, &coreConfig); err != nil {
		return nil, fmt.Errorf("parse Xray JSON profile: %w", err)
	}
	if coreConfig == nil {
		return nil, ErrCoreIsNill
	}
	if _, err := getOutbounds(coreConfig); err != nil {
		return nil, err
	}

	socksTag, httpTag := "socks", "http"
	if inbounds, ok := coreConfig["inbounds"].([]any); ok {
		for _, inboundValue := range inbounds {
			inbound, ok := inboundValue.(map[string]any)
			if !ok {
				continue
			}
			tag := stringValue(inbound["tag"])
			if tag == "" {
				continue
			}
			switch stringValue(inbound["protocol"]) {
			case "socks", "mixed":
				socksTag = tag
			case "http":
				httpTag = tag
			}
		}
	}

	inbounds := []any{localInbound(socksTag, "socks", socksPort)}
	if httpPort != -1 {
		inbounds = append(inbounds, localInbound(httpTag, "http", httpPort))
	}
	if tproxyPort != -1 {
		inbounds = append(inbounds, map[string]any{
			"tag":      tproxyInboundTag,
			"port":     tproxyPort,
			"protocol": "tunnel",
			"settings": map[string]any{
				"network":        "tcp,udp",
				"allowedNetwork": "tcp,udp",
				"followRedirect": true,
			},
			"sniffing":       sniffing(),
			"streamSettings": map[string]any{"sockopt": map[string]any{"tproxy": "tproxy"}},
		})
	}
	coreConfig["inbounds"] = inbounds
	if tproxyPort != -1 {
		addTunDNSHijack(coreConfig)
	}

	// Happ-TUI-only metadata that Xray does not need.
	delete(coreConfig, "remarks")
	delete(coreConfig, "meta")

	return json.Marshal(coreConfig)
}

// addTunDNSHijack answers DNS queries caught by TUN mode with the provider's
// own dns servers instead of letting them reach the system resolver's upstream.
// Queries other than A/AAAA are rejected so they can't leak either.
func addTunDNSHijack(coreConfig map[string]any) {
	outbounds, _ := getOutbounds(coreConfig)
	tag := uniqueOutboundTag(outbounds, xrayDNSOutboundTag)
	coreConfig["outbounds"] = append(outbounds, map[string]any{
		"tag":      tag,
		"protocol": "dns",
		"settings": map[string]any{"nonIPQuery": "reject"},
	})

	routing := getOrCreateObject(coreConfig, "routing")
	rules := getOrCreateArray(routing, "rules")
	routing["rules"] = append([]any{map[string]any{
		"type":        "field",
		"ruleTag":     xrayDNSHijackRuleTag,
		"inboundTag":  []string{tproxyInboundTag},
		"port":        "53",
		"outboundTag": tag,
	}}, rules...)
}

func localInbound(tag string, protocol string, port int) map[string]any {
	settings := map[string]any{}
	if protocol == "socks" {
		settings["udp"] = true
		settings["auth"] = "noauth"
	}
	return map[string]any{
		"tag":      tag,
		"listen":   "127.0.0.1",
		"port":     port,
		"protocol": protocol,
		"settings": settings,
		"sniffing": sniffing(),
	}
}

func sniffing() map[string]any {
	return map[string]any{
		"enabled":      true,
		"destOverride": []string{"http", "tls", "quic"},
		"routeOnly":    false,
	}
}
