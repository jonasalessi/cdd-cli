package analyze

import (
	"fmt"
	"maps"
	"regexp"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

// Weights are the metric weights one language configures, pattern by
// pattern, in the document order of cdd.config.yaml. Every pattern whose
// regex matches a file applies; a later entry merges its weights over the
// earlier ones, metric by metric.
type Weights struct {
	entries []weightEntry
}

// weightEntry is one compiled "pattern: weights" pair.
type weightEntry struct {
	match   *regexp.Regexp
	weights map[config.MetricID]float64
}

// NewWeights compiles the pattern list config.Config.Metrics holds for one
// language. A pattern that is not a valid RE2 regex is an error naming it;
// config.Validate reports the same problem earlier and with more context.
func NewWeights(patterns config.PatternWeights) (Weights, error) {
	entries := make([]weightEntry, 0, len(patterns))
	for _, p := range patterns {
		match, err := regexp.Compile(p.Pattern)
		if err != nil {
			return Weights{}, fmt.Errorf("pattern %q: %w", p.Pattern, err)
		}
		entries = append(entries, weightEntry{match: match, weights: p.Weights})
	}
	return Weights{entries: entries}, nil
}

// For returns the weights that apply to path, which is slash-separated and
// relative to the configuration file. A metric absent from the result is
// disabled for that file. The map is fresh, so the caller may keep it.
func (w Weights) For(path string) map[config.MetricID]float64 {
	merged := map[config.MetricID]float64{}
	for _, e := range w.entries {
		if !e.match.MatchString(path) {
			continue
		}
		maps.Copy(merged, e.weights)
	}
	return merged
}

// Limits are the ICP limits one language configures, pattern by pattern, in
// document order. The last matching entry wins; limits do not merge.
type Limits struct {
	entries []limitEntry
}

// limitEntry is one compiled "pattern: limit" pair.
type limitEntry struct {
	match *regexp.Regexp
	limit int
}

// NewLimits compiles the pattern list config.Config.ICPLimits holds for one
// language. A pattern that is not a valid RE2 regex is an error naming it.
func NewLimits(patterns config.PatternLimits) (Limits, error) {
	entries := make([]limitEntry, 0, len(patterns))
	for _, p := range patterns {
		match, err := regexp.Compile(p.Pattern)
		if err != nil {
			return Limits{}, fmt.Errorf("pattern %q: %w", p.Pattern, err)
		}
		entries = append(entries, limitEntry{match: match, limit: p.Limit})
	}
	return Limits{entries: entries}, nil
}

// For returns the limit of the last pattern matching path, or 0 when none
// does. A validated configuration always opens with config.PatternAll, so 0
// only comes back from a hand-built list.
func (l Limits) For(path string) int {
	limit := 0
	for _, e := range l.entries {
		if e.match.MatchString(path) {
			limit = e.limit
		}
	}
	return limit
}

// Resolver answers, for one language, how a file is weighed and limited. It
// is built once per run and only read afterwards, so every worker shares it.
type Resolver struct {
	weights Weights
	limits  Limits
}

// NewResolver compiles the metrics and icp-limits pattern lists cfg holds
// for lang.
func NewResolver(cfg *config.Config, lang config.Language) (*Resolver, error) {
	weights, err := NewWeights(cfg.Metrics[lang])
	if err != nil {
		return nil, fmt.Errorf("metrics.%s: %w", lang, err)
	}
	limits, err := NewLimits(cfg.ICPLimits[lang])
	if err != nil {
		return nil, fmt.Errorf("icp-limits.%s: %w", lang, err)
	}
	return &Resolver{weights: weights, limits: limits}, nil
}

// Resolve scores the units of the file at path against the weights and the
// limit that path resolves to.
func (r *Resolver) Resolve(path string, units []Unit) []UnitReport {
	if len(units) == 0 {
		return nil
	}
	weights := r.weights.For(path)
	limit := r.limits.For(path)
	out := make([]UnitReport, len(units))
	for i, u := range units {
		out[i] = Score(u, weights, limit)
	}
	return out
}

// Score applies weights and limit to one unit: a metric absent from weights
// is dropped from the report, the remaining counts are multiplied by their
// weight, and the sum is the unit's ICP total. Metrics are summed in
// config.Metrics order, so the total does not depend on map iteration.
func Score(unit Unit, weights map[config.MetricID]float64, limit int) UnitReport {
	counts := make(map[config.MetricID]int, len(unit.Counts))
	scores := make(map[config.MetricID]float64, len(unit.Counts))
	total := 0.0
	for _, id := range config.Metrics() {
		count, counted := unit.Counts[id]
		weight, enabled := weights[id]
		if !counted || !enabled {
			continue
		}
		counts[id] = count
		scores[id] = float64(count) * weight
		total += scores[id]
	}
	return UnitReport{
		Name:    unit.Name,
		Kind:    unit.Kind,
		Line:    unit.Line,
		Col:     unit.Col,
		Counts:  counts,
		Scores:  scores,
		Total:   total,
		Limit:   limit,
		Exceeds: total > float64(limit),
	}
}
