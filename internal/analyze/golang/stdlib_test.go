package golang

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsStdlib (TC-C7) pins the rule the go command itself applies: the
// first element of the path holds no dot. The cgo pseudo-package C is
// deliberately standard library — it names no module a reader can go and
// read, so it belongs with the toolchain rather than with the dependencies.
func TestIsStdlib(t *testing.T) {
	cases := map[string]bool{
		"fmt":                        true,
		"net/http":                   true,
		"go/ast":                     true,
		"embed":                      true,
		"unsafe":                     true,
		"C":                          true,
		"example.com/app":            false,
		"github.com/x/y":             false,
		"golang.org/x/sync":          false,
		"golang.org/x/sync/errgroup": false,
		"gopkg.in/yaml.v3":           false,
	}
	for importPath, want := range cases {
		t.Run(importPath, func(t *testing.T) {
			require.Equal(t, want, isStdlib(importPath))
		})
	}
}
