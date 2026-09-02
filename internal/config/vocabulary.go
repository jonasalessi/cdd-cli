package config

import (
	"slices"
	"time"
)

// SchemaVersion is the only value "version" may hold.
const SchemaVersion = 1

// MinMetrics is the smallest number of metrics a language must count
// (docs/cdd.md section 2).
const MinMetrics = 3

// PatternAll matches every file and must open every pattern list.
const PatternAll = ".*"

// Canonical metric ids.
const (
	MetricCodeBranch        MetricID = "code_branch"
	MetricCondition         MetricID = "condition"
	MetricExceptionHandling MetricID = "exception_handling"
	MetricInternalCoupling  MetricID = "internal_coupling"
	MetricExternalCoupling  MetricID = "external_coupling"
	MetricInheritance       MetricID = "inheritance"
	MetricLocalVariable     MetricID = "local_variable"
	MetricLambda            MetricID = "lambda"
)

// Project types.
const (
	ProjectGreenfield = "greenfield"
	ProjectLegacy     = "legacy"
)

// Legacy modes.
const (
	ModeStrictAll       = "strict_all"
	ModeStrictOnNewOnly = "strict_on_new_only"
	ModeBoyScout        = "boy_scout"
	ModeMeasureOnly     = "measure_only"
)

// Reporter formats.
const (
	FormatConsole  = "console"
	FormatJSON     = "json"
	FormatXML      = "xml"
	FormatMarkdown = "markdown"
)

// RegexPrefix marks an include/exclude entry as a regular expression; every
// other entry is a glob.
const RegexPrefix = "regex:"

// GlobPrefix is the optional, default prefix of an include/exclude entry.
const GlobPrefix = "glob:"

var metrics = []MetricID{
	MetricCodeBranch,
	MetricCondition,
	MetricExceptionHandling,
	MetricInternalCoupling,
	MetricExternalCoupling,
	MetricInheritance,
	MetricLocalVariable,
	MetricLambda,
}

var defaultWeights = map[MetricID]float64{
	MetricExternalCoupling: 0.5,
	MetricLocalVariable:    0.5,
}

var defaultSelection = []MetricID{
	MetricCodeBranch,
	MetricCondition,
	MetricExceptionHandling,
	MetricInternalCoupling,
	MetricExternalCoupling,
	MetricInheritance,
}

var projectTypes = []string{ProjectGreenfield, ProjectLegacy}

type limitBand struct{ def, lo, hi int }

var limitBands = map[string]limitBand{
	ProjectGreenfield: {def: 10, lo: 7, hi: 14},
	ProjectLegacy:     {def: 25, lo: 20, hi: 40},
}

var legacyModes = []string{ModeStrictAll, ModeStrictOnNewOnly, ModeBoyScout, ModeMeasureOnly}

var reporterFormats = []string{FormatConsole, FormatJSON, FormatXML, FormatMarkdown}

// descriptions holds the generic inline comment written next to each metric
// weight; a LanguageSpec may override it with its own wording.
var descriptions = map[MetricID]string{
	MetricCodeBranch:        "if/else, switch case, ternary, loops",
	MetricCondition:         "&& and || clauses",
	MetricExceptionHandling: "try / catch / finally blocks",
	MetricInternalCoupling:  "references to project packages",
	MetricExternalCoupling:  "framework / stdlib types",
	MetricInheritance:       "extends / implements, per level",
	MetricLocalVariable:     "locals and fields",
	MetricLambda:            "lambdas and method refs",
}

// Metrics returns every metric id in canonical order.
func Metrics() []MetricID {
	return append([]MetricID(nil), metrics...)
}

// IsMetric reports whether m is a known metric id.
func IsMetric(m MetricID) bool {
	return slices.Contains(metrics, m)
}

// DefaultWeight returns the weight docs/cdd.md suggests for m: 0.5 for
// external_coupling and local_variable, 1.0 for everything else.
func DefaultWeight(m MetricID) float64 {
	if w, ok := defaultWeights[m]; ok {
		return w
	}
	return 1.0
}

// DefaultSelection returns the metrics enabled by default. local_variable
// and lambda are opt-in.
func DefaultSelection() []MetricID {
	return append([]MetricID(nil), defaultSelection...)
}

// MetricDescription returns the generic inline comment for m. A language
// with its own wording overrides it through LanguageSpec.Description.
func MetricDescription(m MetricID) string {
	return descriptions[m]
}

// ProjectTypes returns the allowed project_type values.
func ProjectTypes() []string {
	return append([]string(nil), projectTypes...)
}

// IsProjectType reports whether pt is a known project type.
func IsProjectType(pt string) bool {
	return slices.Contains(projectTypes, pt)
}

// DefaultLimit returns the ICP limit cdd init proposes for pt: 10 for
// greenfield, 25 for legacy. Unknown types get the greenfield value.
func DefaultLimit(pt string) int {
	return bandFor(pt).def
}

// LimitBand returns the inclusive limit range docs/cdd.md recommends for pt:
// 7-14 for greenfield, 20-40 for legacy. A limit outside the band is only a
// warning.
func LimitBand(pt string) (lo, hi int) {
	b := bandFor(pt)
	return b.lo, b.hi
}

func bandFor(pt string) limitBand {
	if b, ok := limitBands[pt]; ok {
		return b
	}
	return limitBands[ProjectGreenfield]
}

// LegacyModes returns the allowed legacy_mode values.
func LegacyModes() []string {
	return append([]string(nil), legacyModes...)
}

// IsLegacyMode reports whether mode is a known legacy mode.
func IsLegacyMode(mode string) bool {
	return slices.Contains(legacyModes, mode)
}

// ReporterFormats returns the allowed reporter.format values.
func ReporterFormats() []string {
	return append([]string(nil), reporterFormats...)
}

// IsReporterFormat reports whether f is a known reporter format.
func IsReporterFormat(f string) bool {
	return slices.Contains(reporterFormats, f)
}

// DefaultTimeout is the analysis budget written to a new file.
func DefaultTimeout() time.Duration {
	return 5 * time.Minute
}

// DefaultScanTimeout is the budget cdd init spends detecting languages and
// packages. It is not part of the schema.
func DefaultScanTimeout() time.Duration {
	return 4 * time.Second
}
