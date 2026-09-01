package cmd

import (
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

// initOptions collects every flag of "cdd init". The full set is declared now
// so that --help stays stable once the command is implemented.
type initOptions struct {
	force             bool
	languages         []string
	projectType       string
	legacyMode        string
	limit             int
	metrics           []string
	weights           []string
	packages          []string
	noDefaultExcludes bool
	timeout           time.Duration
	scanTimeout       time.Duration
	yes               bool
	output            string
}

func newInitCmd() *cobra.Command {
	opts := &initOptions{}
	c := &cobra.Command{
		Use:   "init",
		Short: "Create a cdd.config.yaml for this project",
		Long: `init writes the configuration file that the other cdd commands read.

It walks through the first two steps of CDD (docs/cdd.md, section 3):

  1. Pick the ICP variables the team agrees to count, and their weights.
  2. Pick the ICP limit a code unit may not exceed, calibrated on whether the
     project is greenfield (7-14, default 10) or legacy (20-40, default 25).

Without flags the command asks each question interactively. Every answer can
also be given as a flag, and --yes skips the questions entirely.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			_ = opts // wired in Round 2
			return errors.New("init: not implemented yet")
		},
	}
	f := c.Flags()
	f.BoolVar(&opts.force, "force", false, "overwrite an existing configuration file")
	f.StringSliceVar(&opts.languages, "languages", nil, "languages to configure (go, java, kotlin, typescript)")
	f.StringVar(&opts.projectType, "project-type", "", "greenfield or legacy")
	f.StringVar(&opts.legacyMode, "legacy-mode", "", "enforcement mode for legacy projects")
	f.IntVar(&opts.limit, "limit", 0, "ICP limit applied to every language (0 = default for the project type)")
	f.StringSliceVar(&opts.metrics, "metrics", nil, "metric ids to enable (at least 3)")
	f.StringSliceVar(&opts.weights, "weight", nil, "metric weight override as id=value (repeatable)")
	f.StringSliceVar(&opts.packages, "packages", nil, "package prefixes considered internal to the project")
	f.BoolVar(&opts.noDefaultExcludes, "no-default-excludes", false, "do not exclude tests and generated code")
	f.DurationVar(&opts.timeout, "timeout", config.DefaultTimeout(), "analysis time budget written to the file")
	f.DurationVar(
		&opts.scanTimeout,
		"scan-timeout",
		config.DefaultScanTimeout(),
		"time budget for detecting languages and packages",
	)
	f.BoolVar(&opts.yes, "yes", false, "skip every prompt and use defaults or flags")
	f.StringVar(&opts.output, "output", "", "write the file here instead of --config")
	return c
}
