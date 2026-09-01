// Package initcmd is the UI-agnostic core of cdd init: it turns interview
// answers into a validated configuration and writes it atomically. Every
// default and coupling rule of the command lives here, so both the prompt
// flow and the flags stay thin.
package initcmd

import (
	"time"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

// Answers collects everything the interview or the flags decide. The zero
// value of a field means "use the default": Build fills it in.
type Answers struct {
	// Languages to configure; at least one is required.
	Languages []config.Language
	// ProjectType is greenfield or legacy; "" means greenfield.
	ProjectType string
	// LegacyMode is the enforcement mode; "" means the default for
	// ProjectType.
	LegacyMode string
	// Limit is the ICP limit for every language; 0 means
	// config.DefaultLimit(ProjectType).
	Limit int
	// Metrics are the selected metric ids; nil means
	// config.DefaultSelection().
	Metrics []config.MetricID
	// Weights overrides the default weight of selected metrics; a missing
	// id keeps config.DefaultWeight.
	Weights map[config.MetricID]float64
	// Packages are the internal package prefixes.
	Packages []string
	// DefaultExcludes adds config.DefaultExcludes per language when true.
	DefaultExcludes bool
	// Timeout is the analysis budget; 0 means config.DefaultTimeout().
	Timeout time.Duration
}
