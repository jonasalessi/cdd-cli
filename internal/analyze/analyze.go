// Package analyze defines the contract every language analyzer implements.
// It holds no pipeline and no parser: an analyzer turns one source file into
// raw metric counts per code unit, and later features weigh, limit and
// report them.
package analyze

import (
	"context"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

// Analyzer counts the ICP constructs of one source file. Implementations
// never read the configuration: weights, disabled metrics, limits and
// enforcement are applied downstream.
type Analyzer interface {
	// Analyze parses src, the content of the file at path, and returns the
	// raw counts of every unit it contains.
	Analyze(ctx context.Context, path string, src []byte) (FileResult, error)
}

// FileResult is the outcome of analyzing one file.
type FileResult struct {
	// Units are the measured code units in source order.
	Units []Unit
	// Warnings are non-fatal findings, e.g. a construct the analyzer could
	// not classify.
	Warnings []string
}

// Unit is one measured code unit: a type, a function or a file, depending
// on the language.
type Unit struct {
	// Name identifies the unit for the reader, e.g. a type or function name.
	Name string
	// Kind says what the unit is, e.g. "class", "function" or "file".
	Kind string
	// Line and Col locate the unit's declaration, 1-based.
	Line, Col int
	// Counts are raw occurrences per metric, before any weight is applied.
	Counts map[config.MetricID]int
}
