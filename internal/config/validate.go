package config

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Issue severities.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

// Rule ids. Each one is a line of the schema contract:
//
//	V1  version is 1
//	V2  project_type is greenfield or legacy
//	V3  metrics has at least one language and every language key is known
//	V4  every metric id is known and applicable to its language
//	V5  every weight is > 0
//	V6  every limit is >= 1 (error); inside the band of project_type (warning)
//	V7  every pattern is a valid RE2 regex; include/exclude entries are not
//	    empty and "regex:" entries compile
//	V8  every pattern list opens with ".*"; the ".*" weights count at least
//	    MinMetrics metrics
//	V9  legacy_mode is known; greenfield needs strict_all; measure_only needs
//	    block_on_ci false; boy_scout is not supported yet (warning)
//	V10 reporter.format is known
//	V11 metrics and icp-limits configure the same languages
//	V12 timeout is >= 0
const (
	RuleVersion      = "V1"
	RuleProjectType  = "V2"
	RuleLanguages    = "V3"
	RuleMetricIDs    = "V4"
	RuleWeights      = "V5"
	RuleLimits       = "V6"
	RulePatterns     = "V7"
	RuleDefaultEntry = "V8"
	RuleLegacyMode   = "V9"
	RuleReporter     = "V10"
	RuleLanguageSets = "V11"
	RuleTimeout      = "V12"
)

// Issue is one finding of Validate.
type Issue struct {
	Rule     string
	Severity string
	Message  string
}

func (i Issue) String() string {
	return fmt.Sprintf("%s [%s]: %s", i.Rule, i.Severity, i.Message)
}

// Issues is the aggregated result of Validate.
type Issues []Issue

// HasErrors reports whether at least one issue has error severity.
func (issues Issues) HasErrors() bool {
	for _, i := range issues {
		if i.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Errors returns the issues with error severity.
func (issues Issues) Errors() Issues {
	return issues.bySeverity(SeverityError)
}

// Warnings returns the issues with warning severity.
func (issues Issues) Warnings() Issues {
	return issues.bySeverity(SeverityWarning)
}

func (issues Issues) bySeverity(s string) Issues {
	var out Issues
	for _, i := range issues {
		if i.Severity == s {
			out = append(out, i)
		}
	}
	return out
}

func (issues Issues) String() string {
	lines := make([]string, len(issues))
	for i, issue := range issues {
		lines[i] = issue.String()
	}
	return strings.Join(lines, "\n")
}

// Validate checks cfg against rules V1 to V12 and returns every finding, so
// a user sees all problems in one run. specs are the known languages; a nil
// cfg is one V1 error.
func Validate(cfg *Config, specs []LanguageSpec) Issues {
	if cfg == nil {
		return Issues{{Rule: RuleVersion, Severity: SeverityError, Message: "no configuration"}}
	}
	v := &validator{specs: specs}
	v.version(cfg)
	v.projectType(cfg)
	v.languages(cfg)
	v.metrics(cfg)
	v.limits(cfg)
	v.filters(cfg)
	v.enforcement(cfg)
	v.reporter(cfg)
	v.languageSets(cfg)
	v.timeout(cfg)
	return v.issues
}

type validator struct {
	specs  []LanguageSpec
	issues Issues
}

// isLanguage reports whether lang is one of the known specs.
func (v *validator) isLanguage(lang Language) bool {
	_, ok := FindSpec(v.specs, lang)
	return ok
}

// isApplicable reports whether id can be counted for lang. An unknown
// language, already reported by V3, applies every known metric.
func (v *validator) isApplicable(lang Language, id MetricID) bool {
	spec, ok := FindSpec(v.specs, lang)
	if !ok {
		return IsMetric(id)
	}
	return spec.IsApplicable(id)
}

func (v *validator) errorf(rule, format string, args ...any) {
	v.issues = append(v.issues, Issue{Rule: rule, Severity: SeverityError, Message: fmt.Sprintf(format, args...)})
}

func (v *validator) warnf(rule, format string, args ...any) {
	v.issues = append(v.issues, Issue{Rule: rule, Severity: SeverityWarning, Message: fmt.Sprintf(format, args...)})
}

func (v *validator) version(cfg *Config) {
	if cfg.Version != SchemaVersion {
		v.errorf(RuleVersion, "version: got %d, want %d", cfg.Version, SchemaVersion)
	}
}

func (v *validator) projectType(cfg *Config) {
	if !IsProjectType(cfg.ProjectType) {
		v.errorf(RuleProjectType, "project_type: %q is not one of %s", cfg.ProjectType, join(ProjectTypes()))
	}
}

func (v *validator) languages(cfg *Config) {
	if len(cfg.Metrics) == 0 {
		v.errorf(RuleLanguages, "metrics: at least one language is required")
	}
	for _, lang := range sortedLanguages(cfg.Metrics, v.specs) {
		if !v.isLanguage(lang) {
			v.errorf(RuleLanguages, "metrics: %q is not one of %s", lang, join(LanguageIDs(v.specs)))
		}
	}
	for _, lang := range sortedLanguages(cfg.ICPLimits, v.specs) {
		if !v.isLanguage(lang) {
			v.errorf(RuleLanguages, "icp-limits: %q is not one of %s", lang, join(LanguageIDs(v.specs)))
		}
	}
}

func (v *validator) metrics(cfg *Config) {
	for _, lang := range sortedLanguages(cfg.Metrics, v.specs) {
		patterns := cfg.Metrics[lang]
		v.defaultEntry("metrics", lang, len(patterns), func(i int) string { return patterns[i].Pattern })
		for i, p := range patterns {
			v.pattern("metrics", lang, p.Pattern)
			if i == 0 && p.Pattern == PatternAll && len(p.Weights) < MinMetrics {
				v.errorf(
					RuleDefaultEntry,
					"metrics.%s.%q: %d metrics configured, at least %d are required",
					lang,
					p.Pattern,
					len(p.Weights),
					MinMetrics,
				)
			}
			for _, id := range sortedMetrics(p.Weights) {
				switch {
				case !IsMetric(id):
					v.errorf(RuleMetricIDs, "metrics.%s.%q: %q is not one of %s", lang, p.Pattern, id, join(Metrics()))
				case !v.isApplicable(lang, id):
					v.errorf(RuleMetricIDs, "metrics.%s.%q: %s does not apply to %s", lang, p.Pattern, id, lang)
				}
				if w := p.Weights[id]; w <= 0 {
					v.errorf(RuleWeights, "metrics.%s.%q.%s: weight %v must be > 0", lang, p.Pattern, id, w)
				}
			}
		}
	}
}

func (v *validator) limits(cfg *Config) {
	lo, hi := LimitBand(cfg.ProjectType)
	for _, lang := range sortedLanguages(cfg.ICPLimits, v.specs) {
		patterns := cfg.ICPLimits[lang]
		v.defaultEntry("icp-limits", lang, len(patterns), func(i int) string { return patterns[i].Pattern })
		for _, p := range patterns {
			v.pattern("icp-limits", lang, p.Pattern)
			switch {
			case p.Limit < 1:
				v.errorf(RuleLimits, "icp-limits.%s.%q: limit %d must be >= 1", lang, p.Pattern, p.Limit)
			case p.Limit < lo || p.Limit > hi:
				v.warnf(
					RuleLimits,
					"icp-limits.%s.%q: limit %d is outside the %s band %d-%d",
					lang,
					p.Pattern,
					p.Limit,
					cfg.ProjectType,
					lo,
					hi,
				)
			}
		}
	}
}

// defaultEntry checks that a pattern list exists and opens with ".*".
func (v *validator) defaultEntry(section string, lang Language, n int, pattern func(int) string) {
	if n == 0 {
		v.errorf(RuleDefaultEntry, "%s.%s: no patterns; a %q entry is required", section, lang, PatternAll)
		return
	}
	if first := pattern(0); first != PatternAll {
		v.errorf(RuleDefaultEntry, "%s.%s: first pattern is %q, must be %q", section, lang, first, PatternAll)
	}
}

func (v *validator) pattern(section string, lang Language, pattern string) {
	if _, err := regexp.Compile(pattern); err != nil {
		v.errorf(RulePatterns, "%s.%s.%q: invalid regex: %v", section, lang, pattern, err)
	}
}

func (v *validator) filters(cfg *Config) {
	v.filterList("include", cfg.Include)
	v.filterList("exclude", cfg.Exclude)
}

func (v *validator) filterList(section string, list []string) {
	for i, entry := range list {
		if strings.TrimSpace(entry) == "" {
			v.errorf(RulePatterns, "%s[%d]: empty pattern", section, i)
			continue
		}
		if re, ok := strings.CutPrefix(entry, RegexPrefix); ok {
			if _, err := regexp.Compile(re); err != nil {
				v.errorf(RulePatterns, "%s[%d]: invalid regex %q: %v", section, i, re, err)
			}
		}
	}
}

func (v *validator) enforcement(cfg *Config) {
	mode := cfg.Enforcement.LegacyMode
	if !IsLegacyMode(mode) {
		v.errorf(RuleLegacyMode, "enforcement.legacy_mode: %q is not one of %s", mode, join(LegacyModes()))
		return
	}
	if cfg.ProjectType == ProjectGreenfield && mode != ModeStrictAll {
		v.errorf(
			RuleLegacyMode,
			"enforcement.legacy_mode: %s project requires %s, got %s",
			ProjectGreenfield,
			ModeStrictAll,
			mode,
		)
	}
	if mode == ModeMeasureOnly && cfg.Enforcement.BlockOnCI {
		v.errorf(RuleLegacyMode, "enforcement.block_on_ci: must be false when legacy_mode is %s", ModeMeasureOnly)
	}
	if mode == ModeBoyScout {
		v.warnf(
			RuleLegacyMode,
			"enforcement.legacy_mode: %s needs the baseline store, which is not supported yet",
			ModeBoyScout,
		)
	}
}

func (v *validator) reporter(cfg *Config) {
	if !IsReporterFormat(cfg.Reporter.Format) {
		v.errorf(RuleReporter, "reporter.format: %q is not one of %s", cfg.Reporter.Format, join(ReporterFormats()))
	}
}

func (v *validator) languageSets(cfg *Config) {
	for _, lang := range sortedLanguages(cfg.Metrics, v.specs) {
		if _, ok := cfg.ICPLimits[lang]; !ok {
			v.errorf(RuleLanguageSets, "icp-limits: missing entry for %s, which metrics configures", lang)
		}
	}
	for _, lang := range sortedLanguages(cfg.ICPLimits, v.specs) {
		if _, ok := cfg.Metrics[lang]; !ok {
			v.errorf(RuleLanguageSets, "metrics: missing entry for %s, which icp-limits configures", lang)
		}
	}
}

func (v *validator) timeout(cfg *Config) {
	if cfg.Timeout < 0 {
		v.errorf(RuleTimeout, "timeout: %s must be >= 0", cfg.Timeout)
	}
}

// sortedLanguages returns the keys of m in specs order, with unknown
// languages last in lexical order, so messages are stable.
func sortedLanguages[V any](m map[Language]V, specs []LanguageSpec) []Language {
	var out []Language
	for _, spec := range specs {
		if _, ok := m[spec.ID]; ok {
			out = append(out, spec.ID)
		}
	}
	var unknown []string
	for lang := range m {
		if _, ok := FindSpec(specs, lang); !ok {
			unknown = append(unknown, string(lang))
		}
	}
	slices.Sort(unknown)
	for _, s := range unknown {
		out = append(out, Language(s))
	}
	return out
}

// sortedMetrics returns the keys of m in Metrics order, unknown ids last.
func sortedMetrics(m map[MetricID]float64) []MetricID {
	var out []MetricID
	for _, id := range Metrics() {
		if _, ok := m[id]; ok {
			out = append(out, id)
		}
	}
	var unknown []string
	for id := range m {
		if !IsMetric(id) {
			unknown = append(unknown, string(id))
		}
	}
	slices.Sort(unknown)
	for _, s := range unknown {
		out = append(out, MetricID(s))
	}
	return out
}

func join[T ~string](list []T) string {
	parts := make([]string, len(list))
	for i, s := range list {
		parts[i] = string(s)
	}
	return strings.Join(parts, ", ")
}
