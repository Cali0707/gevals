package cli

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
)

// Set via ldflags at build time:
// go build -ldflags "-X github.com/mcpchecker/mcpchecker/pkg/cli.Version=v1.0.0 -X github.com/mcpchecker/mcpchecker/pkg/cli.Commit=$(git rev-parse --short HEAD)"
var (
	Version = "development"
	Commit  = ""
)

var semverRegex = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("mcpchecker version %s\n", version())
		},
	}
}

func version() string {
	// Show commit for dev builds, but not for clean releases (vX.Y.Z)
	if Commit != "" && !semverRegex.MatchString(Version) {
		return fmt.Sprintf("%s@%s", Version, Commit)
	} else {
		return Version
	}
}
