package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

func TestPackagesGoModule(t *testing.T) {
	got, err := Packages(fixture("go-only"), []config.Language{config.LangGo})
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/goonly"}, got)
}

func TestPackagesJVMPrefixes(t *testing.T) {
	langs := []config.Language{config.LangJava, config.LangKotlin}
	got, err := Packages(fixture("java-kotlin"), langs)
	require.NoError(t, err)
	assert.Equal(t, []string{"com.acme.billing", "com.acme.shared"}, got)
}

func TestPackagesJavaOnlySkipsKotlinFiles(t *testing.T) {
	got, err := Packages(fixture("java-kotlin"), []config.Language{config.LangJava})
	require.NoError(t, err)
	assert.Equal(t, []string{"com.acme.billing", "com.acme.shared"}, got)
}

func TestPackagesTSConfigWithComments(t *testing.T) {
	got, err := Packages(fixture("ts"), []config.Language{config.LangTypeScript})
	require.NoError(t, err)
	assert.Equal(t, []string{"@app/", "@lib/"}, got)
}

func TestPackagesMissingFiles(t *testing.T) {
	got, err := Packages(fixture("empty"), []config.Language{
		config.LangGo, config.LangJava, config.LangKotlin, config.LangTypeScript,
	})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestPackagesMixed(t *testing.T) {
	got, err := Packages(fixture("mixed"), []config.Language{config.LangGo, config.LangTypeScript})
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/mixed"}, got, "mixed has no tsconfig, only the module line")
}

// writeJVM lays out a java source declaring pkg at the given relative path.
func writeJVM(t *testing.T, root, rel, pkg string) {
	t.Helper()
	path := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("package "+pkg+";\n\nclass C {}\n"), 0o644))
}

func TestGoModuleWithoutAModuleLine(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("go 1.25.0\n"), 0o644))
	assert.Nil(t, goModule(dir))
}

func TestDeclaredPackageOfAnUnreadableFile(t *testing.T) {
	assert.Empty(t, declaredPackage(filepath.Join(t.TempDir(), "gone.java")))
}

func TestJVMPrefixesMissingRoot(t *testing.T) {
	assert.Nil(t, jvmPrefixes(filepath.Join(t.TempDir(), "gone"), []config.Language{config.LangJava}))
}

func TestJVMPrefixesSkipsBuildOutput(t *testing.T) {
	dir := t.TempDir()
	writeJVM(t, dir, "src/main/java/com/acme/app/C.java", "com.acme.app")
	writeJVM(t, dir, "build/generated/com/vendor/gen/G.java", "com.vendor.gen")

	assert.Equal(t, []string{"com.acme.app"}, jvmPrefixes(dir, []config.Language{config.LangJava}),
		"a build directory at the root is not scanned")
}

func TestJVMPrefixesStopsAtTheFileCap(t *testing.T) {
	dir := t.TempDir()
	// One package past the cap, so the walk must stop before reaching it.
	for i := range maxJVMFiles + 1 {
		writeJVM(t, dir, fmt.Sprintf("src/p%03d/C.java", i), fmt.Sprintf("com.acme.p%03d", i))
	}
	got := jvmPrefixes(dir, []config.Language{config.LangJava})
	assert.NotEmpty(t, got)
	assert.LessOrEqual(t, len(got), maxJVMPrefixes, "the prefix list is capped too")
}

func TestJVMPrefixesCapsTheePrefixList(t *testing.T) {
	dir := t.TempDir()
	for i := range maxJVMPrefixes + 3 {
		writeJVM(t, dir, fmt.Sprintf("src/r%d/C.java", i), fmt.Sprintf("root%d.pkg", i))
	}
	assert.Len(t, jvmPrefixes(dir, []config.Language{config.LangJava}), maxJVMPrefixes)
}

func TestTSAliasesOfABrokenTSConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{not json"), 0o644))
	assert.Nil(t, tsAliases(dir))
}

func TestCommonPrefixes(t *testing.T) {
	tests := map[string]struct {
		in   []string
		want []string
	}{
		"empty":  {in: nil, want: nil},
		"single": {in: []string{"com.acme.billing.api"}, want: []string{"com.acme.billing.api"}},
		"branches under a shared prefix": {
			in:   []string{"com.acme.billing.api", "com.acme.billing.db", "com.acme.shared"},
			want: []string{"com.acme.billing", "com.acme.shared"},
		},
		"one name equals the shared prefix": {
			in:   []string{"com.acme", "com.acme.billing.api"},
			want: []string{"com.acme", "com.acme.billing"},
		},
		"nothing shared groups by first segment": {
			in:   []string{"com.acme.a", "com.acme.b", "org.other.x", "org.other.y.z"},
			want: []string{"com.acme.a", "com.acme.b", "org.other.x", "org.other.y"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, commonPrefixes(tt.in))
		})
	}
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
