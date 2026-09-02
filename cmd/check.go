package cmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jonasalessi/cdd-cli/internal/analyze"
	"github.com/jonasalessi/cdd-cli/internal/config"
	"github.com/jonasalessi/cdd-cli/internal/languages"
	"github.com/jonasalessi/cdd-cli/internal/report"
)

// Exit codes of "cdd check". Anything the command cannot do at all — a
// missing configuration, an unusable flag — leaves through the root command
// as exit code 1.
const (
	// exitViolations reports that at least one unit is above its limit and
	// the configured enforcement blocks on it.
	exitViolations = 1
	// exitTimeout reports that the time budget elapsed; the report printed
	// before it covers the files that were analyzed in time.
	exitTimeout = 2
)

func newCheckCmd() *cobra.Command {
	var all, explain bool
	c := &cobra.Command{
		Use:   "check",
		Short: "Measure the project and compare every unit with its limit",
		Long: `check runs the last two steps of CDD (docs/cdd.md, section 3):

  4. Compute the ICPs of every code unit the configured languages claim.
  5. Evaluate each unit against the ICP limit its file resolves to.

It reads the configuration named by --config (cdd.config.yaml by default) and
analyzes the tree rooted at that file's directory, so a configuration in a
subdirectory measures that subdirectory alone.

The report lists the units above their limit; --all lists every measured unit.
The summary counts the whole run either way.

--explain details whatever is listed, adding every counted construct with its
position and the ICPs it contributed, so an editor plugin can turn the json
report into inline hints.

Exit codes:

  0  no unit is above its limit, or the enforcement only reports them
  1  a unit is above its limit and the enforcement blocks on it, or the
     configuration could not be read or is invalid
  2  the timeout elapsed; the printed report covers the files analyzed in time`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			return runCheck(c, configPath, report.Options{All: all, Explain: explain})
		},
	}
	c.Flags().BoolVar(&all, "all", false, "list every unit, not only the ones over their limit")
	c.Flags().BoolVar(&explain, "explain", false, "list every counted construct with its position and ICPs")
	return c
}

// runCheck implements cdd check: load the configuration, analyze the tree it
// governs, report the outcome and turn it into an exit code.
func runCheck(c *cobra.Command, path string, opts report.Options) error {
	cfg, err := loadCheckConfig(c, path)
	if err != nil {
		return err
	}
	res, runErr := analyze.Run(commandContext(c), analyze.Request{
		Root:      filepath.Dir(path),
		Config:    cfg,
		Languages: languages.All(),
	})
	if runErr != nil && !errors.Is(runErr, analyze.ErrTimeout) {
		return runErr
	}
	if err := emitReport(c, cfg.Reporter, res, opts); err != nil {
		return err
	}
	if runErr != nil {
		fmt.Fprintf(c.ErrOrStderr(), "cdd: %v\n", runErr)
		return exitCodeError{code: exitTimeout}
	}
	return violationExit(res, cfg.Enforcement)
}

// loadCheckConfig reads path and validates it against the registry. Errors
// end the command; warnings only reach stderr, since the run is still the
// one the user asked for.
func loadCheckConfig(c *cobra.Command, path string) (*config.Config, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	issues := config.Validate(cfg, languages.Specs())
	if errs := issues.Errors(); len(errs) > 0 {
		return nil, fmt.Errorf("%s is invalid:\n%s", path, errs)
	}
	for _, warning := range issues.Warnings() {
		fmt.Fprintf(c.ErrOrStderr(), "warning: %s\n", warning)
	}
	return cfg, nil
}

// commandContext prefers the context cobra carries, so a caller that cancels
// the command also stops the analysis.
func commandContext(c *cobra.Command) context.Context {
	if ctx := c.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

// emitReport renders the run where the reporter points and prints the
// receipt when that was a file.
func emitReport(c *cobra.Command, r config.Reporter, res analyze.RunResult, opts report.Options) error {
	path, err := report.Emit(c.OutOrStdout(), r, res, opts)
	if err != nil {
		return err
	}
	if path != "" {
		fmt.Fprintf(c.OutOrStdout(), "Report written to %s\n", path)
	}
	return nil
}

// violationExit turns the reported violations into the command's exit code.
// The report is already out, so a blocking run only carries the code.
func violationExit(res analyze.RunResult, e config.Enforcement) error {
	if res.Violations() > 0 && e.Blocks() {
		return exitCodeError{code: exitViolations}
	}
	return nil
}
