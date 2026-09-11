package golang

import "strings"

// isStdlib reports whether an import path names a package of the Go
// standard library. The rule is the one the go command itself applies: a
// path whose first slash-delimited element holds no dot is resolved against
// GOROOT, because every module path outside it starts with a domain name.
//
// So "fmt", "net/http", "go/ast", "embed", "unsafe" and the cgo
// pseudo-package "C" are standard library, while "golang.org/x/sync",
// "github.com/google/uuid", "gopkg.in/yaml.v3" and "example.com/app" are
// not — golang.org/x is versioned and released apart from the toolchain, so
// it is an external dependency like any other. Nothing is hard-coded: a
// package added in the next release classifies correctly the first time it
// is imported.
func isStdlib(importPath string) bool {
	root, _, _ := strings.Cut(importPath, "/")
	return !strings.Contains(root, ".")
}
