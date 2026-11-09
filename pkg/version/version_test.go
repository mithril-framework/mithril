package version

import (
	"runtime/debug"
	"testing"
)

func TestResolve(t *testing.T) {
	cases := []struct {
		name string
		bi   *debug.BuildInfo
		ok   bool
		want string
	}{
		{"no build info", nil, false, Base},
		{"source build", &debug.BuildInfo{Main: debug.Module{Path: Module, Version: "(devel)"}}, true, Base},
		{"pseudo-version", &debug.BuildInfo{Main: debug.Module{Path: Module, Version: "v1.3.1-0.20261109000000-abcdef123456"}}, true, Base},
		{"go install @tag", &debug.BuildInfo{Main: debug.Module{Path: Module, Version: "v1.3.7"}}, true, "1.3.7"},
		{"library dependency", &debug.BuildInfo{Main: debug.Module{Path: "example.com/shop", Version: "(devel)"},
			Deps: []*debug.Module{{Path: "github.com/x/y", Version: "v9.9.9"}, {Path: Module, Version: "v1.4.2"}}}, true, "1.4.2"},
		{"replaced dependency", &debug.BuildInfo{Main: debug.Module{Path: "example.com/shop"},
			Deps: []*debug.Module{{Path: Module, Version: "v1.4.2", Replace: &debug.Module{Path: "../mithril"}}}}, true, Base},
	}
	for _, tc := range cases {
		if got := resolve(tc.bi, tc.ok); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
