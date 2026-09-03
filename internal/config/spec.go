package config

import (
	"context"
	"slices"
)

// LanguageSpec is everything the configuration side of cdd knows about one
// language: its id, how it is presented, which files belong to it, which
// metrics its analyzer cannot count, and the wording and defaults cdd init
// writes for it. Every language ships one under internal/analyze/<id>/, and
// consumers receive the registered specs as a slice; there is no global
// table.
//
// The struct is plain data plus one function, so config stays free of any
// analyzer dependency.
type LanguageSpec struct {
	// ID is the canonical language id used as key in cdd.config.yaml.
	ID Language
	// DisplayName is the label shown in prompts.
	DisplayName string
	// Extensions are the file extensions, dot included, the analyzer reads.
	Extensions []string
	// NotApplicable lists the metrics the analyzer cannot count.
	NotApplicable []MetricID
	// DefaultExcludes are the globs cdd init writes to "exclude": tests and
	// generated code.
	DefaultExcludes []string
	// Descriptions overrides the generic MetricDescription wording for the
	// metrics whose constructs have a language-specific name.
	Descriptions map[MetricID]string
	// PackageExample shows the shape of an internal package prefix.
	PackageExample string
	// LimitExamples are the commented layer overrides rendered under the
	// language's icp-limits entry when it comes first.
	LimitExamples []string
	// DetectPackages guesses the internal package prefixes of the project at
	// root. A missing manifest is not an error; the result is whatever was
	// found.
	DetectPackages func(ctx context.Context, root string) ([]string, error)
}

// Applicable returns the metrics the analyzer can count, in Metrics order.
func (s LanguageSpec) Applicable() []MetricID {
	var out []MetricID
	for _, m := range metrics {
		if !slices.Contains(s.NotApplicable, m) {
			out = append(out, m)
		}
	}
	return out
}

// IsApplicable reports whether m is a known metric the analyzer can count.
func (s LanguageSpec) IsApplicable(m MetricID) bool {
	return IsMetric(m) && !slices.Contains(s.NotApplicable, m)
}

// Description returns the inline comment written next to m: the language's
// own wording when it has one, else the generic MetricDescription.
func (s LanguageSpec) Description(m MetricID) string {
	if d, ok := s.Descriptions[m]; ok {
		return d
	}
	return MetricDescription(m)
}

// FindSpec returns the spec with id, if specs holds one.
func FindSpec(specs []LanguageSpec, id Language) (LanguageSpec, bool) {
	for _, s := range specs {
		if s.ID == id {
			return s, true
		}
	}
	return LanguageSpec{}, false
}

// LanguageIDs returns the ids of specs in the same order.
func LanguageIDs(specs []LanguageSpec) []Language {
	out := make([]Language, len(specs))
	for i, s := range specs {
		out[i] = s.ID
	}
	return out
}
