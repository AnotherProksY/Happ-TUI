package subscription

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AnotherProksY/Happ-TUI/core/lib/config"
	"github.com/AnotherProksY/Happ-TUI/core/structs"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Result is the downloaded subscription body together with the metadata the
// provider sent in Happ-style response headers.
type Result struct {
	Body string
	Info structs.SubscriptionInfo
}

// NewRequest builds a subscription request that looks like the configured client.
// providerUserAgent is the user-agent the provider asked for with a
// Change-User-Agent header on a previous update, if any.
func NewRequest(url string, appConfig config.AppConfig, providerUserAgent string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if appConfig.SubscriptionClient == config.SubscriptionClientPlain {
		if appConfig.SubscriptionUserAgent == "" {
			req.Header["User-Agent"] = nil
		} else {
			req.Header.Set("User-Agent", appConfig.SubscriptionUserAgent)
		}
	} else {
		device := GetDeviceInfo(appConfig.HWID)
		userAgent := appConfig.SubscriptionUserAgent
		if userAgent == "" {
			userAgent = providerUserAgent
		}
		if userAgent == "" {
			userAgent = "Happ/" + HappVersion
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("X-Hwid", device.HWID)
		req.Header.Set("X-Device-Os", device.OS)
		req.Header.Set("X-Ver-Os", device.OSVersion)
		req.Header.Set("X-Device-Model", device.Model)
		req.Header.Set("X-Device-Locale", device.Locale)
		req.Header.Set("Accept", "*/*")
	}

	for name, value := range appConfig.SubscriptionHeaders {
		if value == "" {
			req.Header.Del(name)
			if http.CanonicalHeaderKey(name) == "User-Agent" {
				req.Header["User-Agent"] = nil
			}
			continue
		}
		req.Header.Set(name, value)
	}

	return req, nil
}

// Fetch downloads a subscription. Non-2xx responses are reported as errors so
// that an error page never replaces the profiles of a group.
func Fetch(url string, appConfig config.AppConfig, providerUserAgent string) (Result, error) {
	var result Result
	req, err := NewRequest(url, appConfig, providerUserAgent)
	if err != nil {
		return result, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return result, statusError(resp)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	result.Body = string(bodyBytes)
	result.Info = ParseInfo(resp.Header)
	return result, nil
}

func statusError(resp *http.Response) error {
	reason := ""
	switch {
	case resp.Header.Get("X-Hwid-Max-Devices-Reached") == "true" || resp.Header.Get("X-Hwid-Limit") == "true":
		reason = ": device limit reached"
	case resp.Header.Get("X-Hwid-Not-Supported") == "true":
		reason = ": provider requires HWID"
	}
	return fmt.Errorf("subscription server returned %s%s", resp.Status, reason)
}

// ParseInfo reads the response headers Happ understands.
func ParseInfo(header http.Header) structs.SubscriptionInfo {
	info := structs.SubscriptionInfo{
		Title:      decodeHeaderText(header.Get("Profile-Title")),
		SupportUrl: decodeHeaderText(header.Get("Support-Url")),
		Announce:   decodeHeaderText(header.Get("Announce")),
		WebPageUrl: decodeHeaderText(header.Get("Profile-Web-Page-Url")),

		ChangeUserAgent: strings.TrimSpace(header.Get("Change-User-Agent")),
		CheckUrl:        strings.TrimSpace(header.Get("Check-Url-Via-Proxy")),
	}

	if hours, err := strconv.ParseFloat(strings.TrimSpace(header.Get("Profile-Update-Interval")), 64); err == nil && hours > 0 {
		info.UpdateInterval = int(hours * 3600)
	}

	for part := range strings.SplitSeq(header.Get("Subscription-Userinfo"), ";") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		number, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "upload":
			info.Upload = number
		case "download":
			info.Download = number
		case "total":
			info.Total = number
		case "expire":
			info.Expire = number
		}
	}

	return info
}

// decodeHeaderText decodes values that may be sent as "base64:<data>".
func decodeHeaderText(value string) string {
	value = strings.TrimSpace(value)
	encoded, ok := strings.CutPrefix(value, "base64:")
	if !ok {
		return value
	}
	if decoded, ok := DecodeBase64(encoded); ok {
		return strings.TrimSpace(decoded)
	}
	return value
}

// DecodeBase64 accepts standard and URL-safe base64, padded or not.
func DecodeBase64(str string) (string, bool) {
	str = strings.Join(strings.Fields(str), "")
	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		if decoded, err := encoding.DecodeString(str); err == nil {
			return string(decoded), true
		}
	}
	return "", false
}
