package detect

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

// maxJVMFiles caps how many java / kotlin sources Packages reads before it
// settles on the prefixes found so far.
const maxJVMFiles = 200

// maxJVMPrefixes caps how many package prefixes the java / kotlin strategy
// returns.
const maxJVMPrefixes = 5

// jvmPackageRe matches a java or kotlin package declaration; the semicolon
// is optional so one expression covers both.
var jvmPackageRe = regexp.MustCompile(`^\s*package\s+([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)`)

// Packages guesses the internal package prefixes of the project at root, one
// strategy per language: the module line of go.mod, java / kotlin package
// declarations reduced to their shortest telling prefixes, and tsconfig.json
// path aliases. A missing file is not an error; the result is whatever was
// found.
func Packages(root string, langs []config.Language) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	add := func(prefixes []string) {
		for _, p := range prefixes {
			if p != "" && !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	jvmDone := false
	for _, lang := range config.Languages() {
		if !slices.Contains(langs, lang) {
			continue
		}
		switch lang {
		case config.LangGo:
			add(goModule(root))
		case config.LangJava, config.LangKotlin:
			if !jvmDone {
				jvmDone = true
				add(jvmPrefixes(root, langs))
			}
		case config.LangTypeScript:
			add(tsAliases(root))
		}
	}
	return out, nil
}

// goModule reads the module line of <root>/go.mod.
func goModule(root string) []string {
	f, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if mod, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "module "); ok {
			return []string{strings.TrimSpace(mod)}
		}
	}
	return nil
}

// jvmPrefixes scans up to maxJVMFiles java / kotlin sources for package
// declarations and reduces them with commonPrefixes.
func jvmPrefixes(root string, langs []config.Language) []string {
	exts := map[string]bool{}
	for _, lang := range langs {
		if lang == config.LangJava || lang == config.LangKotlin {
			for _, ext := range extensions[lang] {
				exts[ext] = true
			}
		}
	}
	pkgs := map[string]bool{}
	read := 0
	walk := func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // an unreadable entry is skipped, not fatal
		}
		if entry.IsDir() {
			if skipDirs[entry.Name()] && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if read >= maxJVMFiles {
			return filepath.SkipAll
		}
		if !exts[strings.ToLower(filepath.Ext(entry.Name()))] {
			return nil
		}
		read++
		if pkg := declaredPackage(path); pkg != "" {
			pkgs[pkg] = true
		}
		return nil
	}
	if err := filepath.WalkDir(root, walk); err != nil {
		return nil
	}
	prefixes := commonPrefixes(slices.Sorted(maps.Keys(pkgs)))
	if len(prefixes) > maxJVMPrefixes {
		prefixes = prefixes[:maxJVMPrefixes]
	}
	return prefixes
}

// declaredPackage returns the package declared in the first 100 lines of the
// file at path, or "".
func declaredPackage(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for lines := 0; sc.Scan() && lines < 100; lines++ {
		if m := jvmPackageRe.FindStringSubmatch(sc.Text()); m != nil {
			return m[1]
		}
	}
	return ""
}

// commonPrefixes reduces package names to the shortest prefixes that still
// tell project areas apart: the longest shared segment prefix plus one more
// segment per branch. {com.acme.billing.api, com.acme.billing.db,
// com.acme.shared} becomes {com.acme.billing, com.acme.shared}.
func commonPrefixes(names []string) []string {
	if len(names) <= 1 {
		return names
	}
	segs := make([][]string, len(names))
	for i, name := range names {
		segs[i] = strings.Split(name, ".")
	}
	shared := segs[0]
	for _, s := range segs[1:] {
		shared = sharedSegments(shared, s)
	}
	if len(shared) == 0 {
		return groupByFirstSegment(names)
	}
	prefix := strings.Join(shared, ".")
	var out []string
	seen := map[string]bool{}
	for _, s := range segs {
		branch := prefix
		if len(s) > len(shared) {
			branch = prefix + "." + s[len(shared)]
		}
		if !seen[branch] {
			seen[branch] = true
			out = append(out, branch)
		}
	}
	slices.Sort(out)
	return out
}

// groupByFirstSegment splits names that share nothing into groups by their
// first segment and reduces each group on its own.
func groupByFirstSegment(names []string) []string {
	groups := map[string][]string{}
	for _, name := range names {
		first, _, _ := strings.Cut(name, ".")
		groups[first] = append(groups[first], name)
	}
	var out []string
	for _, first := range slices.Sorted(maps.Keys(groups)) {
		out = append(out, commonPrefixes(groups[first])...)
	}
	return out
}

// tsAliases returns the compilerOptions.paths keys of <root>/tsconfig.json
// with the trailing "*" stripped, so "@app/*" becomes "@app/". tsconfig
// files may carry comments and trailing commas; both are removed before
// parsing.
func tsAliases(root string) []string {
	data, err := os.ReadFile(filepath.Join(root, "tsconfig.json"))
	if err != nil {
		return nil
	}
	var doc struct {
		CompilerOptions struct {
			Paths map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}
	if err := json.Unmarshal(stripJSONC(data), &doc); err != nil {
		return nil
	}
	aliases := make([]string, 0, len(doc.CompilerOptions.Paths))
	for key := range doc.CompilerOptions.Paths {
		if alias := strings.TrimSuffix(key, "*"); alias != "" {
			aliases = append(aliases, alias)
		}
	}
	slices.Sort(aliases)
	return slices.Compact(aliases)
}

// trailingComma matches a comma that directly precedes a closing brace or
// bracket, which strict JSON forbids.
var trailingComma = regexp.MustCompile(`,(\s*[}\]])`)

// stripJSONC removes // and /* */ comments outside strings, then trailing
// commas, so a JSONC document passes encoding/json.
func stripJSONC(src []byte) []byte {
	out := make([]byte, 0, len(src))
	var inStr, inLine, inBlock bool
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case inLine:
			if c == '\n' {
				inLine = false
				out = append(out, c)
			}
		case inBlock:
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				inBlock = false
				i++
			}
		case inStr:
			out = append(out, c)
			if c == '\\' && i+1 < len(src) {
				i++
				out = append(out, src[i])
			} else if c == '"' {
				inStr = false
			}
		case c == '"':
			inStr = true
			out = append(out, c)
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			inLine = true
			i++
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			inBlock = true
			i++
		default:
			out = append(out, c)
		}
	}
	return trailingComma.ReplaceAll(out, []byte("$1"))
}

// sharedSegments returns the longest common prefix of two segment lists.
func sharedSegments(a, b []string) []string {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return a[:i]
		}
	}
	return a[:n]
}
