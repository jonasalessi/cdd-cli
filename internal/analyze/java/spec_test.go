package java

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecID(t *testing.T) {
	assert.Equal(t, "java", string(Spec().ID))
	assert.Len(t, Spec().NotApplicable, 0, "every metric applies to java")
}

func TestDetectPackagesSkipsKotlinFiles(t *testing.T) {
	got, err := Spec().DetectPackages(t.Context(), filepath.Join("testdata", "project"))
	require.NoError(t, err)
	assert.Equal(t, []string{"com.acme.billing", "com.acme.shared"}, got,
		"the kotlin source under com.acme.other is not read")
}

func TestDetectPackagesEmptyProject(t *testing.T) {
	got, err := Spec().DetectPackages(t.Context(), t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, got)
}
