package main

import (
	"runtime/debug"
	"testing"
)

// TestBuildVersion pins the version contract: proxy builds report the
// released module version consumers gate on; source builds report the VCS
// revision; builds with no identity say "devel" instead of masquerading as a
// release.
func TestBuildVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		bi   *debug.BuildInfo
		want string
	}{
		{
			name: "proxy build reports the module version",
			bi:   &debug.BuildInfo{Main: debug.Module{Version: "v0.2.1"}},
			want: "v0.2.1",
		},
		{
			name: "source build reports the short revision",
			bi: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "d428c1e000000000000000000000000000000000"},
					{Key: "vcs.time", Value: "2026-09-20T10:00:00Z"},
					{Key: "vcs.modified", Value: "false"},
				},
			},
			want: "devel+d428c1e",
		},
		{
			name: "dirty source build says so",
			bi: &debug.BuildInfo{
				Main: debug.Module{Version: "devel"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "d428c1e000000000000000000000000000000000"},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			want: "devel+d428c1e.dirty",
		},
		{
			name: "no build identity falls back honestly",
			bi:   &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			want: "devel",
		},
		{
			name: "empty version falls back honestly",
			bi:   &debug.BuildInfo{},
			want: "devel",
		},
		{
			name: "nil build info falls back honestly",
			bi:   nil,
			want: "devel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := buildVersion(tt.bi); got != tt.want {
				t.Errorf("buildVersion = %q, want %q", got, tt.want)
			}
		})
	}
}
