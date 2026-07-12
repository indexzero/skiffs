package cmd

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

// Version is the release version, injected via -ldflags by GoReleaser. It stays
// "dev" for `go build`/`go install`, where resolveVersion falls back to the
// module version embedded in the build info instead.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("skiffs %s\n", version())
	},
}

// version reports the resolved version, reading the embedded build info so that
// `go install …@v1.2.3` (or @latest) reports its module version rather than the
// "dev" default.
func version() string {
	build := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		build = info.Main.Version
	}
	return resolveVersion(Version, build)
}

// resolveVersion prefers an ldflags-injected version, then the module version
// from build info, and finally the "dev" default. The leading "v" is trimmed so
// a go-installed binary reports "1.2.3" to match GoReleaser's {{.Version}}.
func resolveVersion(ldflags, build string) string {
	if ldflags != "dev" {
		return ldflags
	}
	if build != "" && build != "(devel)" {
		return strings.TrimPrefix(build, "v")
	}
	return ldflags
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
