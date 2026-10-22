package structs

import "encoding/json"

// database types
type DBAddProfileData struct {
	Uri      string `json:"uri"`
	GroupId  int    `json:"group_id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Address  string `json:"address,omitzero"`
	Host     string `json:"host,omitzero"`
	NanoID   string `json:"nano-id,omitzero"`
}

// commands and responses
type DeleteProfilesData struct {
	Profiles []ProfileID `json:"profiles"`
}

type ProfilesDeleted struct {
	DeletedProfiles []ProfileID `json:"deleted-profiles"`
}

type AddProfilesData struct {
	Uris    string `json:"uris"`
	GroupId int    `json:"group_id"`
}

type ProfilesAdded struct {
	Profiles []Profile `json:"profiles"`
}

type AddGroupData struct {
	Name            string `json:"name"`
	SubscriptionUrl string `json:"subscription_url"`
}

type GroupAdded struct {
	Id              int    `json:"id"`
	SubscriptionUrl string `json:"subscription_url"`
	Name            string `json:"name"`
}

type DeleteGroupData struct {
	Id int `json:"id"`
}

type GroupDeleted struct {
	Id int `json:"id"`
}

type ConnectData struct {
	Profile ProfileID `json:"profile"`
}

type DisconnectData struct{}

type TestProfileData struct {
	Profile ProfileID `json:"profile"`
}

type StopTestsData struct{}

type ProfileUpdated struct {
	Profile Profile `json:"profile"`
}

type GetApplicationStateData struct{}

type ApplicationState struct {
	Groups           []GroupWithProfiles `json:"groups"`
	ConnectionStatus ProxyStatus         `json:"connection-status"`
	TunStatus        bool                `json:"tun-status"`
}

type UpdateSubscriptionData struct {
	GroupId int `json:"group_id"`
}

type SubscriptionUpdated struct {
	GroupId          int              `json:"group_id"`
	Name             string           `json:"name"`
	Profiles         []Profile        `json:"profiles"`
	SubscriptionInfo SubscriptionInfo `json:"subscription_info"`
}

type Warning struct {
	Key     string `json:"key"`
	Content string `json:"content"`
}

type DieData struct{}

type (
	IsRootData   struct{}
	IsRootAnswer struct {
		IsRoot bool
	}
)

type UpdateProfileData struct {
	Profile ProfileID
	Name    string
}

// general types
type DBConfig struct {
	LastGroupId            int       `json:"last_group_id"`
	LatestConnectedProfile ProfileID `json:"latest_connected_profile"`
}

type GroupWithProfiles struct {
	Group    Group     `json:"group"`
	Profiles []Profile `json:"profiles"`
}

type Group struct {
	Id               int              `json:"id"`
	SubscriptionUrl  string           `json:"subscription_url"`
	Name             string           `json:"name"`
	LastId           int              `json:"last_id"`
	SubscriptionInfo SubscriptionInfo `json:"subscription_info,omitzero"`
	LastUpdated      int64            `json:"last_updated,omitzero"`
}

// SubscriptionInfo is the metadata a provider sends in Happ-style response headers.
type SubscriptionInfo struct {
	Title          string `json:"title,omitzero"`
	Upload         int64  `json:"upload,omitzero"`
	Download       int64  `json:"download,omitzero"`
	Total          int64  `json:"total,omitzero"`
	Expire         int64  `json:"expire,omitzero"`
	UpdateInterval int    `json:"update_interval,omitzero"` // seconds
	SupportUrl     string `json:"support_url,omitzero"`
	WebPageUrl     string `json:"web_page_url,omitzero"`
	Announce       string `json:"announce,omitzero"`
	// CheckUrl is the url the provider wants profiles to be tested with.
	CheckUrl string `json:"check_url,omitzero"`
	// ChangeUserAgent is the user-agent the provider asked to use for later updates.
	ChangeUserAgent string `json:"change_user_agent,omitzero"`
}

type Profile struct {
	Id         int    `json:"id"`
	GroupId    int    `json:"group_id"`
	NanoID     string `json:"nano-id,omitzero"`
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	Uri        string `json:"uri"`
	Address    string `json:"address,omitzero"`
	Host       string `json:"host,omitzero"`
	TestResult int    `json:"test-result"`
}

type ProfileID struct {
	Id      int `json:"id"`
	GroupId int `json:"group_id"`
}

type TCPMessage struct {
	Msg       string          `json:"msg"`
	RequestId string          `json:"request_id,omitempty"`
	Data      json.RawMessage `json:"data"`
}

// Reply answers a command that carried request_id, to that client only.
type Reply struct {
	Msg       string `json:"msg"`
	RequestId string `json:"request_id"`
	Data      any    `json:"data"`
}

// UpData connects to Profile, or to the last used profile when it is nil.
type UpData struct {
	Profile *ProfileID `json:"profile,omitempty"`
	Tun     bool       `json:"tun"`
}

// ToggleData is a click on a widget, see Cmd.Toggle.
type ToggleData struct {
	Profile *ProfileID `json:"profile,omitempty"`
	Tun     bool       `json:"tun"`
}

// StatusReport is what `happ-tui status --json` prints. Numbers are raw
// (bytes, bytes/s, unix seconds); null means unknown or not applicable.
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

// LogsData asks for the last N lines of the core's log; with Follow, new
// lines are sent as "log" notifications to this client.
type LogsData struct {
	N      int  `json:"n"`
	Follow bool `json:"follow"`
}

type LogsReply struct {
	Lines []string `json:"lines"`
}

// ClientConfig is the part of the config that clients need.
type ClientConfig struct {
	SocksPort    int  `json:"socks-port"`
	HttpPort     int  `json:"http-port"`
	NoBackground bool `json:"no-background"`
	Root         bool `json:"root"`
	Service      bool `json:"service"`
}

// ErrorReply is the data of an "error" reply. Key is stable for scripts.
type ErrorReply struct {
	Key     string `json:"key"`
	Message string `json:"message"`
}

type Message[T any] struct {
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type ProxyStatus struct {
	// Connection is "connecting", "connected", "disconnected" or "error".
	Connection     string  `json:"connection"`
	IsTunEnabled   bool    `json:"is_tun_enabled"`
	Profile        Profile `json:"profile"`
	ConnectedSince int64   `json:"connected_since,omitzero"`
	LastError      string  `json:"last_error,omitzero"`
}

type (
	DisableTunData struct{}
	EnableTunData  struct{}
)
