package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build information, injected by the Makefile through -ldflags -X. A plain
// "go build" leaves the defaults in place.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the cdd version",
		Args:  cobra.NoArgs,
		Run: func(c *cobra.Command, _ []string) {
			fmt.Fprintln(c.OutOrStdout(), versionLine())
		},
	}
}

// versionLine formats the build information as "cdd <version> (<commit>, <date>)".
func versionLine() string {
	return fmt.Sprintf("cdd %s (%s, %s)", version, commit, date)
}
