package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/jonasalessi/cdd-cli/internal/config"
	"github.com/jonasalessi/cdd-cli/internal/detect"
	"github.com/jonasalessi/cdd-cli/internal/initcmd"
	"github.com/jonasalessi/cdd-cli/internal/prompt"
)

// exitCtrlC is the conventional exit code for a run ended by ctrl-c.
const exitCtrlC = 130

// initOptions collects every flag of "cdd init".
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
		RunE: func(c *cobra.Command, _ []string) error {
			return runInit(c, opts)
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

// runInit implements cdd init: guard the target file, detect defaults, ask
// or read the answers, then build and write the configuration.
func runInit(c *cobra.Command, opts *initOptions) error {
	path := opts.output
	if path == "" {
		path = configPath
	}
	interactive := !opts.yes && term.IsTerminal(int(os.Stdin.Fd()))
	force, done, err := guardExisting(c, path, opts.force, interactive)
	if err != nil || done {
		return err
	}
	answers, det, err := gatherDefaults(opts)
	if err != nil {
		return err
	}
	if det.Truncated && !interactive {
		fmt.Fprintf(c.ErrOrStderr(), "warning: language scan stopped after %s; detection may be incomplete\n",
			opts.scanTimeout)
	}
	if interactive {
		if answers, err = prompt.Run(answers, det); err != nil {
			if errors.Is(err, prompt.ErrAborted) {
				return exitCodeError{code: exitCtrlC}
			}
			return err
		}
	} else if len(answers.Languages) == 0 {
		return errors.New("no languages detected; pass --languages to pick them")
	}
	return writeConfig(c, answers, path, force)
}

// guardExisting decides what happens when path is already there. done means
// the user declined the overwrite and the command ends successfully.
func guardExisting(c *cobra.Command, path string, force, interactive bool) (effectiveForce, done bool, err error) {
	if force {
		return true, false, nil
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, false, nil
		}
		return false, false, err
	}
	if !interactive {
		return false, false, fmt.Errorf("%s exists; pass --force to overwrite", path)
	}
	ok, err := prompt.ConfirmOverwrite(path)
	if err != nil {
		if errors.Is(err, prompt.ErrAborted) {
			return false, false, exitCodeError{code: exitCtrlC}
		}
		return false, false, err
	}
	if !ok {
		fmt.Fprintln(c.OutOrStdout(), "aborted")
		return false, true, nil
	}
	return true, false, nil
}

// gatherDefaults runs language and package detection and folds the flags in;
// a flag always wins over a detected value.
func gatherDefaults(opts *initOptions) (initcmd.Answers, detect.Detected, error) {
	ctx := context.Background()
	if opts.scanTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.scanTimeout)
		defer cancel()
	}
	det, err := detect.Languages(ctx, ".")
	if err != nil {
		return initcmd.Answers{}, det, err
	}
	weights, err := parseWeightFlags(opts.weights)
	if err != nil {
		return initcmd.Answers{}, det, err
	}
	a := initcmd.Answers{
		Languages:       toLanguages(opts.languages),
		ProjectType:     opts.projectType,
		LegacyMode:      opts.legacyMode,
		Limit:           opts.limit,
		Metrics:         toMetrics(opts.metrics),
		Weights:         weights,
		Packages:        opts.packages,
		DefaultExcludes: !opts.noDefaultExcludes,
		Timeout:         opts.timeout,
	}
	if len(a.Languages) == 0 {
		a.Languages = det.Languages()
	}
	if len(a.Packages) == 0 {
		if a.Packages, err = detect.Packages(".", a.Languages); err != nil {
			return a, det, err
		}
	}
	return a, det, nil
}

// writeConfig runs the use case tail: build, report warnings, write, and
// print the one-line receipt.
func writeConfig(c *cobra.Command, a initcmd.Answers, path string, force bool) error {
	cfg, warnings, err := initcmd.Build(a)
	if err != nil {
		return err
	}
	for _, warning := range warnings {
		fmt.Fprintf(c.ErrOrStderr(), "warning: %s\n", warning)
	}
	if err := initcmd.Write(cfg, path, force); err != nil {
		if errors.Is(err, initcmd.ErrExists) {
			return fmt.Errorf("%s exists; pass --force to overwrite", path)
		}
		return err
	}
	fmt.Fprintln(c.OutOrStdout(), summary(path, cfg))
	return nil
}

// summary is the receipt printed after a successful write, e.g.
// "Created cdd.config.yaml — languages: go · project: greenfield · limit: 10 · metrics: 4".
func summary(path string, cfg *config.Config) string {
	var langs []string
	limit := 0
	ids := map[config.MetricID]bool{}
	for _, lang := range config.Languages() {
		patterns, ok := cfg.Metrics[lang]
		if !ok {
			continue
		}
		langs = append(langs, string(lang))
		for id := range patterns[0].Weights {
			ids[id] = true
		}
		limit = cfg.ICPLimits[lang][0].Limit
	}
	return fmt.Sprintf("Created %s — languages: %s · project: %s · limit: %d · metrics: %d",
		path, strings.Join(langs, ", "), cfg.ProjectType, limit, len(ids))
}

// toLanguages converts the --languages values, trimming stray spaces.
func toLanguages(ids []string) []config.Language {
	var out []config.Language
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, config.Language(id))
		}
	}
	return out
}

// toMetrics converts the --metrics values, trimming stray spaces.
func toMetrics(ids []string) []config.MetricID {
	var out []config.MetricID
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, config.MetricID(id))
		}
	}
	return out
}

// parseWeightFlags turns --weight id=value pairs into the override map.
func parseWeightFlags(pairs []string) (map[config.MetricID]float64, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	out := make(map[config.MetricID]float64, len(pairs))
	for _, pair := range pairs {
		id, raw, ok := strings.Cut(pair, "=")
		if !ok {
			return nil, fmt.Errorf("--weight %q: expected id=value", pair)
		}
		weight, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return nil, fmt.Errorf("--weight %q: %q is not a number", pair, raw)
		}
		out[config.MetricID(strings.TrimSpace(id))] = weight
	}
	return out, nil
}
