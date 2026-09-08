package version //nolint:testpackage // Supply build metadata to the unexported formatting helper.

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionFromBuildInfo(t *testing.T) {
	const (
		unknownVersion     = "unknown"
		shortRevision      = "abcdef"
		developmentVersion = "(devel)"
		fullRevision       = "abcdef123456"
		modifiedValue      = "true"
		modifiedKey        = "vcs.modified"
		revisionKey        = "vcs.revision"
	)
	for _, tt := range []struct {
		name string
		info *debug.BuildInfo
		ok   bool
		want string
	}{
		{name: "unavailable build info", want: unknownVersion},
		{
			name: "installed release", ok: true,
			info: &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}},
			want: "v1.2.3",
		},
		{
			name: "release takes precedence over vcs", ok: true,
			info: &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3-rc.1"}, Settings: []debug.BuildSetting{
				{Key: revisionKey, Value: shortRevision}, {Key: modifiedKey, Value: modifiedValue},
			}},
			want: "v1.2.3-rc.1",
		},
		{
			name: "development without vcs", ok: true,
			info: &debug.BuildInfo{Main: debug.Module{Version: developmentVersion}},
			want: unknownVersion,
		},
		{
			name: "clean revision", ok: true,
			info: &debug.BuildInfo{Main: debug.Module{Version: developmentVersion}, Settings: []debug.BuildSetting{
				{Key: revisionKey, Value: fullRevision}, {Key: modifiedKey, Value: "false"},
			}},
			want: fullRevision,
		},
		{
			name: "dirty revision with reordered settings", ok: true,
			info: &debug.BuildInfo{Main: debug.Module{Version: developmentVersion}, Settings: []debug.BuildSetting{
				{Key: modifiedKey, Value: modifiedValue}, {Key: "GOOS", Value: "linux"},
				{Key: revisionKey, Value: fullRevision},
			}},
			want: "abcdef123456-dirty",
		},
		{
			name: "revision without modified setting", ok: true,
			info: &debug.BuildInfo{Main: debug.Module{Version: developmentVersion}, Settings: []debug.BuildSetting{
				{Key: revisionKey, Value: shortRevision},
			}},
			want: shortRevision,
		},
		{
			name: "dirty without revision", ok: true,
			info: &debug.BuildInfo{Main: debug.Module{Version: developmentVersion}, Settings: []debug.BuildSetting{
				{Key: modifiedKey, Value: modifiedValue},
			}},
			want: "unknown-dirty",
		},
		{
			name: "unrelated settings", ok: true,
			info: &debug.BuildInfo{Main: debug.Module{Version: developmentVersion}, Settings: []debug.BuildSetting{
				{Key: "GOARCH", Value: "amd64"}, {Key: "vcs.time", Value: "2026-01-01T00:00:00Z"},
			}},
			want: unknownVersion,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, versionFromBuildInfo(tt.info, tt.ok))
		})
	}
}

func TestGetVersionUsesBuildInfo(t *testing.T) {
	info, ok := debug.ReadBuildInfo()
	assert.Equal(t, versionFromBuildInfo(info, ok), GetVersion())
}
