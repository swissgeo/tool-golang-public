package aws

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"gopkg.in/ini.v1"
)

// GetLocalProfiles returns profiles according to $HOME/.aws/config
func GetLocalProfiles(ssoRole string) ([]string, error) {
	f, err := ini.Load(config.DefaultSharedConfigFilename()) // Load ini file
	if err != nil {
		return nil, err
	}
	list := []string{}
	for _, v := range f.Sections() {
		if role, e := v.GetKey("sso_role_name"); e != nil || role.Value() != ssoRole {
			continue // skip if sso_role_name not found or does not match
		}
		if len(v.Keys()) != 0 { // Get only the sections having Keys
			parts := strings.Split(v.Name(), " ")
			if len(parts) == 2 && parts[0] == "profile" { // skip default
				list = append(list, parts[1])
			}
		}
	}
	return list, nil
}

func GetLocalBgdiAdminProfiles() ([]string, error) {
	return GetLocalProfiles("BgdiAdmin")
}
