package cmd

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

// stubBuildInfo makes readBuildInfo return info until the test ends. A nil
// info stands for a binary the Go tool stamped nothing into.
func stubBuildInfo(t *testing.T, info *debug.BuildInfo) {
	t.Helper()
	previous := readBuildInfo
	readBuildInfo = func() (*debug.BuildInfo, bool) { return info, info != nil }
	t.Cleanup(func() { readBuildInfo = previous })
}

// stubLDFlags sets the values the Makefile injects until the test ends.
func stubLDFlags(t *testing.T, v, c, d string) {
	t.Helper()
	pv, pc, pd := version, commit, date
	version, commit, date = v, c, d
	t.Cleanup(func() { version, commit, date = pv, pc, pd })
}

func vcsInfo(mainVersion, revision, when string) *debug.BuildInfo {
	return &debug.BuildInfo{
		Main: debug.Module{Version: mainVersion},
		Settings: []debug.BuildSetting{
			{Key: "-compiler", Value: "gc"},
			{Key: "vcs.revision", Value: revision},
			{Key: "vcs.time", Value: when},
		},
	}
}

func TestVersionLine(t *testing.T) {
	tests := []struct {
		name    string
		v, c, d string
		info    *debug.BuildInfo
		want    string
	}{
		{
			name: "no ldflags and no build information",
			v:    defaultVersion, c: defaultCommit, d: defaultDate,
			want: "cdd dev",
		},
		{
			name: "ldflags carry everything",
			v:    "v1.2.3", c: "abc1234", d: "2026-01-02T03:04:05Z",
			info: vcsInfo(develVersion, "0000000000000000000000000000000000000000", "2020-01-01T00:00:00Z"),
			want: "cdd v1.2.3 (abc1234, 2026-01-02T03:04:05Z)",
		},
		{
			name: "go install of a published version",
			v:    defaultVersion, c: defaultCommit, d: defaultDate,
			info: &debug.BuildInfo{Main: debug.Module{Version: "v0.1.0"}},
			want: "cdd v0.1.0",
		},
		{
			name: "go install resolved to a pseudo-version",
			v:    defaultVersion, c: defaultCommit, d: defaultDate,
			info: &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0-20260903134529-1304014670f6"}},
			want: "cdd v0.0.0-20260903134529-1304014670f6",
		},
		{
			name: "plain go build inside the repository",
			v:    defaultVersion, c: defaultCommit, d: defaultDate,
			info: vcsInfo(develVersion, "1304014670f688a79501941f665c5469f0302611", "2026-09-03T13:45:29Z"),
			want: "cdd dev (1304014, 2026-09-03T13:45:29Z)",
		},
		{
			name: "revision shorter than the printed width",
			v:    defaultVersion, c: defaultCommit, d: defaultDate,
			info: vcsInfo(develVersion, "130401", "2026-09-03T13:45:29Z"),
			want: "cdd dev (130401, 2026-09-03T13:45:29Z)",
		},
		{
			name: "date alone",
			v:    defaultVersion, c: defaultCommit, d: "2026-09-03T13:45:29Z",
			want: "cdd dev (2026-09-03T13:45:29Z)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubLDFlags(t, tt.v, tt.c, tt.d)
			stubBuildInfo(t, tt.info)
			assert.Equal(t, tt.want, versionLine())
		})
	}
}
