# Feature 04 — Kotlin Support: Test Cases

Companion to [task.md](task.md). Every case here is a constraint the
implementation must satisfy; a task is not done until the cases listed under
it are checked-in tests that pass. Case ids are stable — reference them in
test names or comments (`// TC-U3`) so a reviewer can map the suite back to
this file.

Levels follow CLAUDE.md: **unit** tests exercise one package through its
exported or package-level contract with fixtures on disk; **integration**
tests exercise the CLI boundary (`cmd/check_test.go`: arguments, exit codes,
stdout/stderr, filesystem). No mocks of tree-sitter; the parser is cheap and
deterministic.

## Conventions

Mirror `internal/analyze/typescript/helper_test.go` in
`internal/analyze/kotlin/helper_test.go`:

- `newTestAnalyzer(t, prefixes...)` — builds through `NewAnalyzer`, asserts
  `io.Closer`, closes in `t.Cleanup`.
- `analyzeFixture(t, name, prefixes...)` — reads `testdata/<name>`, analyzes
  under that name, `require.NoError`.
- `unitNamed(t, res, name)` and `requireCount(t, unit, metric, want)` — the
  latter also asserts `len(Counts) == len(config.Metrics())`.

Fixtures live in `internal/analyze/kotlin/testdata/*.kt` and are the files
spelled out in task.md's "Worked fixtures" section, verbatim where a total is
quoted there. One fixture per metric family plus `units.kt`, `broken.kt`,
`cdd_examples.kt`. A fixture line that carries an expected value in a comment
is the source of truth for the assertion next to it.

Metric ids in tests come from `config.Metric…` constants, never string
literals (`make check-literals`).

---

## T1 — Shared tree-sitter helpers (FR-1)

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-R1 | unit | Move the existing TS tests that cover `walk`, `namedChildren*`, `spanOf`, `position`, `firstErrorNode`, `budget`, `timeoutMicros`, `sortedOccurrences` into `internal/analyze/internal/treesitter`. | They pass unchanged apart from package name and receiver. No test is deleted. |
| TC-R2 | unit | Every existing test in `internal/analyze/typescript` still passes. | `go test ./internal/analyze/typescript` green with no assertion edits — only import paths and helper names change. |
| TC-R3 | integration | Before the refactor, run `cdd check --format json` over `internal/analyze/typescript/testdata` with every metric enabled and keep the output. After the refactor, run it again. | Byte-identical JSON (ignoring the `elapsed` field). Add this as a script step in the PR description, not as a permanent test. |
| TC-R4 | unit | `treesitter.Budget(ctx)` with no deadline, with a deadline below the budget, with an expired deadline. | Returns the parse budget, the remaining time, and `time.Microsecond` respectively (existing `TestBudget` semantics). |
| TC-R5 | unit | `go vet` / lint on the new package. | No exported identifier without a doc comment; no `config` import in the package (it is grammar- and metric-agnostic). |

## T2 — Spec corrections (FR-2, FR-3)

One test pins the whole spec rather than one field at a time: a golden
struct fails with a diff naming exactly the field that drifted, and the
registry's `TestSpecCompleteness` already validates shape (non-empty fields,
known metric ids, at least three applicable metrics), so the per-language
test only has to pin values.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-S1 | unit | `TestSpec`: `got := Spec()`; `require.NotNil(got.DetectPackages)`; set `got.DetectPackages = nil`; `assert.Equal(t, want, got)` where `want` is the full `config.LanguageSpec` literal — `ID`, `DisplayName` `"Kotlin"`, `Extensions` `[]string{".kt"}`, `NotApplicable` nil, `DefaultExcludes` `{"**/src/test/**", "**/build/**", "**/target/**"}`, `Descriptions` keyed by `config.Metric…` constants with `code_branch` = `if/when, loops, safe calls (?.)`, `condition` = `&&, \|\| and ?: clauses`, `inheritance` and `lambda` unchanged, `PackageExample` `"com.acme.app"`, `LimitExamples` unchanged. | Equal. Go funcs are not comparable, which is why `DetectPackages` is checked for presence and then cleared on the copy. |
| TC-S2 | unit | Mutation safety: `Spec().Extensions[0] = ".java"` then `Spec().Extensions` again; same for `DefaultExcludes[0]` and a `Descriptions` entry. | Second call returns the original values — `Spec()` must `slices.Clone` its slices and copy its map, as the TypeScript spec does. Today `Extensions: extensions` hands out the package variable; T2 fixes that. |
| TC-S3 | unit | `TestDetectPackagesSkipsJavaFiles` (existing) | Still returns `["com.acme.billing", "com.acme.shared"]` — `build.gradle.kts` was never contributing a package, so dropping `.kts` changes nothing. |
| TC-S4 | unit | Create a temp project holding only `Main.kts` with a `package a.b` line. | `DetectPackages` returns empty: `.kts` is no longer read. |
| TC-S5 | integration | `cdd init --languages kotlin` golden in `cmd/testdata/golden` and the `internal/config` render goldens. | Regenerated; the rendered `code_branch` comment for Kotlin no longer mentions `?:`, the `condition` comment does. CI dogfood gate green. |
| TC-S6 | unit | `internal/languages` `TestSpecCompleteness`, `TestLiterals` | Still green — the literal `".kt"` and the description strings live only in `spec.go`. |

## T3 — Parsing (FR-6, FR-12)

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-P1 | unit | `TestGrammarResolvesEveryKind`: for every entry in `kindNames`, `lang.IdForNodeKind(name, true) != 0`. | Passes. A grammar bump that renames a node fails here with the node named. |
| TC-P2 | unit | `TestFieldsResolve`: `condition`, `name`, `type`, `operator`, `left`, `right`, `argument` resolve to non-zero field ids. | Passes. Also assert that the fields the analyzer must **not** rely on (`alternative`, `consequence`, `body`, `receiver`, `delegate`) resolve to 0, so nobody starts using them without noticing the grammar lacks them. |
| TC-P3 | unit | `TestTokensResolve`: anonymous `?.`, `*`, `else`, `val`, `var`, `interface`, `enum` resolve with `IdForNodeKind(name, false) != 0`. | Passes. |
| TC-P4 | unit | `TestKindOfOutOfRange`: a synthetic node id beyond `len(byID)` | Returns `kindOther`, no panic (mirror the TS test). |
| TC-P5 | unit | Analyze `broken.kt`. | `Units` empty, `Warnings` is exactly one string with prefix `syntax error at ` followed by `L:C`, and `L:C` points at the `ERROR`/`MISSING` node's 1-based position. `err == nil`. |
| TC-P6 | unit | Analyze a source whose root `HasError()` is true but which contains no `ERROR`/`MISSING` node (the grammar does this for `interface I { val x: Int\n fun g() }`). | Warning is `syntax error at 1:1`. Document the source in the test so the case is re-checked on a grammar bump. |
| TC-P7 | unit | Analyze a path with an extension other than `.kt` (e.g. `a.kts`, `a.java`). | Returns an error naming the path and the extension; no units. |
| TC-P8 | unit | `TestReanalyzeDoesNotLeak`: analyze the same fixture 200 times with one analyzer. | No growth in the number of open trees; no panic. (Mirror the TS test's approach.) |
| TC-P9 | unit | `TestCloseIsIdempotent`: `Close()` twice, then `Analyze`. | Second `Close` returns nil; `Analyze` after close returns an error mentioning "closed", not a panic. |
| TC-P10 | unit | `TestCanceledContext`: already-canceled context. | `Analyze` returns `context.Canceled` wrapped, no units. |
| TC-P11 | unit | `TestAnalyzersShareOnlyImmutableGrammarMetadata`: two analyzers from `NewAnalyzer`. | Distinct `parser` pointers; same `grammar` pointer. |
| TC-P12 | unit | Empty file and a file holding only `package a.b` and imports. | No units, no warnings, no error. |
| TC-P13 | unit | A file with a UTF-8 BOM and CRLF line endings. | Parses; positions are 1-based lines unaffected by the BOM byte count. (Kotlin sources from Windows tooling do this.) |

## T4 — Units (FR-4)

Fixture: `units.kt` from task.md.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-U1 | unit | Names in order. | Exactly `A B C D E F G H shout I j k l m O` — no `n`, no `P`, `q`, `R`, `s`, `local`. |
| TC-U2 | unit | Kinds. | `A`→class, `B`→interface, `C`→enum, `D`→class, `E`→class, `F`→class, `G`→object, `h`→function, `shout`→function, `I`→typealias, `j`→property, `k`→property, `l`→property, `m`→function, `O`→class. |
| TC-U3 | unit | `sealed interface S` and `fun interface Fn { fun call() }` added to the fixture. | Both kind `interface`. |
| TC-U4 | unit | Positions. | `Line`/`Col` of `private fun m()` point at `fun`, not `private`; of `data class E` at `class`, not `data`; of `enum class C` at `class`, not `enum`; of `val j = { 1 }` at `val`. |
| TC-U5 | unit | Extension function name. | `fun String.shout()` is named `shout`; `fun List<Map<String, Int>>.flat()` is named `flat` (receiver with nested generics does not leak into the name). |
| TC-U6 | unit | Top-level property with a plain value, a call initializer, a string template, an object expression. | None is a unit: `val n = 4`, `val cfg = load()`, `val s = "$n"`, `val o = object : Runnable { override fun run() {} }`. |
| TC-U7 | unit | Top-level property with a lambda, with an anonymous function, with a getter body, with a setter body, with `by lazy`. | All five are units of kind `property`. |
| TC-U8 | unit | Nested declarations. | `class O` is one unit; its counts include everything inside `P`, the companion, `R`, `s` and `local`. Add `if (x) {}` inside `local()` and assert `O`'s `code_branch` sees it. |
| TC-U9 | unit | Two files' worth of declarations under different visibilities: `private`, `internal`, `public`, `protected` (inside a class — bills to the class). | Top-level visibility never filters a unit. |
| TC-U10 | unit | `TestUnitsAreInSourceOrder`: shuffle-proof assertion that `Line` is strictly increasing across `Units`. | Passes. |
| TC-U11 | unit | An `object` with a body and an `object` declared as `companion object` at top level (invalid Kotlin — grammar may still parse). | Only the real top-level `object_declaration` is a unit; a `companion_object` node at top level, if the grammar produces one, is not. Document what the grammar does. |

## T5 — Branches and conditions (FR-9, FR-10)

Fixtures: `cdd_examples.kt`, `branches.kt`, `conditions.kt`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-B1 | unit | `check` in `cdd_examples.kt` | `code_branch` 1, `condition` 2 — the doc's `if (a > b && c < d)` = 3. |
| TC-B2 | unit | `ifElse` | `code_branch` 2 — the doc's "if-else = 2". |
| TC-B3 | unit | `chain` (`if / else if / else if / else`) | `code_branch` 4, not 6. Occurrences: three on `if` keywords and one on the final `else` branch. |
| TC-B4 | unit | `describe` (`when` with subject, `2, 3 ->` arm, `else`) | `code_branch` 2. Two occurrences, on the two conditioned `when_entry` nodes; none on the `else` entry. |
| TC-B5 | unit | `when` without subject: `when { a > 1 -> x(); b -> y(); else -> z() }` | `code_branch` 2, `condition` 0 (`a > 1` is a comparison, not a clause). |
| TC-B6 | unit | `when` as expression used as a function body and `when` inside a lambda inside a class. | Both counted, attributed to the enclosing unit. |
| TC-B7 | unit | `safe`: `s?.trim()?.length ?: 0` | `code_branch` 2, `condition` 2. |
| TC-B8 | unit | `a?.b?.c?.d` versus `a.b.c.d` | 3 versus 0. The `.` token never counts. |
| TC-B9 | unit | `x?.let { }` | `code_branch` 1 (+ `lambda` 1 in T7). |
| TC-B10 | unit | `x!!.y` and `x!!` alone | `code_branch` 0. |
| TC-B11 | unit | `loops`: `for (x in xs)`, `for ((k, v) in m)`, `while`, `do … while` | `code_branch` 4, one occurrence per statement on the keyword-anchored node. |
| TC-B12 | unit | Clause counting: `a && b` → 2; `a && b \|\| c` → 3; `(a && b) \|\| !(c \|\| d)` → 4; `!a` → 0; `a == b` → 0; `x ?: y` → 2; `x ?: y ?: z` → 3. | Each in its own function in `conditions.kt`; assert per function. |
| TC-B13 | unit | Nested chain in the middle of an argument list: `f(a && b, c \|\| d)` | `condition` 4. No clause counted twice (the `consumed` set works). |
| TC-B14 | unit | `if (a && b) x else y` | `code_branch` 2, `condition` 2; the `if` occurrence sorts before its own clauses' occurrences (stable sort by position). |
| TC-B15 | unit | Elvis whose right side is a `throw` or `return`: `x ?: return`, `x ?: throw E()` | `condition` 2 each. `return`/`throw` are not branches. |
| TC-B16 | unit | `try` used as an expression: `val v = try { a() } catch (e: E) { 0 }` | `exception_handling` 2 (checked in T6), `code_branch` 0, `local_variable` 1. |

## T6 — Exceptions, inheritance, locals

Fixtures: `cdd_examples.kt`, `inheritance.kt`, `locals.kt`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-E1 | unit | `guarded` | `exception_handling` 3: occurrences on the try `block`, the `catch_block`, the `finally_block`. The try occurrence's range does **not** cover the catch. |
| TC-E2 | unit | `try` with two `catch_block`s and no `finally` | 3. |
| TC-E3 | unit | `try` with `finally` only | 2. |
| TC-E4 | unit | `runCatching { }` | `exception_handling` 0, `lambda` 1. Syntactic analyzer: library calls are not blocks. |
| TC-I1 | unit | `Ledger` in `inheritance.kt` | `inheritance` 3; three occurrences whose text is `Base`, `Auditable`, `Printer` (the `user_type`, not the whole specifier). |
| TC-I2 | unit | `interface Auditable : Named, Timestamped` | 2. |
| TC-I3 | unit | `object Registry : Auditable` | 1. |
| TC-I4 | unit | `class A` with no `:` | 0. |
| TC-I5 | unit | `class Outer { inner class In : Base() }` | `Outer` has `inheritance` 1 — nested heritage bills to the enclosing unit. |
| TC-I6 | unit | `enum class Color : Named { RED }` and `class X : Generic<String>()` | 1 each; type arguments do not add. |
| TC-I7 | unit | Anonymous object inside a function: `val r = object : Runnable { … }` | `inheritance` 1 on the enclosing unit (the object expression's `delegation_specifier`), `local_variable` 1. State this choice in a comment: an anonymous object implements the interface as much as a named one. |
| TC-L1 | unit | `data class E(val v: Int)` | `local_variable` 1. |
| TC-L2 | unit | `class P(val a: Int, var b: Int, c: Int)` | 2 — `c` is a parameter. |
| TC-L3 | unit | Locals in a function body: `val x = 1; var y = 2; val (p, q) = pair; lateinit var z: String` | 4 (destructuring is 1). |
| TC-L4 | unit | Class-body properties, including one with a getter and one with `by lazy`. | Each is 1. |
| TC-L5 | unit | `for (x in xs)` and `for ((k, v) in m)` | 1 each, occurrence on the binding, not the whole statement. |
| TC-L6 | unit | `interface I { val x: Int; val y: Int get() = 1; fun f() }` (multi-line) | `local_variable` 1 — `x` is a shape, `y` has an accessor body. |
| TC-L7 | unit | `abstract class A { abstract val x: Int }` | 1 — the interface exemption applies to `interface` units only. Document the asymmetry in the test. |
| TC-L8 | unit | `enum class C { X, Y, Z }` | 0. |
| TC-L9 | unit | Function parameters, lambda parameters (`{ a, b -> }`), `catch (e: E)` binding, `it`. | 0. |
| TC-L10 | unit | The `property` unit's own declaration (`val j = { 1 }`) | 0 on unit `j` (FR-11). |
| TC-L11 | unit | Property inside a companion object and inside an `init` block's local scope. | Both 1, on the enclosing class. |

## T7 — Lambdas (FR-11)

Fixture: `lambdas.kt`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-F1 | unit | `total` | `lambda` 4: two trailing lambdas, one `::render`, one anonymous function. |
| TC-F2 | unit | Scope functions: `x.let { }`, `x.apply { }`, `x.also { }`, `x.run { }`, `with(x) { }`, `x.takeIf { }` | 1 each — no exemption. |
| TC-F3 | unit | `onEvent` property unit | `lambda` 0 on `onEvent`. |
| TC-F4 | unit | `val l by lazy { 3 }` unit | `lambda` 1 — the lambda is `lazy`'s argument, not the unit's own body. |
| TC-F5 | unit | Lambda nested in lambda: `xs.map { it.filter { it > 0 } }` | 2. |
| TC-F6 | unit | `String::trim` | 0, with a test comment naming the grammar quirk (`navigation_expression`), so a future grammar bump that fixes it is noticed when the count becomes 1 and the assertion is updated deliberately. |
| TC-F7 | unit | `::render` and `this::render` and `Foo::class` | `::render` 1; assert whatever the grammar produces for the other two and pin it, with a comment. `Foo::class` is a class literal and should be 0 if the grammar distinguishes it. |
| TC-F8 | unit | Lambda assigned to a class property inside a class: `class A { val f = { 1 } }` | `lambda` 1 and `local_variable` 1 on `A` — only top-level `property` units skip their own body. |
| TC-F9 | unit | Suspend lambda / inline lambda arguments: `launch { }`, `withContext(Dispatchers.IO) { }` | 1 each — modifiers on the callee do not matter. |

## T8 — Coupling (FR-7, FR-8)

Fixture: `coupling.kt` with `InternalPrefixes = ["com.acme"]`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-C1 | unit | `Invoice` | `internal_coupling` 1, `external_coupling` 2. |
| TC-C2 | unit | `Note` | `internal_coupling` 1 (alias `L`), `external_coupling` 1 (star). |
| TC-C3 | unit | `Plain` | `internal_coupling` 0, `external_coupling` 1 — the star charges every unit. |
| TC-C4 | unit | Remove the star import from the fixture in a second file. | `Plain` drops to 0/0; `Invoice` to 1/1. |
| TC-C5 | unit | Occurrence position. | Every coupling occurrence's `Line` is that of the `import` line, above the unit's own `Line`. |
| TC-C6 | unit | `isInternal` table: `com.acme.shared.Money` with prefixes `["com.acme"]` → internal; `["com.acme.shared"]` → internal; `["com.acmecorp"]` → external; `["com.acme.shared.Money"]` (exact) → internal; empty prefix `""` → ignored, external; `java.util.List` → external. | Table-driven, pure function, no parsing. |
| TC-C7 | unit | Two imports of the same path (`import a.B` twice, or `import a.B` and `import a.B as C`). | One module; charged once to a unit that uses either binding. |
| TC-C8 | unit | Import used only in a type position (`val m: Money`) versus only in a call (`Money.of(1)`) versus in an annotation (`@Inject`). | All three count as a use. |
| TC-C9 | unit | Import whose binding the unit never mentions. | Not charged to that unit; charged to a sibling unit that does mention it. |
| TC-C10 | unit | Shadowing: `import a.Money` and a local `val Money = 1` in a unit that never uses the imported one. | Charged (by-name test). Document as the same caveat TS carries. |
| TC-C11 | unit | No `InternalPrefixes` at all. | Every import is external; the analyzer never guesses from the file's own `package`. |
| TC-C12 | unit | Same-package class used with no import. | 0 — pinned as the documented blind spot. |
| TC-C13 | unit | `import a.b.C` where the unit mentions `C` only inside a string literal or a comment. | 0 — string contents and comments are not `identifier` nodes. |

## Cross-cutting invariants (every fixture)

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-X1 | unit | `TestOccurrencesAccountForEveryCount`: for every unit in every fixture, sum of `Occurrences[i].Count` per metric equals `Counts[metric]`. | Passes for every `.kt` under `testdata` — iterate the directory, do not list files. |
| TC-X2 | unit | `TestOccurrencesAreWellFormed`: `Line ≥ 1`, `Col ≥ 1`, `(EndLine, EndCol) > (Line, Col)`, `Count ≥ 1`, `Metric` is a known id. | Passes for every fixture. |
| TC-X3 | unit | `TestOccurrencesAreSorted`: occurrences are non-decreasing by `(Line, Col)`. | Passes for every fixture. |
| TC-X4 | unit | Every unit's `Counts` has exactly `len(config.Metrics())` keys (disabled metrics are the pipeline's business). | Passes. |
| TC-X5 | unit | Determinism: analyze each fixture twice with two analyzers. | `reflect.DeepEqual` on the results. |
| TC-X6 | unit | `-race` is on in `make test`; run a small parallel test that builds one analyzer per goroutine and analyzes concurrently. | No race reported; the shared grammar is read-only. |

## T9 — End to end (FR-13)

In `cmd/check_test.go`, following the `writeTSFixture` pattern with a
`writeKotlinFixture` that lays out `src/main/kotlin/com/acme/app/*.kt` and a
config produced by `cdd init --languages kotlin`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-Z1 | integration | Clean project: one class of 1 ICP. | Exit 0; console output lists the unit with `unit:`. |
| TC-Z2 | integration | Over-limit project: one class of 18 ICPs (six `if (a && b)` lines) under greenfield limit 10. | Exit 1; console output names `OrderService` with `violation:`. |
| TC-Z3 | integration | `--format json` on the over-limit project. | Valid JSON; `files[0].language == "kotlin"` (compare against `string(config.Language…)` through the registry, not a literal); the unit's `kind` is `class`. |
| TC-Z4 | integration | `--format xml` and `--format markdown`. | Both render without error and mention the unit name. |
| TC-Z5 | integration | Mixed project: `--languages kotlin,typescript`, one file of each. | Both files reported; each with its own language; exit code reflects the worst unit. |
| TC-Z6 | integration | A `.kts` file next to the `.kt` sources. | Not in the report at all. |
| TC-Z7 | integration | `src/test/kotlin/...` file with 50 ICPs. | Not in the report: `DefaultExcludes` written by `init` still applies. |
| TC-Z8 | integration | `cdd check src/main/kotlin/com/acme/app/Ledger.kt` (path narrowing). | Only that file reported. |
| TC-Z9 | integration | `cdd check --explain` | Occurrences present in JSON, one of them on an `import` line for a coupling metric. |
| TC-Z10 | integration | A file with a syntax error in the project. | Exit code unchanged by it; the warning appears in the report; other files still analyzed. |
| TC-Z11 | integration | `internal_coupling.auto_detect: true` with the `jvm` testdata layout. | `com.acme.*` imports classified internal without listing them in `packages`. |
| TC-Z12 | integration | Timeout: `timeout: 1ms` on a project with several files (reuse the existing partial-report test approach). | Exit 2 with a partial report — proves the Kotlin analyzer honours the deadline through `parse`. |

## Registry and lint gates

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-G1 | unit | `internal/languages` `TestEveryLanguageHasADirectory`, `TestEveryDirectoryIsRegistered`, `TestSpecCompleteness` | Green after the one-line registration. |
| TC-G2 | unit | `make check-literals` | Green: no `"kotlin"`, no metric id string, no `".kt"` outside `spec.go`; `cmd/check_test.go` obtains ids through `config`/registry. |
| TC-G3 | lint | `make lint` | `0 issues` — in particular `funlen`/`gocyclo` on the counter switch and `lll` on the fixture-heavy tests. |
| TC-G4 | build | `make build` | Binary builds with CGO; note its size delta in the PR description against `main`. |

## Coverage targets

- `internal/analyze/kotlin` ≥ 90 % (statements). Run `make cover` and paste
  the per-function table for the package in the PR description.
- `internal/analyze/internal/treesitter` ≥ 90 %.
- `internal/analyze/typescript` coverage must not fall below its pre-T1 value.
