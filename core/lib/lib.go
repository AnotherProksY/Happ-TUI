package lib

import (
	"encoding/json"
	"fmt"
	"github.com/AnotherProksY/Happ-TUI/core/db"
	"github.com/AnotherProksY/Happ-TUI/core/lib/subscription"
	"github.com/AnotherProksY/Happ-TUI/core/structs"
	"github.com/AnotherProksY/Happ-TUI/core/utils"
	"github.com/matoous/go-nanoid/v2"
	"log"
	"os/exec"
	"strings"
	"unicode/utf8"
)

type profileMetaData struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Address  string `json:"address,omitzero"`
	Host     string `json:"host,omitzero"`
}

func AddProfiles(DB *db.DB, data structs.AddProfilesData) structs.ProfilesAdded {
	profiles_data := GetDBAddProfileDatasFromStr(data.Uris, data.GroupId)
	var profiles []structs.Profile
	for _, profile_data := range profiles_data {
		profile_added, err := DB.AddProfile(profile_data)
		if err == nil {
			profiles = append(profiles, profile_added)
		}
	}
	return structs.ProfilesAdded{
		Profiles: profiles,
	}
}

func GetDBAddProfileDatasFromStr(str string, group_id int) []structs.DBAddProfileData {
	str = decode64(strings.TrimSpace(str))
	if strings.HasPrefix(str, "[") || strings.HasPrefix(str, "{") {
		profiles, err := getDBAddProfileDatasFromXrayJSON(str, group_id)
		if err == nil {
			return profiles
		}
		log.Println("failed to parse Xray JSON subscription:", err)
	}
	uris := strings.FieldsSeq(str)
	var profiles []structs.DBAddProfileData
	for uri := range uris {
		if strings.HasPrefix(uri, "#") || strings.HasPrefix(uri, "happ://") {
			continue
		}
		profile, err := getDBAddProfileDataFromURI(uri, group_id)
		if err != nil {
			continue
		}
		profiles = append(profiles, profile)
	}
	return profiles
}

func getDBAddProfileDataFromURI(uri string, group_id int) (structs.DBAddProfileData, error) {
	var profile_data structs.DBAddProfileData
	v2parserbin, err := utils.GetV2parserBin()
	if err != nil {
		return profile_data, fmt.Errorf("failed to parse uri: %w", err)
	}
	v2parser_metadata_cmd := exec.Command(v2parserbin, uri, "--get-metadata")
	metadata_output, err := v2parser_metadata_cmd.Output()
	if err != nil {
		return profile_data, fmt.Errorf("getting metadata failed: %w", err)
	}

	var profile_metadata profileMetaData
	if err := json.Unmarshal(metadata_output, &profile_metadata); err != nil {
		return profile_data, fmt.Errorf("err unmarshaling metadata output: %w", err)
	}

	profile_data = structs.DBAddProfileData{
		Protocol: profile_metadata.Protocol,
		Name:     profile_metadata.Name,
		Address:  profile_metadata.Address,
		Host:     profile_metadata.Host,
		Uri:      uri,
		GroupId:  group_id,
		NanoID:   generateNanoID(),
	}
	return profile_data, nil
}

func decode64(str string) string {
	decoded, ok := subscription.DecodeBase64(str)
	if !ok {
		return str
	}
	if utf8.ValidString(decoded) {
		return decoded
	}
	return str
}

func generateNanoID() string {
	return gonanoid.Must()
}
