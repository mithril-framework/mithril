// Package version is the framework and API release version.
package version

import (
	"regexp"
	"runtime/debug"
	"strings"
)

// Module is the framework's module path.
const Module = "github.com/mithril-framework/mithril"

// Base is the version declared in source. The release workflow publishes Base when its tag does
// not exist yet, and otherwise the next patch after the newest tag, so bump Base for a minor or
// major release.
const Base = "1.3.0"

// Version is the running framework version, without the leading "v". Release binaries set it with
// -ldflags "-X github.com/mithril-framework/mithril/pkg/version.Version=1.2.3". Otherwise it comes
// from the module's build info (`go install ...@v1.2.3`, or an app that requires that version),
// falling back to Base for source builds.
var Version string

var release = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

func init() {
	if Version == "" {
		Version = resolve(debug.ReadBuildInfo())
	}
}

func resolve(bi *debug.BuildInfo, ok bool) string {
	if ok {
		mods := append([]*debug.Module{&bi.Main}, bi.Deps...)
		for _, m := range mods {
			if m == nil || m.Path != Module {
				continue
			}
			if m.Replace == nil && release.MatchString(m.Version) {
				return strings.TrimPrefix(m.Version, "v")
			}
			break
		}
	}
	return Base
}
