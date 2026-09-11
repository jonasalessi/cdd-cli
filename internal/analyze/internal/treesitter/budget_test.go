package treesitter

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ts "github.com/tree-sitter/go-tree-sitter"
	tsbind "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// TestBudget derives the parse timeout from the caller's deadline (TC-R4).
func TestBudget(t *testing.T) {
	require.Equal(t, ParseBudget, Budget(context.Background()))

	short, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	left := Budget(short)
	require.Positive(t, left)
	require.Less(t, left, ParseBudget)

	past, cancelPast := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelPast()
	require.Equal(t, time.Microsecond, Budget(past))

	long, cancelLong := context.WithTimeout(context.Background(), time.Hour)
	defer cancelLong()
	require.Equal(t, ParseBudget, Budget(long))
}

// TestTimeoutMicros never hands the parser a zero or negative timeout,
// which would mean "no limit at all".
func TestTimeoutMicros(t *testing.T) {
	require.Equal(t, uint64(1), TimeoutMicros(0))
	require.Equal(t, uint64(1), TimeoutMicros(-time.Second))
	require.Equal(t, uint64(ParseBudget.Microseconds()), TimeoutMicros(ParseBudget))
}

func newParser(t *testing.T) *ts.Parser {
	t.Helper()
	p := ts.NewParser()
	t.Cleanup(p.Close)
	require.NoError(t, p.SetLanguage(ts.NewLanguage(tsbind.LanguageTypescript())))
	return p
}

func TestParse(t *testing.T) {
	tree, err := Parse(context.Background(), newParser(t), []byte("class A {}\n"))
	require.NoError(t, err)
	defer tree.Close()
	require.False(t, tree.RootNode().HasError())
}

func TestParseCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tree, err := Parse(ctx, newParser(t), []byte("class A {}\n"))
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, tree)
}

// TestParseOutOfTime hands the parser a deadline it cannot meet on a large
// file and checks it gives up with an error instead of a tree, and that the
// parser is usable afterwards.
func TestParseOutOfTime(t *testing.T) {
	p := newParser(t)
	src := []byte(strings.Repeat("function f(a: number) { return a > 1 && a < 3 ? a : 0; }\n", 20000))
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Nanosecond))
	defer cancel()
	tree, err := Parse(ctx, p, src)
	require.Error(t, err)
	require.Nil(t, tree)

	tree, err = Parse(context.Background(), p, []byte("class A {}\n"))
	require.NoError(t, err)
	tree.Close()
}
