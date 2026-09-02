package kotlin

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecID(t *testing.T) {
	assert.Equal(t, "kotlin", string(Spec().ID))
	assert.Len(t, Spec().NotApplicable, 0, "every metric applies to kotlin")
}

func TestDetectPackagesSkipsJavaFiles(t *testing.T) {
	got, err := Spec().DetectPackages(filepath.Join("testdata", "project"))
	require.NoError(t, err)
	assert.Equal(t, []string{"com.acme.billing", "com.acme.shared"}, got,
		"the java source under com.acme.other is not read; the build script has no package")
}

func TestDetectPackagesEmptyProject(t *testing.T) {
	got, err := Spec().DetectPackages(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, got)
}
