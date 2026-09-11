package kotlin

import (
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/jonasalessi/cdd-cli/internal/analyze/internal/treesitter"
	"github.com/jonasalessi/cdd-cli/internal/config"
)

// module is one qualified path a file imports, with the import statements
// that name the same path merged into a single entry.
type module struct {
	path string
	// bindings are the local names the imports introduce: the last segment
	// of the path, or the alias after `as`.
	bindings []string
	// internal says which coupling metric the module is charged to.
	internal bool
	// star marks `import a.b.*`, which binds no name the analyzer can see
	// and is therefore charged to every unit of the file, like a
	// side-effect import in TypeScript.
	star bool
	// at is the range of the first import naming the path, which is where
	// the module's coupling occurrence points. That statement sits outside
	// every unit it is charged to, as the contract on analyze.Occurrence
	// says.
	at treesitter.Span
}

// modules returns the modules imported by the file, in source order
// (FR-7, FR-8). Every import names one qualified path; two imports of the
// same path are one module with two bindings.
func modules(g *grammar, root *ts.Node, src []byte, prefixes []string) []module {
	var out []module
	index := map[string]int{}
	for _, child := range treesitter.NamedChildren(root) {
		n := child
		if g.kindOf(&n) != kindImport {
			continue
		}
		path := importPath(g, &n, src)
		if path == "" {
			continue
		}
		at, ok := index[path]
		if !ok {
			at = len(out)
			index[path] = at
			out = append(out, module{
				path:     path,
				internal: isInternal(path, prefixes),
				at:       treesitter.SpanOf(&n),
			})
		}
		if hasToken(&n, g.tokens.star) {
			out[at].star = true
			continue
		}
		if name := importBinding(g, &n, src); name != "" {
			out[at].bindings = append(out[at].bindings, name)
		}
	}
	return out
}

// importPath returns the qualified path an import names: the text of its
// qualified_identifier, which for a star import stops before the `.*`.
func importPath(g *grammar, n *ts.Node, src []byte) string {
	for _, child := range treesitter.NamedChildren(n) {
		q := child
		if g.kindOf(&q) == kindQualifiedIdentifier {
			return treesitter.Text(&q, src)
		}
	}
	return ""
}

// importBinding returns the local name an import introduces: the alias
// after `as`, which the grammar hangs off the import as a bare identifier,
// or else the last segment of the qualified path.
func importBinding(g *grammar, n *ts.Node, src []byte) string {
	var last *ts.Node
	for _, child := range treesitter.NamedChildren(n) {
		c := child
		switch g.kindOf(&c) {
		case kindIdentifier:
			return treesitter.Text(&c, src)
		case kindQualifiedIdentifier:
			segments := treesitter.NamedChildren(&c)
			if len(segments) > 0 {
				last = &segments[len(segments)-1]
			}
		}
	}
	return treesitter.Text(last, src)
}

// isInternal classifies a qualified path: it is internal when it equals one
// of the configured prefixes or is its dot-delimited subpath ("com.acme"
// matching "com.acme.shared.Money"); everything else, `java.*` and
// `kotlinx.*` included, is external. An empty prefix matches nothing.
// Kotlin has no relative import, and the analyzer never guesses from the
// file's own package: a same-package reference needs no import and is
// invisible to it.
func isInternal(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if prefix != "" && (path == prefix || strings.HasPrefix(path, prefix+".")) {
			return true
		}
	}
	return false
}

// countCoupling charges the unit for the modules it uses. CDD counts
// "direct references to domain classes" and "external library units" per
// unit (docs/cdd.md section 2), but imports are file-level, so a module is
// charged to a unit when the unit mentions one of the module's bindings,
// once per module however many bindings or mentions there are. A star
// import binds nothing the analyzer can see and is charged to every unit
// of the file.
//
// The charge points at the import that brings the module in, which is
// above the unit rather than inside it: it is the one place the dependency
// is written down.
//
// The reference test is by name only: a local declaration that shadows an
// imported name makes the unit look like a user of that module.
func (c *counter) countCoupling(mods []module) {
	for i := range mods {
		m := &mods[i]
		if !m.star && !c.uses(m.bindings) {
			continue
		}
		if m.internal {
			c.chargeSpan(config.MetricInternalCoupling, m.at)
			continue
		}
		c.chargeSpan(config.MetricExternalCoupling, m.at)
	}
}

// uses reports whether the unit mentions any of the given bindings.
func (c *counter) uses(bindings []string) bool {
	for _, b := range bindings {
		if _, ok := c.refs[b]; ok {
			return true
		}
	}
	return false
}
