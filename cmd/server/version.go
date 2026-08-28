package main

import (
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

// version is read once at boot: it is the boot log line, the /healthz field,
// and the cache-buster on every static asset.
var version = sync.OnceValue(func() string {
	info, ok := debug.ReadBuildInfo() // false outside module mode; reading the nil panics
	if !ok {
		return bootID()
	}
	return resolveVersion(info)
})

// resolveVersion picks the name for a build. It takes the build info rather
// than reading it, because the three cases it decides between cannot all be
// produced by the toolchain running the test.
func resolveVersion(info *debug.BuildInfo) string {
	// A real released version: go install of a tagged module. A tag built from
	// an edited tree is not one — "v0.3.0+dirty" is the same string after every
	// further edit, the exact thing this function exists to avoid — so it falls
	// through to the boot-by-boot name below.
	if v := info.Main.Version; v != "" && v != "(devel)" && !strings.HasSuffix(v, "+dirty") {
		return v
	}

	var revision string
	var edited bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			edited = s.Value == "true"
		}
	}
	// A clean checkout names its assets by the commit they came from, so two
	// deployments of the same commit do not make anyone download them twice.
	if revision != "" && !edited {
		return revision[:min(len(revision), 12)]
	}
	return bootID()
}

// bootID names one run of an edited tree, which is the finest grain there is:
// the toolchain cannot tell two edits apart, but it can tell two starts apart.
func bootID() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 36)
}
