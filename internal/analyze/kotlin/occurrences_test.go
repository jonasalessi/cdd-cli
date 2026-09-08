package kotlin

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fixtureNames returns every Kotlin fixture under testdata, named the way
// analyzeFixture wants them, so a fixture added later is covered by the
// invariants without anyone remembering to list it.
func fixtureNames(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir("testdata", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.EqualFold(filepath.Ext(p), extKotlin) {
			rel, relErr := filepath.Rel("testdata", p)
			if relErr != nil {
				return relErr
			}
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, out)
	return out
}
