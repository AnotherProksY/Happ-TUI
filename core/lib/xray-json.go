package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/AnotherProksY/Happ-TUI/core/structs"
	"slices"
	"strings"
)

// getDBAddProfileDatasFromXrayJSON reads a subscription made of full Xray
// configs (a single object or an array), as served to Happ by panels like
// Remnawave. Each config becomes one profile.
func getDBAddProfileDatasFromXrayJSON(str string, group_id int) ([]structs.DBAddProfileData, error) {
	var configs []json.RawMessage
	if strings.HasPrefix(str, "{") {
		configs = []json.RawMessage{json.RawMessage(str)}
	} else if err := json.Unmarshal([]byte(str), &configs); err != nil {
		return nil, err
	}

	var profiles []structs.DBAddProfileData
	for i, raw := range configs {
		profile, err := getDBAddProfileDataFromXrayJSON(raw, group_id)
		if err != nil {
			return nil, fmt.Errorf("config %d: %w", i, err)
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

type xrayJSONProfile struct {
	Remarks   string             `json:"remarks"`
	Outbounds []xrayJSONOutbound `json:"outbounds"`
}

type xrayJSONOutbound struct {
	Protocol string `json:"protocol"`
	Settings struct {
		Address string `json:"address"`
		Vnext   []struct {
			Address string `json:"address"`
		} `json:"vnext"`
		Servers []struct {
			Address string `json:"address"`
		} `json:"servers"`
	} `json:"settings"`
	StreamSettings struct {
		TlsSettings struct {
			ServerName string `json:"serverName"`
		} `json:"tlsSettings"`
		RealitySettings struct {
			ServerName string `json:"serverName"`
		} `json:"realitySettings"`
	} `json:"streamSettings"`
}

func getDBAddProfileDataFromXrayJSON(raw json.RawMessage, group_id int) (structs.DBAddProfileData, error) {
	var profile_data structs.DBAddProfileData
	var parsed xrayJSONProfile
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return profile_data, err
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return profile_data, err
	}

	profile_data = structs.DBAddProfileData{
		Name:    strings.TrimSpace(parsed.Remarks),
		Uri:     compact.String(),
		GroupId: group_id,
		NanoID:  generateNanoID(),
	}

	for _, outbound := range parsed.Outbounds {
		if slices.Contains([]string{"freedom", "blackhole", "dns", "loopback"}, outbound.Protocol) {
			continue
		}
		profile_data.Protocol = outbound.Protocol
		switch {
		case outbound.Settings.Address != "":
			profile_data.Address = outbound.Settings.Address
		case len(outbound.Settings.Vnext) > 0:
			profile_data.Address = outbound.Settings.Vnext[0].Address
		case len(outbound.Settings.Servers) > 0:
			profile_data.Address = outbound.Settings.Servers[0].Address
		}
		profile_data.Host = outbound.StreamSettings.RealitySettings.ServerName
		if profile_data.Host == "" {
			profile_data.Host = outbound.StreamSettings.TlsSettings.ServerName
		}
		break
	}
	if profile_data.Protocol == "" {
		return profile_data, fmt.Errorf("no proxy outbound found")
	}
	if profile_data.Name == "" {
		profile_data.Name = profile_data.Address
	}
	return profile_data, nil
}
