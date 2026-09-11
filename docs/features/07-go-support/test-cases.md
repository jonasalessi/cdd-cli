# Feature 07 — Go Support: Test Cases

Companion to [task.md](task.md). Every case here is a constraint the
implementation must satisfy; a task is not done until the cases listed under it
are checked-in tests that pass. Case ids are stable — reference them in test
names or comments (`// TC-U3`) so a reviewer can map the suite back to this
file.

Levels follow AGENTS.md: **unit** tests exercise one package through its
exported or package-level contract with fixtures on disk; **integration** tests
exercise the CLI boundary (`cmd/check_go_test.go`: arguments, exit codes,
stdout/stderr, filesystem). Nothing is mocked — `go/parser` is cheap,
deterministic and already a dependency of the toolchain.

## Conventions

`internal/analyze/golang/helper_test.go` mirrors
`internal/analyze/java/helper_test.go`, **minus the `io.Closer` assertion and
the `t.Cleanup` close**: the Go analyzer holds no native resource, builds a
fresh `token.FileSet` per call and has nothing to close.

- `newTestAnalyzer(t, prefixes...)` — builds through
  `NewAnalyzer(analyze.Options{InternalPrefixes: prefixes})` and returns it.
- `readFixture(t, name)` — the bytes of `testdata/<name>`.
- `analyzeFixture(t, name, prefixes...)` — reads `testdata/<name>`, analyzes it
  under that same name, `require.NoError`.
- `analyzeSource(t, src, prefixes...)` — analyzes an inline source as
  `inline.go` and requires it to parse with no warning.
- `inMethod(src)` — wraps a snippet in a unit so a table test needs no
  boilerplate per case:

  ```go
  package p

  type Wrapper struct{}

  func (Wrapper) M(a, b, c bool, xs []int) {
  	// src
  }
  ```

  Three booleans and an `[]int` cover every clause, branch and loop case
  without redeclaring anything; the wrapper's own counts are all zero, so a
  case asserts the snippet alone.
- `unitNamed(t, res, name)`, `unitNames(res)` and
  `requireCount(t, unit, metric, want)` — the last also asserts
  `len(Counts) == len(config.Metrics())`.
- `units(fset, file)` is package-level, so `occurrences_test.go` can read a
  unit's declaration nodes back (TC-X4).

Fixtures live in `internal/analyze/golang/testdata/` and are the files spelled
out in task.md's "Worked fixtures" section, verbatim where a total is quoted
there. A fixture line that carries an expected value in a comment is the source
of truth for the assertion next to it. The two unparsable fixtures are
`broken.go.txt` and `empty.go.txt`; a test reads them and analyzes them under
`broken.go` and `empty.go`.

Metric ids in tests come from the `config.Metric…` constants, never string
literals (`make check-literals`), and the language id comes from the registry,
never from `"go"`.

---

## T1 — Spec, template and goldens (FR-1)

One test pins the whole spec rather than one field at a time: a golden struct
fails with a diff naming exactly the field that drifted, and the registry's
`TestSpecCompleteness` already validates shape.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-S1 | unit | `TestSpec`: `got := Spec()`; `require.NotNil(got.DetectPackages)`; set `got.DetectPackages = nil`; `assert.Equal(t, want, got)` with `want` the full `config.LanguageSpec` literal — `ID` `"go"`, `DisplayName` `"Go"`, `Extensions` `[]string{extGo}`, `NotApplicable` `[]config.MetricID{config.MetricExceptionHandling}`, `DefaultExcludes` `{"**/*_test.go", "vendor/**"}`, `Descriptions` keyed by `config.Metric…` with `code_branch`, `lambda`, the new `inheritance` and the new `stdlib_coupling` rows, `PackageExample` `"github.com/acme/api"`, `LimitExamples` unchanged. | Equal. Go funcs are not comparable, which is why `DetectPackages` is checked for presence and then cleared on the copy. |
| TC-S2 | unit | Mutation safety: `Spec().Extensions[0] = ".rs"`, `Spec().NotApplicable[0] = config.MetricLambda`, `Spec().DefaultExcludes[0] = "**/nothing/**"`, `Spec().Descriptions[config.MetricLambda] = "changed"`, then call `Spec()` again. | The second call returns the original values — `Spec()` clones its slices and copies its map. |
| TC-S3 | unit | `MetricInheritance` is no longer in `NotApplicable`; `MetricExceptionHandling` still is. | Exactly one entry. A Go project may now select `inheritance`, and may still not select `exception_handling`. |
| TC-S4 | unit | `extGo` is the only spelling of the extension in the package, and `"go"` the only spelling of the id — both in `spec.go`. | `make check-literals` green; `analyzer.go` compares against `extGo`. |
| TC-S5 | unit | Template: `internal/config/templates/cdd.config.yaml.tmpl:36` reads `inheritance … all`, `:32` still reads `exception_handling … not go`, and the "unit measured" sentence at `:64-65` describes Go's unit (a top-level type with its methods, or a top-level function) instead of "the source file for go". | The four goldens, `docs/features/01-init/config-template.yaml` and the dogfood `cdd.config.yaml` carry the same three lines after `-update`. |
| TC-S6 | unit | `internal/config` golden tests, `cmd` golden tests, `TestInitDogfoodConfigReproducible`, `internal/languages` `TestSpecCompleteness`, `TestEveryLanguageHasADirectory`, `TestEveryDirectoryIsRegistered`. | Green after the regeneration; `git diff --exit-code` after the dogfood `cdd init` is clean. |
| TC-S7 | unit | `DetectPackages` on `testdata/module/` and on a directory with no `go.mod`. | Unchanged by this feature: the module path, and empty. |

## T2 — Parsing and registration (FR-2, FR-9)

Fixtures: `broken.go.txt`, `empty.go.txt`, `header_only.go`, `generated.go`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-P1 | unit | Analyze `broken.go.txt` under the name `broken.go`. Its content is exactly `package app`, a blank line and `func f( {`. | `Units` empty; `Warnings` exactly `["syntax error at 3:9"]` — the position `go/parser` reports for `expected ')', found '{'`, pinned literally so a toolchain change is visible; `err == nil`. |
| TC-P2 | unit | A parse failure whose error is not a `scanner.ErrorList`, or whose first `scanner.Error` carries no position — asserted through the `syntaxWarning` helper directly if no source produces it. | Warning is `syntax error at 1:1`. Say in a test comment that the fallback is unreachable from real Go source, if that is what the exploration shows. |
| TC-P3 | unit | Analyze `empty.go.txt` (zero bytes) under `empty.go`. | No units, **no warnings**, no error — the analyzer short-circuits before `go/parser` says `expected 'package', found 'EOF'`. |
| TC-P4 | unit | Analyze a source of only spaces, tabs, newlines and a CRLF pair. | Same as TC-P3: no units, no warnings, no error. |
| TC-P5 | unit | Analyze `generated.go`, whose first comment before the package clause is `// Code generated by protoc-gen-go. DO NOT EDIT.`. | No units and no warnings, even though the file parses and holds a struct field and an `if`. A second case with the marker **after** the package clause yields the units normally. |
| TC-P6 | unit | Analyze `header_only.go` (package clause, `import "fmt"`, `var _ = fmt.Sprint()`). | No units, no warnings, no error; `fmt` is charged to nothing, because there is no unit to charge. |
| TC-P7 | unit | Analyze a path whose extension is not `.go`: `a.kt`, `a.gox`, `a`, `a.go.txt`. | Returns an error whose message names both the path and the extension; `Units` empty. |
| TC-P8 | unit | Analyze `A.GO` and `A.Go` holding valid Go. | Accepted — the extension test is case-insensitive (`strings.EqualFold` on `filepath.Ext`). |
| TC-P9 | unit | `TestCanceledContext`: an already-canceled `context.Context`. | `Analyze` returns an error satisfying `errors.Is(err, context.Canceled)` and no units; the check happens before parsing. |
| TC-P10 | unit | `internal/languages`: `TestEveryAnalyzerIsWired` plus a case asserting the Go entry's `NewAnalyzer` is non-nil and returns a non-nil `analyze.Analyzer`. | Green. `analyzer.go` and the `languages.go:22` line land in the same commit, or `TestEveryAnalyzerIsWired` fails. |
| TC-P11 | integration | The four pinned `cmd` assertions. `cmd/init_test.go:116-117`, `:140` and `:343` become `assert.NotContains(t, stderr, "no analyzer")`; `TestCheckSelectedUnavailableLanguage` (`cmd/check_test.go:466`) is deleted, its behaviour already covered by `internal/analyze/run_test.go` with fake languages. `cmd/check_test.go:368` is untouched and keeps passing. | `cdd init` on a Go fixture prints no analyzer warning at all, and `cdd check` on it no longer errors. `internal/analyze/run_test.go` still proves the "no analyzer for %s yet" path for an unwired language. |
| TC-P12 | unit | No resources: the analyzer does **not** implement `io.Closer`; analyzing the same fixture 200 times with one analyzer, and two analyzers built from `NewAnalyzer`, share no mutable state. | `_, ok := a.(io.Closer); assert.False(ok)`. Results identical across the 200 calls; each call builds its own `token.FileSet`, so nothing accumulates. |

## T3 — Units (FR-3)

Fixture: `units.go`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-U1 | unit | Names in order. | Exactly `A B C D E F G H Free init` — ten units. |
| TC-U2 | unit | Kinds. | `A`→`struct`, `B`→`interface`, `C`→`type`, `D`→`type` (an alias), `E`→`type` (a func type), `F`→`struct`, `G`→`type`, `H`→`methods`, `Free`→`func`, `init`→`func`. |
| TC-U3 | unit | Positions. | A `TypeSpec` unit sits on its **name**: `A` on `A`, and `F`/`G` on their own names inside the `type ( … )` group, not on the shared `type` keyword — so `F` and `G` have different `Line`. A `FuncDecl` unit sits on the `func` keyword: `Free` and `init` at column 1. |
| TC-U4 | unit | `func (g G) Early()` is written **above** the `type ( F; G )` group. | It is not a unit; its counts bill to `G`, whose own `Line` is still the group's `G` line. Add an `if` to `Early` and assert `G`'s `code_branch` sees it. |
| TC-U5 | unit | `func (H) M()` and `func (h H) N()` with no `TypeSpec` for `H` in the file. | One unit named `H`, kind `methods`, positioned at the `func` of `M` (the first method seen), carrying the counts of both methods. A second file that *does* declare `type H struct{}` produces a `struct` unit instead, never both. |
| TC-U6 | unit | Nothing nested is a unit: a type declared inside a function body, a struct type inside another struct, a `FuncLit` assigned to a local, a method of a nested type. | None appears in `Units`; each bills to the enclosing unit. |
| TC-U7 | unit | `var v sync.Mutex` and `const k = 1` at top level, plus a top-level `var f = func() { if x { } }`. | Not units. They add no `local_variable` and no `code_branch` to any unit, the func literal adds no `lambda`, and `sync` is charged to no unit. Assert the total across all units, not just one. |
| TC-U8 | unit | Receiver unwrapping: `func (s Stack[T]) Push()`, `func (p *Pair[K, V]) Key()`, `func (s *Stack[T]) Pop()` and a parenthesized `func (s (*Stack)) Peek()`. | Each bills to the unit named by the bare identifier — `Stack`, `Pair`, `Stack`, `Stack`. A receiver that unwraps to no `Ident` bills nowhere and warns nothing. |
| TC-U9 | unit | `TestUnitsAreInSourceOrder` over every fixture: `(Line, Col)` is strictly increasing across `Units`. | Passes, `units.go` included, where `Early` precedes `G`'s declaration. |

## T4 — Branches and conditions (FR-5, FR-6)

Fixtures: `cdd_examples.go`, `branches.go`, `conditions.go`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-B1 | unit | `Check` in `cdd_examples.go`. | `code_branch` 1, `condition` 2 — the doc's `if a > b && c < d` = 3. |
| TC-B2 | unit | `IfElse` in `cdd_examples.go`. | `code_branch` 2 — the doc's "if-else = 2": the `IfStmt` and its `BlockStmt` alternative. |
| TC-B3 | unit | `Chain` in `branches.go` (`if / else if / else if / else`). | `code_branch` 4, not 7. Three occurrences on `IfStmt` nodes and one on the final `else` **block**, whose range starts at the `{` — `go/ast` keeps no `else` position. |
| TC-B4 | unit | `if a { } else if b { }` with no trailing `else`, and a bare `if a { }`. | 2 and 1. |
| TC-B5 | unit | `Switch` in `branches.go`, plus inline `switch { default: }` and `switch x { case 1, 2, 3: }`. | `Switch` is `code_branch` 2 — one occurrence on `case 1:`, one on `case 2, 3:` (one arm, one point), none on `default:`. Inline: 0 and 1. |
| TC-B6 | unit | `TypeSwitch` in `branches.go`. | `code_branch` 2 (`case int`, `case string, []byte`; `default` 0), `local_variable` 0 for the `t :=` guard. A `switch v.(type)` with no guard counts the same. |
| TC-B7 | unit | `Select` in `branches.go`. | `code_branch` 2 — a `CommClause` with a `Comm` statement counts whether it is a send, a receive or a receive-with-`:=`; `default:` is 0. `local_variable` 1, on `x`. |
| TC-B8 | unit | `Loops` in `branches.go`: `for i := 0; …`, `for _, x := range xs`, `for flag`, `for {}`, plus an inline `for range xs` and `for k, v := range m`. | One `code_branch` per loop in every form. `local_variable` on `i`, `x`, `k` and `v`; the blank `_` and a value-less `for range xs` add none. |
| TC-B9 | unit | Clause table through `inMethod`: `a && b` → 2; `a && b \|\| c` → 3; `!(a \|\| b) && x` → 3; `a > 1` → 0; `xs[0]&1 \| 3` → 0; `a && (b \|\| !c)` → 3; `!a` → 0. | As stated. The corresponding methods of `conditions.go` (`Both`, `Either`, `Negated`, `Compare`, `Bits`) carry the first five. |
| TC-B10 | unit | A nested chain in an argument list: `use(a && b, c \|\| a)` and `if (a && b) && (c \|\| a) { }`. | `condition` 4 both times. No clause counted twice — the consumed set holds. |
| TC-B11 | unit | Statements that are not branches: `return`, `break`, `continue`, `goto L`, a `LabeledStmt`, `fallthrough`, `go f()`, `defer f()`, `v, ok := x.(int)` (a `TypeAssertExpr` outside a switch), `panic("x")` and `recover()`. | `code_branch` 0 for a method containing only those; `exception_handling` 0 for all of them, and the key is still present in `Counts`. |
| TC-B12 | unit | Whole-unit totals. | `Examples` 3 / 2, `Branches` `code_branch` 14 + `condition` 0 + `local_variable` 3, `noop` all zero, `Conditions` `condition` 8 + `code_branch` 0. |

## T5 — Embedding and locals (FR-7, inheritance)

Fixtures: `inheritance.go`, `locals.go`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-E1 | unit | `Ledger` in `inheritance.go`. | `inheritance` 3 — one occurrence each on `Base`, `*Printer` and `fmt.Stringer` — `local_variable` 1 (`name`), `stdlib_coupling` 1 (`fmt`). |
| TC-E2 | unit | Pointer embedding `*Printer`. | +1, occurrence on the field's type expression. |
| TC-E3 | unit | Qualified embedding `fmt.Stringer` inside a struct. | `inheritance` 1 **and** `stdlib_coupling` 1 — an embedded qualified type is both an edge and a use of the package. |
| TC-E4 | unit | Generic embedding: `type Box struct { Base[int] }` and `type Node struct { *List[K, V] }`. | +1 each; an `IndexExpr` / `IndexListExpr` type is still an embedded field. |
| TC-E5 | unit | `ReadWriter` in `inheritance.go`. | `inheritance` 2 (`Reader`, `Writer`); the `Close() error` method signature adds nothing — it has `Names`. |
| TC-E6 | unit | `Number interface { ~int \| ~float64 }`, plus `interface { int \| string }` and `interface { ~[]byte }`. | `inheritance` 0 for all three — a union `BinaryExpr` and a tilde `UnaryExpr` are type terms, not embedded interfaces. |
| TC-E7 | unit | `Stringish interface { fmt.Stringer; ~string }`. | `inheritance` 1, `stdlib_coupling` 1 — the named embedded interface counts, the term does not. |
| TC-E8 | unit | `Locals`'s fields: `a, b int` and `c string`. | `local_variable` 3 — one per **name**, one occurrence per name, matching TypeScript's and Java's per-declarator rule. |
| TC-E9 | unit | The anonymous `var cfg struct { Name string }` in `coupling.go`'s `Render`. | `local_variable` 2: `cfg` from the `ValueSpec` and `Name` from the anonymous `StructType`'s field. Every `StructType` inside the unit is counted, not only the unit's own. |
| TC-E10 | unit | `ValueSpec` per name: `var x, y = 1, 2` → 2; `const limit = 10` → 1; a grouped `var ( p int; q, r string )` → 3; `var _ = f()` → 0. | As stated — the blank name is not a variable. |
| TC-E11 | unit | `:=` new names only: `z := x` then `z, err := split(y)`. | 1 and 1 — `z` redeclares (`Obj.Decl` is the first statement), `err` is new. `a, b := 1, 2` is 2; `_, err := f()` is 1. |
| TC-E12 | unit | `RangeStmt` bindings: `for _, x := range xs` → 1; `for k, v := range m` → 2; `for range xs` → 0; `for i = range xs` with `i` already declared (`token.ASSIGN`) → 0. | As stated — counted on the `RangeStmt`, because the resolver gives range bindings no `Obj.Decl` pointing at the statement. |
| TC-E13 | unit | The type-switch guard `switch v := o.(type)`. | `local_variable` 0, while `if s, ok := o.(string); ok` charges 2 — the guard's `AssignStmt` is marked consumed before the generic `:=` rule sees it. |
| TC-E14 | unit | Zero cases: method parameters and results, a receiver, named results (`func named() (n int, err error)`), type parameters (`func F[T any]()`), `FuncLit` parameters, an interface method's parameters, a `LabeledStmt`'s label, and anything at top level. | `local_variable` 0 for every one of them. |
| TC-E15 | unit | Whole-unit totals. | `Locals` `local_variable` 10 + `code_branch` 3; `split` and `named` 0; `Base`, `Printer`, `Reader`, `Writer` and `Number` every metric 0. |

## T6 — Func literals

Fixture: `lambdas.go`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-L1 | unit | `Wire` in `lambdas.go`. | `lambda` 4, `local_variable` 1 (`double` only — `x`, `i` and `j` are `FuncLit` parameters), `stdlib_coupling` 1 (`sort`). |
| TC-L2 | unit | `go func() {}()` and `defer func() {}()`. | +1 each, occurrence on the `FuncLit`, not on the `GoStmt` or `DeferStmt` — a deferred closure is as much a function to read as any other. |
| TC-L3 | unit | `Value` in `lambdas.go` (`return l.Wire`), plus a method expression `Lambdas.Wire` and a plain field read `x.field`. | `lambda` 0 for all three: without types, a method value is indistinguishable from a field. Documented limitation, asserted here so it does not drift into a heuristic. |
| TC-L4 | unit | Nested literals: `f(func() { g(func() {}) })`. | `lambda` 2. |
| TC-L5 | unit | A func literal in a top-level `var f = func() {}` and in a top-level `const`-adjacent declaration. | `lambda` 0 anywhere — a top-level declaration is not a unit and has nothing to bill to (TC-U7). |

## T7 — Coupling (FR-8)

Fixtures: `coupling.go` and `coupling_dot.go`, both with
`InternalPrefixes = ["example.com/app"]`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-C1 | unit | `Invoice` in `coupling.go`. | `internal_coupling` 2 (`money`, `shared`), `external_coupling` 2 (`uuid`, `yaml`), `stdlib_coupling` 4 (`fmt`, `net/http`, `strings` under the alias `str`, and the blank `embed`), `local_variable` 5. |
| TC-C2 | unit | `Note` in `coupling.go`, whose only method is `func (Note) Text(fmt interface{ Sprint() string }) string { return fmt.Sprint() }`. | `0 / 0 / 1` — the blank `embed` alone. The parameter named `fmt` **shadows** the package: its `Ident` resolves (`Obj != nil`), so the `SelectorExpr` is not a package use. This is the rule Java cannot implement and Go gets free. |
| TC-C3 | unit | `Plain` in `coupling.go`, an empty struct with no method. | `0 / 0 / 1` — a blank import is a side effect of the file and charges every unit. |
| TC-C4 | unit | `coupling_dot.go` with `. "math"` and `"example.com/app/shared"`. | `Invoice` `1 / 0 / 1` (`shared`, plus `math` through the dot import, used via the unqualified `Floor`), `Plain` `0 / 0 / 1` — a dot name is indistinguishable from a local, so the module charges every unit. |
| TC-C5 | unit | `classify` table with prefixes `["example.com/app"]`: `fmt`, `net/http`, `go/ast`, `embed`, `unsafe`, `C` → stdlib; `github.com/google/uuid`, `gopkg.in/yaml.v3`, `golang.org/x/sync/errgroup` → external; `example.com/app`, `example.com/app/shared` → internal; `example.com/apples` → external (the prefix must be followed by `/` or end the path); `example.com/appshared` → external. | As stated. With prefixes `["fmt"]`, `fmt` classifies **internal**: a configured prefix always wins over the stdlib rule. |
| TC-C6 | unit | `assumedName` table: `fmt`→`fmt`; `net/http`→`http`; `github.com/google/uuid`→`uuid`; `gopkg.in/yaml.v3`→`yaml`; `github.com/mattn/go-isatty`→`isatty`; `github.com/nats-io/nats.go`→`nats`; `github.com/jackc/pgx/v5`→`pgx`; `example.com/app/money/v2`→`money`; `golang.org/x/sync/errgroup`→`errgroup`. | As stated: `path.Base`, step over a `vN` segment, strip a leading `go-`, cut at the first non-identifier character. |
| TC-C7 | unit | `isStdlib` table: `fmt`, `net/http`, `go/ast`, `embed`, `unsafe`, `C` → true; `example.com/app`, `github.com/x/y`, `golang.org/x/sync`, `gopkg.in/yaml.v3` → false. | The rule is "the first path segment holds no dot", nothing more. Record in a comment that `C` (cgo) is deliberately stdlib. |
| TC-C8 | unit | The same path imported twice: `import ( "a/b"; alias "a/b" )`. | One module, one occurrence, at the **first** `ImportSpec`; a unit that mentions either `b` or `alias` is charged exactly once. |
| TC-C9 | unit | `golang.org/x/sync/errgroup` in `coupling.go`, used only by the top-level `var _ = errgroup.Group{}`. | Charged to **no** unit — not to every unit, and not to the nearest one. Same for an import no code mentions at all, and for one whose assumed name is wrong. |
| TC-C10 | unit | Occurrence placement over `coupling.go`. | Every coupling occurrence's `(Line, Col)` is that of its `ImportSpec` — the alias when there is one — is strictly above the first unit's `Line`, and the occurrences of one unit appear in **import order** after the sort, since `SortOccurrences` orders by position. |
| TC-C11 | unit | One point per module per unit however many uses: a unit that writes `fmt.Sprintf`, `fmt.Errorf` and `fmt.Sprint`. | `stdlib_coupling` 1, one occurrence. |
| TC-C12 | unit | No `InternalPrefixes` at all. | Nothing classifies internal; `example.com/app/shared` becomes external, `fmt` stays stdlib. The analyzer never guesses from the file's own package clause. |
| TC-C13 | unit | Uses that do and do not count: a field type (`uuid.UUID`), a call (`yaml.Unmarshal`), a composite literal (`errgroup.Group{}`), an embedded type (`fmt.Stringer`), a type argument (`[]shared.Row`) and a qualified constant — all count. A mention inside a string literal or a comment, and a local named like the binding used as `str.field` where `str` resolves — do not. | As stated; the resolved-`Ident` rule is what separates the last two from the rest. |

## T8 — End to end

`cmd/check_go_test.go` mirrors `cmd/check_java_test.go`: it reuses
`writeGoFixture` and `runCdd` from `cmd/init_test.go`, and reads the language
id from the registry rather than spelling it out.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-I1 | integration | Clean project: one type of 1 ICP, `cdd check --all`. | Exit 0; the unit is listed with its name, kind `struct` and `icp=1 limit=10`; stderr empty — in particular no "no analyzer for go yet". |
| TC-I2 | integration | Over-limit project: one type whose methods hold six `if a > 0 && b > 0` lines, under greenfield limit 10. | Exit 1; stdout names the unit with `violation:` and `icp=18 limit=10 over=8`; the breakdown line reads `condition=12 code_branch=6`. |
| TC-I3 | integration | `--format json --explain` on that project plus a file importing `net/http`. | Valid JSON; `files[].language` equals the registry's Go id; the unit's `kind` is one of `struct`/`interface`/`type`/`func`/`methods`; one occurrence sits on the `import` line for a coupling metric. |
| TC-I4 | integration | `--format xml` and `--format markdown`. | Both render without error and mention the unit name. |
| TC-I5 | integration | Mixed project: `--languages go,typescript`, one file of each. | Both files reported, each with its own language id; the exit code reflects the worst unit. |
| TC-I6 | integration | A `foo_test.go` holding 50 ICPs next to the source. | Not in the report — the `DefaultExcludes` (`**/*_test.go`) that `init` writes still apply. A `vendor/` file is excluded the same way. |
| TC-I7 | integration | `cdd check internal/x/ledger.go` (path narrowing). | Only that file reported. |
| TC-I8 | integration | A file with a syntax error in an otherwise clean project. | Exit 0: the warning appears in the report, the other files are still analyzed, and a syntax error never changes the exit code by itself. |
| TC-I9 | integration | `internal_coupling.auto_detect: true` on a project whose `go.mod` says `module example.com/app`, with a file importing `example.com/app/shared`. | `internal_coupling=1` without listing the prefix in `packages` — the module line is picked up by `DetectPackages`. |
| TC-I10 | integration | A `.pb.go` carrying the Code-generated header, holding a 40-ICP type. | Absent from the report, with no warning, and the exit code unaffected. |
| TC-I11 | integration | `timeout: 1ms` on a project with several files. | Exit 2 with a partial report — proves the Go analyzer honours the deadline through its `ctx.Err()` check. |

## Cross-cutting invariants (every fixture)

Iterate `internal/analyze/golang/testdata/*.go` plus the two `.txt` fixtures
analyzed under their `.go` names; do not list files by hand.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-X1 | unit | `TestOccurrencesAccountForEveryCount`: for every unit of every fixture, the sum of `Occurrences[i].Count` per metric equals `Counts[metric]`. | Passes. |
| TC-X2 | unit | `TestOccurrencesAreWellFormed`: `Line ≥ 1`, `Col ≥ 1`, `(EndLine, EndCol) > (Line, Col)`, `Count ≥ 1`, `Metric` is a `config.Metrics()` id. | Passes. |
| TC-X3 | unit | `TestOccurrencesAreSorted`: occurrences are non-decreasing by `(Line, Col)`. | Passes — `treesitter.SortOccurrences` is reused unchanged. |
| TC-X4 | unit | `TestNonCouplingOccurrencesAreInsideTheUnit`: every occurrence whose metric is not a coupling metric lies inside the **union of the unit's declaration ranges**, obtained from the in-package `units()` helper — the `TypeSpec` plus every method billed to the unit. | Passes. The union, rather than a single line range, is what `units.go` requires: `Early` is written above `G`'s declaration, and a `methods` unit's members may be scattered through the file. |
| TC-X5 | unit | `TestCouplingOccurrencesSitOnImports`: every `internal_coupling` / `external_coupling` / `stdlib_coupling` occurrence's `Line` is strictly above the first unit's `Line` and matches an `ImportSpec` line of the file. | Passes for both coupling fixtures and vacuously elsewhere. |
| TC-X6 | unit | `TestEveryUnitCarriesEveryMetric`: each unit's `Counts` has exactly `len(config.Metrics())` keys, `exception_handling` at 0 included — disabled metrics are the pipeline's business. | Passes. |
| TC-X7 | unit | Determinism: analyze each fixture twice, with two analyzers built separately. | `reflect.DeepEqual` on the two `analyze.FileResult`s. |
| TC-X8 | unit | Concurrency: eight goroutines, one analyzer each, analyzing every fixture in a loop, under the `-race` detector that `make test` enables. | No race, no panic; the analyzer shares nothing mutable, and each `Analyze` call owns its `token.FileSet`. |

## Coverage targets

- `internal/analyze/golang` ≥ 90 % (statements). Run `make cover` and paste the
  per-function table for the package in the PR description.
- `internal/analyze/java`, `internal/analyze/kotlin`,
  `internal/analyze/typescript`, `internal/analyze/internal/jvm` and
  `internal/languages` coverage must not fall below their pre-T1 values.
