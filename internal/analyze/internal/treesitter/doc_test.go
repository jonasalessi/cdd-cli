package treesitter

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPackageIsGrammarAndMetricAgnostic (TC-R5): nothing here imports the
// configuration or a grammar binding; the package only knows tree-sitter
// and the occurrence type it sorts.
func TestPackageIsGrammarAndMetricAgnostic(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			require.NotContains(t, path, "/internal/config", "%s imports the configuration", name)
			require.NotContains(t, path, "tree-sitter-", "%s imports a grammar binding", name)
		}
	}
}
