package jvm

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jonasalessi/cdd-cli/internal/analyze/internal/treesitter"
	"github.com/jonasalessi/cdd-cli/internal/config"
)

// span returns a one-character range on the given line, enough to tell two
// import statements apart.
func span(line int) treesitter.Span {
	return treesitter.Span{Line: line, Col: 1, EndLine: line, EndCol: 2}
}

// TestIsInternal (TC-R1) is the classification table Kotlin used to hold, no
// parsing involved.
func TestIsInternal(t *testing.T) {
	cases := []struct {
		path     string
		prefixes []string
		want     bool
	}{
		{"com.acme.shared.Money", []string{"com.acme"}, true},
		{"com.acme.shared.Money", []string{"com.acme.shared"}, true},
		{"com.acme.shared.Money", []string{"com.acmecorp"}, false},
		{"com.acme.shared.Money", []string{"com.acme.shared.Money"}, true},
		{"com.acme.shared.Money", []string{""}, false},
		{"com.acme.shared.Money", nil, false},
		{"java.util.List", []string{"com.acme"}, false},
		{"com.acme", []string{"com.acme"}, true},
		{"com.acmecorp.X", []string{"com.acme"}, false},
	}
	for _, c := range cases {
		require.Equal(t, c.want, IsInternal(c.path, c.prefixes), "%s with %v", c.path, c.prefixes)
	}
}

// TestBindMergesTheSamePath (TC-R2): imports naming one path are one module,
// keeping every binding they introduce and the range of the first of them.
func TestBindMergesTheSamePath(t *testing.T) {
	imports := NewImports([]string{"a"})
	imports.Bind("a.B", "B", span(1))
	imports.Bind("a.B", "B", span(2))
	imports.Bind("a.B", "C", span(3))

	mods := imports.Modules()
	require.Len(t, mods, 1)
	require.Equal(t, "a.B", mods[0].Path)
	require.Equal(t, []string{"B", "B", "C"}, mods[0].Bindings)
	require.Equal(t, span(1), mods[0].At)
	require.True(t, mods[0].Internal)
	require.False(t, mods[0].Star)
}

// TestBindWithoutABinding (TC-R2): an import whose local name the grammar
// could not read still makes the module known, binding nothing.
func TestBindWithoutABinding(t *testing.T) {
	imports := NewImports(nil)
	imports.Bind("a.B", "", span(1))

	mods := imports.Modules()
	require.Len(t, mods, 1)
	require.Empty(t, mods[0].Bindings)
}

// TestStarMarksTheModule (TC-R2, TC-R4): a star import binds no visible name,
// and a named import of the same path leaves the star standing.
func TestStarMarksTheModule(t *testing.T) {
	imports := NewImports(nil)
	imports.Star("a.b", span(1))
	imports.Bind("a.b", "b", span(2))

	mods := imports.Modules()
	require.Len(t, mods, 1)
	require.True(t, mods[0].Star)
	require.Equal(t, span(1), mods[0].At)
}

// TestModulesAreInSourceOrder (TC-R2): the order the imports were reported
// in, whatever their paths sort as.
func TestModulesAreInSourceOrder(t *testing.T) {
	imports := NewImports([]string{"com.acme"})
	imports.Bind("zeta.Last", "Last", span(1))
	imports.Bind("com.acme.Money", "Money", span(2))
	imports.Star("alpha.util", span(3))
	imports.Bind("zeta.Last", "Alias", span(4))

	mods := imports.Modules()
	require.Equal(t, []string{"zeta.Last", "com.acme.Money", "alpha.util"}, paths(mods))
	require.Equal(t, []bool{false, true, false}, internals(mods))
}

// TestEmptyImportsHaveNoModules (TC-R2).
func TestEmptyImportsHaveNoModules(t *testing.T) {
	require.Empty(t, NewImports(nil).Modules())
}

// TestModuleUsedBy (TC-R4): the star is used by every unit, a named module
// only by the units mentioning one of its bindings.
func TestModuleUsedBy(t *testing.T) {
	mentioned := map[string]struct{}{"Money": {}}
	none := map[string]struct{}{}
	cases := []struct {
		name string
		mod  Module
		refs map[string]struct{}
		want bool
	}{
		{"star with no refs", Module{Star: true}, none, true},
		{"binding mentioned", Module{Bindings: []string{"Ledger", "Money"}}, mentioned, true},
		{"bindings absent", Module{Bindings: []string{"Ledger"}}, mentioned, false},
		{"no bindings at all", Module{}, mentioned, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := c.mod
			require.Equal(t, c.want, m.UsedBy(c.refs))
		})
	}
}

// TestModuleMetric (TC-R5): the module's classification picks the metric.
func TestModuleMetric(t *testing.T) {
	internal := Module{Internal: true}
	external := Module{}
	require.Equal(t, config.MetricInternalCoupling, internal.Metric())
	require.Equal(t, config.MetricExternalCoupling, external.Metric())
}

// paths returns the modules' paths, in order.
func paths(mods []Module) []string {
	out := make([]string, len(mods))
	for i := range mods {
		out[i] = mods[i].Path
	}
	return out
}

// internals returns the modules' classifications, in order.
func internals(mods []Module) []bool {
	out := make([]bool, len(mods))
	for i := range mods {
		out[i] = mods[i].Internal
	}
	return out
}
