// Package config defines the cdd.config.yaml contract (schema v1): the Go
// types, the vocabulary of ids the file may use, and the three operations
// every command needs on it: Render, Load and Validate.
package config

import "time"

// Language is a canonical language id, see LanguageSpec.
type Language string

// MetricID is a canonical ICP metric id, see Metrics.
type MetricID string

// Config mirrors cdd.config.yaml. Map keys are languages; the pattern lists
// inside keep document order because the last matching pattern wins.
type Config struct {
	Version          int                         `yaml:"version"`
	ProjectType      string                      `yaml:"project_type"`
	Metrics          map[Language]PatternWeights `yaml:"metrics"`
	ICPLimits        map[Language]PatternLimits  `yaml:"icp-limits"`
	Enforcement      Enforcement                 `yaml:"enforcement"`
	Timeout          time.Duration               `yaml:"timeout"`
	Reporter         Reporter                    `yaml:"reporter"`
	InternalCoupling InternalCoupling            `yaml:"internal_coupling"`
	Include          []string                    `yaml:"include"`
	Exclude          []string                    `yaml:"exclude"`
}

// PatternWeights is the ordered list of file patterns and their metric
// weights for one language.
type PatternWeights []PatternWeight

// PatternWeight assigns weights to the files matching Pattern. A metric that
// is absent from Weights is disabled for those files.
type PatternWeight struct {
	Pattern string
	Weights map[MetricID]float64
}

// PatternLimits is the ordered list of file patterns and their ICP limit for
// one language.
type PatternLimits []PatternLimit

// PatternLimit is the ICP limit for the units inside files matching Pattern.
type PatternLimit struct {
	Pattern string
	Limit   int
}

// Enforcement says what happens when a unit is above its limit.
type Enforcement struct {
	BlockOnCI  bool   `yaml:"block_on_ci"`
	LegacyMode string `yaml:"legacy_mode"`
}

// Blocks reports whether a violation must fail the run. Only strict_all is
// enforced today: strict_on_new_only and boy_scout need the git history and
// the baseline store, neither of which exists yet, and measure_only never
// blocks. A run whose block_on_ci is set while Blocks is false reports the
// violations without failing on them.
func (e Enforcement) Blocks() bool {
	return e.BlockOnCI && e.LegacyMode == ModeStrictAll
}

// Reporter selects the report format and destination. A nil OutputFile
// means stdout.
type Reporter struct {
	Format     string  `yaml:"format"`
	OutputFile *string `yaml:"outputFile"`
}

// InternalCoupling defines which package prefixes count as internal.
type InternalCoupling struct {
	AutoDetect bool     `yaml:"auto_detect"`
	Packages   []string `yaml:"packages"`
}
