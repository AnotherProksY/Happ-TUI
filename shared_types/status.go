package sharedtypes

// StatusReport is what `happ-tui status --json` prints, as sent by the core.
// Numbers are raw (bytes, bytes/s, unix seconds); null means unknown.
type StatusReport struct {
	Core           string             `json:"core"`
	Root           bool               `json:"root"`
	Service        bool               `json:"service"`
	Connection     string             `json:"connection"`
	Tun            bool               `json:"tun"`
	Profile        *StatusProfile     `json:"profile"`
	ConnectedSince *int64             `json:"connected_since"`
	Traffic        *TrafficCounters   `json:"traffic"`
	Rate           *TrafficCounters   `json:"rate"`
	LatencyMs      *int               `json:"latency_ms"`
	Subscription   *SubscriptionUsage `json:"subscription"`
	LastError      *string            `json:"last_error"`
}

type StatusProfile struct {
	GroupId  int    `json:"group_id"`
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Group    string `json:"group"`
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
}

type TrafficCounters struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

type SubscriptionUsage struct {
	Used   int64 `json:"used"`
	Total  int64 `json:"total"`
	Expire int64 `json:"expire"`
}

// UpData connects to Profile, or to the last used profile when it is nil.
type UpData struct {
	Profile *ProfileID `json:"profile,omitempty"`
	Tun     bool       `json:"tun"`
}

type ToggleData struct {
	Profile *ProfileID `json:"profile,omitempty"`
	Tun     bool       `json:"tun"`
}
