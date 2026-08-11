package cmd

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	t.Parallel()
	buildInfo := func(module string, settings ...debug.BuildSetting) *debug.BuildInfo {
		info := &debug.BuildInfo{Settings: settings}
		info.Main.Version = module
		return info
	}
	revision := debug.BuildSetting{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"}

	tests := []struct {
		name    string
		stamped string
		info    *debug.BuildInfo
		ok      bool
		want    string
	}{
		{
			name:    "build-time stamp wins",
			stamped: "v1.2.3",
			info:    buildInfo("v9.9.9"),
			ok:      true,
			want:    "v1.2.3",
		},
		{
			name: "module version from go install",
			info: buildInfo("v0.4.0"),
			ok:   true,
			want: "v0.4.0",
		},
		{
			name: "commit from a repository build",
			info: buildInfo("(devel)", revision),
			ok:   true,
			want: "0123456789ab",
		},
		{
			name: "uncommitted repository build",
			info: buildInfo("(devel)", revision, debug.BuildSetting{Key: "vcs.modified", Value: "true"}),
			ok:   true,
			want: "0123456789ab+dirty",
		},
		{
			name: "no version information at all",
			info: buildInfo("(devel)"),
			ok:   true,
			want: "unknown",
		},
		{
			name: "build information unavailable",
			ok:   false,
			want: "unknown",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveVersion(test.stamped, test.info, test.ok); got != test.want {
				t.Errorf("resolveVersion() = %q, want %q", got, test.want)
			}
		})
	}
}
