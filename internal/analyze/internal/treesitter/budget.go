package treesitter

import (
	"context"
	"fmt"
	"time"

	ts "github.com/tree-sitter/go-tree-sitter"
)

// ParseBudget is how long a single file may take to parse when the caller
// set no deadline. It bounds the damage a minified or generated file can do
// without turning a slow machine into a failure.
const ParseBudget = 30 * time.Second

// Budget returns how long parsing may take: what is left of the caller's
// deadline, or ParseBudget when the caller set none.
func Budget(ctx context.Context) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return ParseBudget
	}
	switch left := time.Until(deadline); {
	case left <= 0:
		return time.Microsecond
	case left < ParseBudget:
		return left
	default:
		return ParseBudget
	}
}

// TimeoutMicros turns a parse budget into the unsigned microseconds the
// parser expects, never below one.
func TimeoutMicros(d time.Duration) uint64 {
	if micros := d.Microseconds(); micros > 1 {
		return uint64(micros)
	}
	return 1
}

// Parse runs parser over src, bounded by the caller's deadline when there is
// one and by ParseBudget when there is not. The parser must already be set
// to its grammar. A parse that does not finish in time leaves the parser
// reset, so the next file starts from the beginning instead of resuming this
// one, and reports the context's error when the context is what ran out.
func Parse(ctx context.Context, parser *ts.Parser, src []byte) (*ts.Tree, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// The parser is bounded by its own timeout rather than by the context:
	// the binding's context-aware entry point dereferences a cancellation
	// flag the caller never set. The timeout is derived from the deadline,
	// so a run that is running out of time does not wait for a whole file.
	parser.SetTimeoutMicros(TimeoutMicros(Budget(ctx)))
	tree := parser.Parse(src, nil)
	if tree == nil {
		parser.Reset()
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("parsing did not finish within %s", Budget(ctx))
	}
	return tree, nil
}
