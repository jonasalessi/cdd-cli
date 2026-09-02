package languages

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

// moduleRoot is the repository root relative to this package.
var moduleRoot = filepath.Join("..", "..")

// skippedDirs are never walked: build output, fixtures and dependencies.
var skippedDirs = map[string]bool{".git": true, "bin": true, "testdata": true, "vendor": true}

// forbiddenLiterals is every id of the vocabulary: spelled out anywhere but
// vocabulary.go and the language spec files, it is a second copy of the
// vocabulary that will drift.
func forbiddenLiterals() map[string]bool {
	out := map[string]bool{}
	for _, l := range All() {
		out[string(l.Spec.ID)] = true
	}
	for _, m := range config.Metrics() {
		out[string(m)] = true
	}
	for _, s := range config.ProjectTypes() {
		out[s] = true
	}
	for _, s := range config.LegacyModes() {
		out[s] = true
	}
	for _, s := range config.ReporterFormats() {
		out[s] = true
	}
	return out
}

// literalExempt reports whether rel may spell out vocabulary ids: the
// vocabulary itself and each language's spec.
func literalExempt(rel string) bool {
	if rel == "internal/config/vocabulary.go" {
		return true
	}
	ok, _ := path.Match("internal/analyze/*/spec.go", rel)
	return ok
}

// languageDir reports whether rel lives where language knowledge belongs:
// the registry or a language directory.
func languageDir(rel string) bool {
	if strings.HasPrefix(rel, "internal/languages/") {
		return true
	}
	dir := path.Dir(rel)
	return strings.HasPrefix(dir, "internal/analyze/") && dir != "internal/analyze"
}

// TestLiterals is FR-9: vocabulary ids appear as string literals only where
// they are defined, and no language table regrows outside the language
// directories. make check-literals runs it.
func TestLiterals(t *testing.T) {
	forbidden := forbiddenLiterals()
	fset := token.NewFileSet()
	var violations []string
	err := filepath.WalkDir(moduleRoot, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skippedDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(moduleRoot, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		file, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		violations = append(violations, inspect(fset, file, rel, forbidden)...)
		return nil
	})
	require.NoError(t, err)
	if len(violations) > 0 {
		t.Errorf("vocabulary literals or language tables outside their home:\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// inspect collects the violations of one file.
func inspect(fset *token.FileSet, file *ast.File, rel string, forbidden map[string]bool) []string {
	var out []string
	report := func(pos token.Pos, msg string) {
		out = append(out, rel+":"+strconv.Itoa(fset.Position(pos).Line)+": "+msg)
	}
	tags := map[*ast.BasicLit]bool{}
	inConfig := file.Name.Name == "config"
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Field:
			if n.Tag != nil {
				tags[n.Tag] = true
			}
		case *ast.BasicLit:
			if n.Kind != token.STRING || tags[n] || literalExempt(rel) {
				return true
			}
			if v, err := strconv.Unquote(n.Value); err == nil && forbidden[v] {
				report(n.Pos(), "vocabulary literal "+n.Value+" belongs in vocabulary.go or a spec.go")
			}
		case *ast.CaseClause:
			for _, e := range n.List {
				if isLanguageConstant(e) && !languageDir(rel) {
					report(e.Pos(), "switch on a language id; put the behavior in the language's spec")
				}
			}
		case *ast.CompositeLit:
			if isLanguageTable(n, inConfig) && !languageDir(rel) {
				report(n.Pos(), "language-keyed table; put the data in each language's spec")
			}
		}
		return true
	})
	return out
}

// isLanguageConstant matches config.Lang… selectors.
func isLanguageConstant(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "config" && strings.HasPrefix(sel.Sel.Name, "Lang")
}

// isLanguageTable matches a non-empty map[config.Language]… literal, or
// map[Language]… inside package config.
func isLanguageTable(lit *ast.CompositeLit, inConfig bool) bool {
	m, ok := lit.Type.(*ast.MapType)
	if !ok || len(lit.Elts) == 0 {
		return false
	}
	switch key := m.Key.(type) {
	case *ast.SelectorExpr:
		pkg, ok := key.X.(*ast.Ident)
		return ok && pkg.Name == "config" && key.Sel.Name == "Language"
	case *ast.Ident:
		return inConfig && key.Name == "Language"
	}
	return false
}

// TestLiteralsHelpers pins the exemptions so a rename of the spec file or
// the registry directory is noticed.
func TestLiteralsHelpers(t *testing.T) {
	require.True(t, literalExempt("internal/config/vocabulary.go"))
	require.True(t, literalExempt("internal/analyze/kotlin/spec.go"))
	require.False(t, literalExempt("internal/analyze/kotlin/analyzer.go"))
	require.False(t, literalExempt("internal/detect/languages.go"))

	require.True(t, languageDir("internal/languages/languages.go"))
	require.True(t, languageDir("internal/analyze/kotlin/analyzer.go"))
	require.True(t, languageDir("internal/analyze/internal/jvm/jvm.go"))
	require.False(t, languageDir("internal/analyze/analyze.go"))
	require.False(t, languageDir("internal/prompt/init_form.go"))

	_, err := os.Stat(filepath.Join(moduleRoot, "go.mod"))
	require.NoError(t, err, "moduleRoot must be the repository root")
}
