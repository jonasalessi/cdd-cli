package golang

import (
	"go/ast"
	"go/token"
)

// The Kind values a Go unit can carry. A type is reported by what a reader
// sees at its declaration, so a struct and an interface are told apart from
// every other named type; `methods` marks the methods of a type this file
// does not declare.
const (
	unitStruct    = "struct"
	unitInterface = "interface"
	unitType      = "type"
	unitFunc      = "func"
	unitMethods   = "methods"
)

// unitDecl is one extracted unit: the nodes whose subtrees the metrics are
// counted over — the TypeSpec, every method billed to it, or the function
// itself — plus the metadata the pipeline reports.
type unitDecl struct {
	name  string
	kind  string
	line  int
	col   int
	nodes []ast.Node
}

// units returns the file's units in declaration order (FR-3).
//
// A unit is a top-level type declaration with its methods billed to it, a
// top-level function without a receiver, or the methods of a receiver type
// this file does not declare, which form one `methods` unit positioned at
// the first of them. A method bills to its type wherever it sits, before or
// after the type's own declaration. Nothing nested is a unit, and top-level
// `var`, `const` and `import` declarations are not units either, so their
// contents are invisible.
//
// Line and Col are the position of the name for a type and of the `func`
// keyword for a function, so the specs of a `type ( … )` group are told
// apart by their own names instead of sharing the group's keyword.
func units(fset *token.FileSet, file *ast.File) []unitDecl {
	s := &unitSet{fset: fset, declared: declaredTypes(file), byType: map[string]int{}}
	for _, decl := range file.Decls {
		s.add(decl)
	}
	return s.billed()
}

// declaredTypes lists the type names this file declares, so a method seen
// before its own type still bills to the type's unit instead of opening a
// `methods` one.
func declaredTypes(file *ast.File) map[string]bool {
	out := map[string]bool{}
	for _, decl := range file.Decls {
		for _, spec := range typeSpecs(decl) {
			out[spec.Name.Name] = true
		}
	}
	return out
}

// typeSpecs returns the specs of a top-level `type` declaration, grouped or
// not, and nothing for every other declaration.
func typeSpecs(decl ast.Decl) []*ast.TypeSpec {
	gen, ok := decl.(*ast.GenDecl)
	if !ok || gen.Tok != token.TYPE {
		return nil
	}
	out := make([]*ast.TypeSpec, 0, len(gen.Specs))
	for _, spec := range gen.Specs {
		if ts, ok := spec.(*ast.TypeSpec); ok {
			out = append(out, ts)
		}
	}
	return out
}

// unitSet collects the units of one file in declaration order and remembers
// which methods each type owes its counts to.
type unitSet struct {
	fset     *token.FileSet
	declared map[string]bool
	byType   map[string]int
	methods  map[string][]ast.Node
	out      []unitDecl
}

// add turns one top-level declaration into units, or bills it to one.
func (s *unitSet) add(decl ast.Decl) {
	for _, spec := range typeSpecs(decl) {
		s.byType[spec.Name.Name] = len(s.out)
		s.out = append(s.out, typeUnit(s.fset, spec))
	}
	if fn, ok := decl.(*ast.FuncDecl); ok {
		s.addFunc(fn)
	}
}

// addFunc opens a unit for a function without a receiver and bills a method
// to the unit of the type it belongs to.
func (s *unitSet) addFunc(fn *ast.FuncDecl) {
	if fn.Recv == nil {
		s.out = append(s.out, funcUnit(s.fset, fn))
		return
	}
	s.addMethod(fn)
}

// addMethod bills a method to its receiver's unit, opening a `methods` unit
// on first sight of a receiver this file declares no type for. A receiver
// that names no type bills nowhere.
func (s *unitSet) addMethod(fn *ast.FuncDecl) {
	name := receiverName(fn)
	if name == "" {
		return
	}
	if _, known := s.byType[name]; !known && !s.declared[name] {
		s.byType[name] = len(s.out)
		s.out = append(s.out, methodsUnit(s.fset, fn, name))
	}
	if s.methods == nil {
		s.methods = map[string][]ast.Node{}
	}
	s.methods[name] = append(s.methods[name], fn)
}

// billed attaches every collected method to the unit of its receiver type
// and returns the units in declaration order.
func (s *unitSet) billed() []unitDecl {
	for name, nodes := range s.methods {
		at := s.byType[name]
		s.out[at].nodes = append(s.out[at].nodes, nodes...)
	}
	return s.out
}

// typeUnit reports a type declaration at its name.
func typeUnit(fset *token.FileSet, spec *ast.TypeSpec) unitDecl {
	pos := fset.Position(spec.Pos())
	return unitDecl{
		name:  spec.Name.Name,
		kind:  unitKind(spec.Type),
		line:  pos.Line,
		col:   pos.Column,
		nodes: []ast.Node{spec},
	}
}

// funcUnit reports a top-level function at its `func` keyword.
func funcUnit(fset *token.FileSet, fn *ast.FuncDecl) unitDecl {
	pos := fset.Position(fn.Pos())
	return unitDecl{
		name:  fn.Name.Name,
		kind:  unitFunc,
		line:  pos.Line,
		col:   pos.Column,
		nodes: []ast.Node{fn},
	}
}

// methodsUnit reports the methods of a type declared in another file at the
// `func` keyword of the first of them. Its nodes arrive with the billing.
func methodsUnit(fset *token.FileSet, fn *ast.FuncDecl, name string) unitDecl {
	pos := fset.Position(fn.Pos())
	return unitDecl{name: name, kind: unitMethods, line: pos.Line, col: pos.Column}
}

// unitKind reports what a reader sees at a type declaration. An alias and a
// named basic, func, map or channel type are all `type`.
func unitKind(expr ast.Expr) string {
	switch expr.(type) {
	case *ast.StructType:
		return unitStruct
	case *ast.InterfaceType:
		return unitInterface
	default:
		return unitType
	}
}

// receiverName returns the base type a method belongs to, unwrapping the
// pointer, the parentheses and the type arguments of a generic receiver.
// A receiver that holds no identifier — which no compilable file has —
// names no type and is reported as none.
func receiverName(fn *ast.FuncDecl) string {
	if len(fn.Recv.List) == 0 {
		return ""
	}
	expr := fn.Recv.List[0].Type
	for {
		switch t := expr.(type) {
		case *ast.StarExpr:
			expr = t.X
		case *ast.ParenExpr:
			expr = t.X
		case *ast.IndexExpr:
			expr = t.X
		case *ast.IndexListExpr:
			expr = t.X
		case *ast.Ident:
			return t.Name
		default:
			return ""
		}
	}
}
