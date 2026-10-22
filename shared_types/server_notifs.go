package sharedtypes

import "encoding/json"

type TcpMessage struct {
	Msg       string          `json:"msg"`
	RequestId string          `json:"request_id,omitempty"`
	Data      json.RawMessage `json:"data"`
}

type ServerNotification interface {
	IsNotification()
}

type ApplicationState struct {
	Groups           []GroupWithProfiles `json:"groups"`
	ConnectionStatus ProxyStatus         `json:"connection-status"`
	TunStatus        bool                `json:"tun-status"`
}

func (a ApplicationState) IsNotification() {}

type ProfileUpdated struct {
	Profile Profile `json:"profile"`
}

func (p ProfileUpdated) IsNotification() {}

type GroupWithProfiles struct {
	Group    Group     `json:"group"`
	Profiles []Profile `json:"profiles"`
}

type ProxyStatus struct {
	Connection   string  `json:"connection"`
	IsTunEnabled bool    `json:"is_tun_enabled"`
	Profile      Profile `json:"profile"`
}

func (p ProxyStatus) IsNotification() {}

type ProfilesAdded struct {
	Profiles []Profile `json:"profiles"`
}

func (p ProfilesAdded) IsNotification() {}

type ProfilesDeleted struct {
	DeletedProfiles []ProfileID `json:"deleted-profiles"`
}

func (p ProfilesDeleted) IsNotification() {}

type GroupAdded struct {
	Id              int    `json:"id"`
	SubscriptionUrl string `json:"subscription_url"`
	Name            string `json:"name"`
}

func (g GroupAdded) IsNotification() {}

type GroupDeleted struct {
	Id int `json:"id"`
}

func (g GroupDeleted) IsNotification() {}

type IsRootAnswer struct {
	IsRoot bool
}

func (i IsRootAnswer) IsNotification() {}

type SubscriptionUpdated struct {
	GroupId          int              `json:"group_id"`
	Name             string           `json:"name"`
	Profiles         []Profile        `json:"profiles"`
	SubscriptionInfo SubscriptionInfo `json:"subscription_info"`
}

func (s SubscriptionUpdated) IsNotification() {}

type Warning struct {
	Key     string `json:"key"`
	Content string `json:"content"`
}

func (w Warning) IsNotification() {}

type Profile struct {
	Id         int    `json:"id"`
	GroupId    int    `json:"group_id"`
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	Uri        string `json:"uri"`
	Address    string `json:"address,omitzero"`
	Host       string `json:"host,omitzero"`
	TestResult int    `json:"test-result"`
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
	UpdateInterval int    `json:"update_interval,omitzero"`
	SupportUrl     string `json:"support_url,omitzero"`
	WebPageUrl     string `json:"web_page_url,omitzero"`
	Announce       string `json:"announce,omitzero"`
}
