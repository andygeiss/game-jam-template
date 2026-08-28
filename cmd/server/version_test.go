package main

import (
	"runtime/debug"
	"testing"
)

func buildInfo(version string, settings ...debug.BuildSetting) *debug.BuildInfo {
	info := &debug.BuildInfo{Settings: settings}
	info.Main.Version = version
	return info
}

func TestResolveVersion(t *testing.T) {
	t.Parallel()

	t.Run("a released version is used as it is", func(t *testing.T) {
		t.Parallel()
		if got := resolveVersion(buildInfo("v1.2.3")); got != "v1.2.3" {
			t.Errorf("got %q, want v1.2.3", got)
		}
	})

	t.Run("a clean checkout is named by its commit", func(t *testing.T) {
		t.Parallel()
		info := buildInfo("(devel)",
			debug.BuildSetting{Key: "vcs.revision", Value: "abcdef1234567890abcdef"},
			debug.BuildSetting{Key: "vcs.modified", Value: "false"},
		)
		if got := resolveVersion(info); got != "abcdef123456" {
			t.Errorf("got %q, want abcdef123456", got)
		}
	})

	// Every edited-tree shape gets a name that is not the toolchain's, because
	// the toolchain's repeats after every further edit.
	edited := map[string]*debug.BuildInfo{
		"a dirty tag": buildInfo("v1.2.3+dirty"),
		"a devel build with local edits": buildInfo("(devel)",
			debug.BuildSetting{Key: "vcs.revision", Value: "abcdef1234567890"},
			debug.BuildSetting{Key: "vcs.modified", Value: "true"},
		),
		"a devel build without VCS data": buildInfo("(devel)"),
		"no version at all":              buildInfo(""),
	}
	for name, info := range edited {
		t.Run(name+" gets a boot name", func(t *testing.T) {
			t.Parallel()
			got := resolveVersion(info)
			if got == "" || got == info.Main.Version {
				t.Errorf("got %q, want a boot name that is not %q", got, info.Main.Version)
			}
		})
	}
}
