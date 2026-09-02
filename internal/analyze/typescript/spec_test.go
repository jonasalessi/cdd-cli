package typescript

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecID(t *testing.T) {
	assert.Equal(t, "typescript", string(Spec().ID))
	assert.Len(t, Spec().NotApplicable, 0, "every metric applies to typescript")
}

func TestDetectPackagesReadsTSConfigWithComments(t *testing.T) {
	got, err := Spec().DetectPackages(filepath.Join("testdata", "aliases"))
	require.NoError(t, err)
	assert.Equal(t, []string{"@app/", "@lib/"}, got)
}

func TestDetectPackagesOfABrokenTSConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{not json"), 0o644))
	got, err := Spec().DetectPackages(dir)
	require.NoError(t, err, "a tsconfig that does not parse is not an error")
	assert.Nil(t, got)
}

func TestDetectPackagesWithoutTSConfig(t *testing.T) {
	got, err := Spec().DetectPackages(t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestStripJSONC(t *testing.T) {
	in := "{\n// line comment\n\"a\": \"http://x//y\", /* block */\n\"b\": [1, 2,],\n}"
	assert.JSONEq(t, `{"a": "http://x//y", "b": [1, 2]}`, string(stripJSONC([]byte(in))))
}

func TestStripJSONCKeepsEscapedQuotes(t *testing.T) {
	in := `{"a": "say \"//hi\"" /* after */}`
	assert.JSONEq(t, `{"a": "say \"//hi\""}`, string(stripJSONC([]byte(in))),
		"an escaped quote does not end the string, so the slashes stay")
}
