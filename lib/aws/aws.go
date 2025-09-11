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
		if len(v.Keys()) == 0 { // Get only the sections having Keys
			continue
		}
		parts := strings.Split(v.Name(), " ")
		if len(parts) == 2 && parts[0] == "profile" { // skip default
			list = append(list, parts[1])
		}
	}
	return list, nil
}

// GetLocalBgdiAdminProfiles returns profiles related to sso role BgdiAdmin.
// For most use cases (like SSM Parameters) this is fine.
func GetLocalBgdiAdminProfiles() ([]string, error) {
	return GetLocalProfiles("BgdiAdmin")
}

// GetLocalSwissgeoAdminProfiles returns profiles related to sso role SwissgeoAdmin.
// For most use cases (like SSM Parameters) this is fine.
func GetLocalSwissgeoAdminProfiles() ([]string, error) {
	return GetLocalProfiles("SwissgeoAdmin")
}

// GetLocalAdminProfiles returns profiles related to sso role SwissgeoAdmin and BgdiAdmin.
// For most use cases (like SSM Parameters) this is fine.
func GetLocalAdminProfiles() ([]string, error) {
	bgdiProfiles, err := GetLocalProfiles("BgdiAdmin")
	if err != nil {
		return nil, err
	}
	swissgeoProfiles, err := GetLocalProfiles("SwissgeoAdmin")
	if err != nil {
		return nil, err
	}
	return append(bgdiProfiles, swissgeoProfiles...), nil
}
