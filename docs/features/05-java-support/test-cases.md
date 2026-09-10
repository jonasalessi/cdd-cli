# Feature 05 — Java Support: Test Cases

Companion to [task.md](task.md). Every case here is a constraint the
implementation must satisfy; a task is not done until the cases listed under it
are checked-in tests that pass. Case ids are stable — reference them in test
names or comments (`// TC-U3`) so a reviewer can map the suite back to this
file.

Levels follow CLAUDE.md: **unit** tests exercise one package through its
exported or package-level contract with fixtures on disk; **integration** tests
exercise the CLI boundary (`cmd/check_java_test.go`: arguments, exit codes,
stdout/stderr, filesystem). No mocks of tree-sitter; the parser is cheap and
deterministic.

## Conventions

Mirror `internal/analyze/kotlin/helper_test.go` in
`internal/analyze/java/helper_test.go`:

- `newTestAnalyzer(t, prefixes...)` — builds through `NewAnalyzer`, asserts
  `io.Closer`, closes in `t.Cleanup`.
- `analyzeFixture(t, name, prefixes...)` — reads `testdata/<name>`, analyzes
  under that name, `require.NoError`.
- `unitNamed(t, res, name)` and `requireCount(t, unit, metric, want)` — the
  latter also asserts `len(Counts) == len(config.Metrics())`.

Fixtures live in `internal/analyze/java/testdata/*.java` and are the files
spelled out in task.md's "Worked fixtures" section, verbatim where a total is
quoted there. One fixture per metric family plus `units.java`, `compact.java`,
`broken.java`, `empty.java`, `header_only.java`, `cdd_examples.java`. A fixture
line that carries an expected value in a comment is the source of truth for the
assertion next to it.

Metric ids in tests come from `config.Metric…` constants, never string literals
(`make check-literals`).

---

## T1 — Shared JVM import attribution (FR-1)

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-R1 | unit | Move Kotlin's `isInternal` table test into `internal/analyze/internal/jvm/imports_test.go` as `TestIsInternal`. | Passes unchanged apart from package name and the exported `IsInternal` spelling. No case is deleted. |
| TC-R2 | unit | `jvm.Imports`: `Bind("a.B", "B", at)` twice for the same path, then `Bind("a.B", "C", at2)`. | `Modules()` has one entry with `Path` `a.B`, `Bindings` `["B", "B", "C"]` (or deduped — pin whichever the implementation chooses), and `At` from the **first** import. |
| TC-R3 | integration | Before the refactor, run `cdd check --format json` over the Kotlin fixture project with every metric enabled and keep the output. After the refactor, run it again. | Byte-identical JSON (ignoring the `elapsed` field). A script step in the PR description, not a permanent test. |
| TC-R4 | unit | `Module.UsedBy(refs)`: star module with empty refs; named module whose binding is in refs; named module whose bindings are all absent. | `true`, `true`, `false`. |
| TC-R5 | unit | `Module.Metric()` with `Internal` true and false. | `config.MetricInternalCoupling` and `config.MetricExternalCoupling`. |
| TC-R6 | unit | Every existing test in `internal/analyze/kotlin` still passes. | `go test ./internal/analyze/kotlin` green with no assertion edits — only the import path and the `module`→`jvm.Module` spelling change. |
| TC-R7 | unit | `go vet` / lint on `internal/analyze/internal/jvm`. | No exported identifier without a doc comment; the package imports `config` and `treesitter` only, no language package. |

## T2 — Spec (FR-2)

One test pins the whole spec rather than one field at a time: a golden struct
fails with a diff naming exactly the field that drifted, and the registry's
`TestSpecCompleteness` already validates shape, so the per-language test only
has to pin values.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-S1 | unit | `TestSpec`: `got := Spec()`; `require.NotNil(got.DetectPackages)`; set `got.DetectPackages = nil`; `assert.Equal(t, want, got)` where `want` is the full `config.LanguageSpec` literal — `ID` `"java"`, `DisplayName` `"Java"`, `Extensions` `[]string{".java"}`, `NotApplicable` nil, `DefaultExcludes` `{"**/src/test/**", "**/build/**", "**/target/**"}`, `Descriptions` keyed by `config.Metric…` with `internal_coupling` = `references to project classes` and `external_coupling` = `framework / JDK types`, `PackageExample` `"com.acme.app"`, `LimitExamples` unchanged. | Equal. This replaces `TestSpecID`. Go funcs are not comparable, which is why `DetectPackages` is checked for presence and then cleared on the copy. |
| TC-S2 | unit | Mutation safety: `Spec().Extensions[0] = ".kt"`, `Spec().DefaultExcludes[0] = "**/nothing/**"`, `Spec().Descriptions[config.MetricInternalCoupling] = "changed"`, then call `Spec()` again. | The second call returns the original values — `Spec()` must `slices.Clone` its slices and copy its map. Today `Extensions: extensions` hands out the package variable. |
| TC-S3 | unit | `TestDetectPackagesSkipsKotlinFiles` and `TestDetectPackagesEmptyProject` (existing). | Still return `["com.acme.billing", "com.acme.shared"]` and empty — FR-2 changes no detection behaviour. |
| TC-S4 | unit | `DetectPackages` on a temp project holding only `Main.kt` with a `package a.b` line. | Empty: `.kt` is not a Java extension. |
| TC-S5 | unit | `internal/languages` `TestSpecCompleteness` and `TestLiterals`. | Still green — `".java"`, `"java"` and the description strings live only in `spec.go`. |

## T3 — Parsing (FR-5, FR-11)

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-P1 | unit | `TestGrammarResolvesEveryKind`: for every entry in `kindNames`, `lang.IdForNodeKind(name, true) != 0`. | Passes. A grammar bump that renames a node fails here with the node named. |
| TC-P2 | unit | `TestFieldsResolve`: `name`, `alternative`, `consequence`, `condition`, `operator`, `left`, `right`, `operand`, `body`, `type`, `parameters`, `declarator`, `value`, `scope` resolve to non-zero field ids. | Passes. Unlike Kotlin, every field the analyzer wants exists; nothing is reached by child position where a field is listed here. |
| TC-P3 | unit | `TestTokensResolve`: anonymous `else` and `default` resolve with `IdForNodeKind(name, false) != 0`. | Passes. |
| TC-P4 | unit | `TestKindOfOutOfRange`: a synthetic node id beyond `len(byID)`, and a negative one. | Returns `kindOther`, no panic (mirror the Kotlin test). |
| TC-P5 | unit | Analyze `broken.java` (`class Oops { void f( { }`). | `Units` empty, `Warnings` exactly one string with prefix `syntax error at ` followed by `L:C` pointing at the 1-based position of the first `ERROR`/`MISSING` node. `err == nil`. |
| TC-P6 | unit | Analyze a source whose root `HasError()` is true but which contains no `ERROR`/`MISSING` node, if one can be found for this grammar. | Warning is `syntax error at 1:1`. If no such source exists for Java, assert the fallback through the helper directly and say so in a test comment. |
| TC-P7 | unit | Analyze `empty.java` (zero bytes) and `header_only.java` (package plus imports). | No units, no warnings, no error, for both. |
| TC-P8 | unit | Analyze a path whose extension is not `.java` (`a.kt`, `a.jav`, `a`). | Returns an error naming the path and the extension; no units. |
| TC-P9 | unit | Analyze `A.JAVA` and `A.Java`. | Accepted: the extension test is case-insensitive. |
| TC-P10 | unit | `TestReanalyzeDoesNotLeak`: analyze the same fixture 200 times with one analyzer. | No growth in the number of open trees; no panic. (Mirror the Kotlin test's approach.) |
| TC-P11 | unit | `TestCloseIsIdempotent`: `Close()` twice, then `Analyze`. | Second `Close` returns nil; `Analyze` after close returns an error mentioning "closed", not a panic. |
| TC-P12 | unit | `TestCanceledContext`: already-canceled context. | `Analyze` returns `context.Canceled` wrapped, no units. |
| TC-P13 | unit | `TestAnalyzersShareOnlyImmutableGrammarMetadata`: two analyzers from `NewAnalyzer`. | Distinct `parser` pointers; same `grammar` pointer. |
| TC-P14 | unit | A file with a UTF-8 BOM and CRLF line endings. | Parses; positions are 1-based lines unaffected by the BOM byte count. |
| TC-P15 | unit | A `module-info.java` holding `module com.acme { requires java.base; }`. | Parses without a warning; no units; `requires` charges no coupling. |

## T4 — Units (FR-3)

Fixtures: `units.java`, `compact.java`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-U1 | unit | Names in order, `units.java`. | Exactly `A B C D E F G` — seven units, no `Inner`, `Nested`, `Shape`, `Kind`, `R`, `m` or `Local`. |
| TC-U2 | unit | Kinds, `units.java`. | `A`→class, `B`→interface, `C`→enum, `D`→record, `E`→annotation, `F`→class, `G`→class. |
| TC-U3 | unit | Positions. | `public class A` points at `class`, not `public`; `abstract class F` at `class`, not `abstract`; `@interface E` at `@interface`; `record D` at `record`. Modifiers and annotations never move the position. |
| TC-U4 | unit | `compact.java`. | One unit, `Name` `main`, `Kind` `method`, `Line`/`Col` on the `void` return type. The top-level `int count = 1;` is not a unit and adds no `local_variable` to `main`. |
| TC-U5 | unit | Nested declarations bill to the enclosing unit. | `G`'s counts include the nested `record R`'s component (`local_variable` 1). Add `if (v > 0) {}` inside `m()` and assert `G`'s `code_branch` sees it. |
| TC-U6 | unit | Visibility. | `public`, `protected`, `private` and package-private top-level types are all units; a `private` nested class is still not one. No visibility filter anywhere. |
| TC-U7 | unit | `TestUnitsAreInSourceOrder`: `Line` is strictly increasing across `Units`. | Passes for every fixture. |
| TC-U8 | unit | A file with a `package_declaration`, several `import_declaration`s and a `module_declaration`. | None of them is a unit. |
| TC-U9 | unit | A generic type: `class Box<T extends Number> {}`. | One unit named `Box` — the type parameters do not leak into the name. |
| TC-U10 | unit | An annotated type: `@Deprecated @SuppressWarnings("x") class H {}`. | One unit named `H`, position at `class`. |

## T5 — Branches and conditions (FR-8, FR-9, FR-10)

Fixtures: `cdd_examples.java`, `branches.java`, `conditions.java`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-B1 | unit | `check` in `cdd_examples.java`. | `code_branch` 1, `condition` 2 — the doc's `if (a > b && c < d)` = 3. |
| TC-B2 | unit | `ifElse` in `cdd_examples.java`. | `code_branch` 2 — the doc's "if-else = 2". |
| TC-B3 | unit | `chain` in `branches.java` (`if / else if / else if / else`). | `code_branch` 4, not 7. Occurrences: three on the `if` statements and one on the final `else` branch, whose range starts at the `else` token. |
| TC-B4 | unit | `if (a) x(); else if (b) y();` with no trailing `else`. | `code_branch` 2. |
| TC-B5 | unit | `oldSwitch` in `branches.java`. | `code_branch` 2: one occurrence on the `case 1:` group, one on the `case 2: case 3:` group, none on `default:`. |
| TC-B6 | unit | `arrowSwitch` in `branches.java`. | `code_branch` 2: one on `case 1 ->`, one on `case 2, 3 ->`, none on `default ->`. |
| TC-B7 | unit | A switch used as an expression assigned to a local, and one used as a statement, with the same arms. | Same `code_branch` either way — `switch_expression` covers both. |
| TC-B8 | unit | A switch with only a `default` arm. | `code_branch` 0. |
| TC-B9 | unit | A pattern switch: `case Circle c -> …`, `case Square s -> …`, `default -> …`. | `code_branch` 2, `local_variable` 0 — pattern bindings are not variables. |
| TC-B10 | unit | `ternary` in `branches.java`, and a nested ternary `a ? b : c ? d : e`. | 1 and 2. |
| TC-B11 | unit | `loops` in `branches.java`. | `code_branch` 4 and `local_variable` 2: one occurrence per loop statement, locals on `i` and on `x`. |
| TC-B12 | unit | The whole `Branches` unit. | `code_branch` 13, `local_variable` 2, `condition` 0. |
| TC-B13 | unit | Clause counting per method in `conditions.java`: `both` → 2; `either` → 3; `negated` → 3; `compare` → 0; `bits` → 0. | Each asserted against its own method's contribution; the unit total is `condition` 8. |
| TC-B14 | unit | Nested chain in an argument list: `f(a && b, c || d)`. | `condition` 4. No clause counted twice (the `consumed` set works). |
| TC-B15 | unit | `if (a && b) x(); else y();`. | `code_branch` 2, `condition` 2; the `if` occurrence sorts before its own clauses' occurrences (stable sort by position). |
| TC-B16 | unit | Statements that are not branches: `return`, `break`, `continue`, `throw`, `yield`, `assert`, a labelled `break label;`, `o instanceof String`. | `code_branch` 0 for a method containing only those (the `instanceof` alone, without an `if`, adds nothing). |
| TC-B17 | unit | A `try`/`catch` used around a loop. | `code_branch` counts the loop only; `try` is `exception_handling` (T6), never a branch. |

## T6 — Exceptions, inheritance and locals

Fixtures: `cdd_examples.java`, `exceptions.java`, `inheritance.java`,
`locals.java`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-E1 | unit | `guarded` in `cdd_examples.java`. | `exception_handling` 3: occurrences on the `try` body block, the `catch_clause` and the `finally_clause`. The try occurrence's range does **not** cover the catch. |
| TC-E2 | unit | `try` with two `catch_clause`s and no `finally`. | 3. |
| TC-E3 | unit | `try` with `finally` only. | 2. |
| TC-E4 | unit | `guard` in `exceptions.java`: try-with-resources plus multi-catch. | `exception_handling` 2 — a `try_with_resources_statement` is one point like a plain `try`, and `catch (IOException \| RuntimeException e)` is one clause. |
| TC-E5 | unit | Resources: `try (var in = open(p); existing)`. | `local_variable` 1 — `in` declares a name, `existing` names an already-declared variable. |
| TC-E6 | unit | `throw new IllegalStateException(e);` and a method with `throws IOException`. | `exception_handling` 0 and `code_branch` 0 for both. |
| TC-E7 | unit | `Ledger` in `inheritance.java`. | `inheritance` 3; three occurrences whose text is `Base`, `Auditable`, `Printer`, one per type in the heritage. |
| TC-E8 | unit | `interface Auditable extends Named, Timestamped`. | `inheritance` 2 — `extends_interfaces` counts per type, like `implements`. |
| TC-E9 | unit | `enum Level implements Auditable { LOW, HIGH }`. | `inheritance` 1, `local_variable` 0. |
| TC-E10 | unit | `record Money(int amount, String currency) implements Comparable<Money>`. | `inheritance` 1 (type arguments do not add), `local_variable` 2 (the record components). |
| TC-E11 | unit | `Factory` in `inheritance.java`: a field initialised with `new Runnable() { … }`. | `inheritance` 1, occurrence on the `Runnable` type of the `object_creation_expression`; `local_variable` 1 (the field); `lambda` 0. Say in a test comment that an anonymous class implements the interface as much as a named one. |
| TC-E12 | unit | `sealed interface Shape permits Circle`. | `inheritance` 0. |
| TC-E13 | unit | `class Outer { class In extends Base {} }`. | `Outer` has `inheritance` 1 — nested heritage bills to the enclosing unit. |
| TC-E14 | unit | `class A {}` with no `extends` and no `implements`. | `inheritance` 0. |
| TC-E15 | unit | `class Loose {}` plus `new Runnable() {}` twice in one method. | `inheritance` 2 — one per anonymous class. |
| TC-E16 | unit | `Locals` in `locals.java`: fields `private int a, b;` and `static final int MAX = 3;`. | `local_variable` 3 — one per `variable_declarator`, `field_declaration` and `constant_declaration` alike. |
| TC-E17 | unit | `body` in `locals.java`: `int x = 1, y = 2;` and `var z = x;`. | `local_variable` 3. The `Locals` unit total is 6. |
| TC-E18 | unit | Pattern variable `if (o instanceof String s)` and catch parameter `catch (Exception e)`. | `local_variable` 0 each; the `instanceof` line still charges `code_branch` 1 through its `if`. |
| TC-E19 | unit | `interface Consts { int LIMIT = 10; }`. | `local_variable` 1 — an interface constant is a `constant_declaration`, and unlike Kotlin there is no bodiless-property shape to exempt. |
| TC-E20 | unit | `enum Color { RED, GREEN }`. | `local_variable` 0 — `enum_constant` is not a variable. |
| TC-E21 | unit | Method parameters, constructor parameters, lambda parameters and a `void f(String... xs)` spread parameter. | `local_variable` 0 for all of them. |
| TC-E22 | unit | `for (Integer x : xs)`. | `local_variable` 1, occurrence on the binding `x`, not on the whole statement. |
| TC-E23 | unit | A field declared inside a nested class and a local declared inside an initializer block. | Both 1, on the enclosing top-level unit. |

## T7 — Lambdas

Fixture: `lambdas.java`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-L1 | unit | `wire` in `lambdas.java`. | `lambda` 4: `x -> x * 2`, `String::valueOf`, `ArrayList::new`, `() -> {}`; `local_variable` 4. |
| TC-L2 | unit | Method reference forms: `String::valueOf` (static), `this::m` (bound), `ArrayList::new` (constructor), `Map.Entry::getKey` (qualified). | 1 each. One grammar node, `method_reference`, covers all four — the Kotlin `Type::method` gap does not repeat. |
| TC-L3 | unit | Lambda nested in lambda: `xs.forEach(x -> ys.forEach(y -> use(x, y)))`. | `lambda` 2. |
| TC-L4 | unit | A lambda with a block body and one with an expression body. | 1 each. |
| TC-L5 | unit | An anonymous class `new Runnable() { public void run() {} }`. | `lambda` 0, `inheritance` 1 (TC-E11). |
| TC-L6 | unit | A lambda assigned to a field of a class: `class A { Runnable r = () -> {}; }`. | `lambda` 1 and `local_variable` 1 on `A` — Java has no property unit, so nothing is exempted. |
| TC-L7 | unit | `Foo.class` and a cast `(Runnable) x`. | `lambda` 0 — a class literal is not a method reference. |

## T8 — Coupling (FR-6, FR-7)

Fixtures: `coupling.java` and `coupling_no_star.java`, both with
`InternalPrefixes = ["com.acme"]`.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-C1 | unit | `Invoice` in `coupling.java`. | `internal_coupling` 2 (`Money`, and `rate` through the static import), `external_coupling` 2 (`Instant` and the star). |
| TC-C2 | unit | `Note` in `coupling.java`. | `internal_coupling` 1 (`Ledger`), `external_coupling` 1 (the star only). |
| TC-C3 | unit | `Plain` in `coupling.java`. | `internal_coupling` 0, `external_coupling` 1 — the star charges every unit. |
| TC-C4 | unit | `coupling_no_star.java`. | `Invoice` 2/1, `Note` 1/0, `Plain` 0/0. |
| TC-C5 | unit | `import static com.acme.shared.Rates.rate;`. | The binding is `rate`, the member, not `Rates`: a unit that calls `rate()` is charged, a unit that mentions only `Rates` is not. |
| TC-C6 | unit | `import static com.acme.shared.Rates.*;`. | A star module: charged to every unit, binding nothing. |
| TC-C7 | unit | Occurrence position. | Every coupling occurrence's `Line` is that of the `import_declaration`, above the unit's own `Line`. |
| TC-C8 | unit | `IsInternal` through the analyzer: `com.acme.shared.Money` with prefixes `["com.acme"]`, `["com.acme.shared"]`, `["com.acme.shared.Money"]` → internal; `["com.acmecorp"]`, `[""]`, none → external; `java.util.List` → external. | As stated; the pure-function table itself lives in `jvm` (TC-R1). |
| TC-C9 | unit | Two imports of the same path (`import a.B;` written twice). | One module; charged once to a unit that uses `B`. |
| TC-C10 | unit | Import used only in a type position (`Money amount;`), only in a call (`Money.of(1)`), only in an annotation (`@Inject`), only in a `new` (`new Money()`), only as a generic argument (`List<Money>`). | All count as a use — refs collect both `identifier` and `type_identifier`. |
| TC-C11 | unit | Import whose binding no unit mentions. | Charged to no unit; a sibling unit that does mention it is still charged. |
| TC-C12 | unit | Shadowing: `import a.Money;` and a local `int Money = 1;` in a unit that never uses the imported type. | Charged (by-name test). Document as the same caveat TypeScript and Kotlin carry. |
| TC-C13 | unit | `import a.b.C;` where the unit mentions `C` only inside a string literal or a comment. | 0 — string contents and comments are not identifier nodes. |
| TC-C14 | unit | A same-package class used with no import, and a fully-qualified `java.time.Instant.now()` with no import. | 0 for both — pinned as the documented blind spots. |
| TC-C15 | unit | No `InternalPrefixes` at all. | Every import is external; the analyzer never guesses from the file's own `package`. |

## T9 — Registration and end to end (FR-12)

In `cmd/check_java_test.go`, mirroring `cmd/check_kotlin_test.go`: a
`writeJavaFixture` that runs `cdd init --languages java` and lays sources out
under `src/main/java/com/acme/app/`, and a `javaID(t)` that reads the id from
the registry rather than spelling it out.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-I1 | unit | `internal/languages` `TestEveryLanguageHasADirectory`, `TestEveryDirectoryIsRegistered`, `TestSpecCompleteness`. | Green after the one-line registration; a new test asserts the Java entry now carries a non-nil `NewAnalyzer`. |
| TC-I2 | integration | Clean project: one class of 1 ICP, `cdd check --all`. | Exit 0; the unit is listed with `unit:` and `class Greeter icp=1 limit=10`; stderr empty — no "no analyzer for java yet". |
| TC-I3 | integration | Over-limit project: one class of 18 ICPs (six `if (a > 0 && b > 0)` lines) under greenfield limit 10. | Exit 1; stdout names `OrderService` with `violation:` and `icp=18 limit=10 over=8`, and the metric line reads `condition=12 code_branch=6`. |
| TC-I4 | integration | `--format json --explain` on the over-limit project plus an `Invoice.java` that imports `java.time.Instant`. | Valid JSON; `files[].language` equals the registry's Java id; the unit's `kind` is `class`; one occurrence sits on the `import` line for a coupling metric. |
| TC-I5 | integration | `--format xml` and `--format markdown`. | Both render without error and mention the unit name. |
| TC-I6 | integration | Mixed project: `--languages java,kotlin`, one file of each. | Both files reported, each with its own language; the exit code reflects the worst unit. |
| TC-I7 | integration | A `src/test/java/...` file with 50 ICPs. | Not in the report: the `DefaultExcludes` written by `init` still apply. |
| TC-I8 | integration | `cdd check src/main/java/com/acme/app/Ledger.java` (path narrowing). | Only that file reported. |
| TC-I9 | integration | A file with a syntax error in the project. | Exit code unchanged by it; the warning appears in the report; other files are still analyzed. |
| TC-I10 | integration | `internal_coupling.auto_detect: true` with the Java source layout. | `com.acme.*` imports classified internal without listing them in `packages`. |
| TC-I11 | integration | `timeout: 1ms` on a project with several files. | Exit 2 with a partial report — proves the Java analyzer honours the deadline through `parse`. |
| TC-I12 | lint | `make check-literals` and `make lint`. | Green: no `"java"`, `".java"` or metric-id literal outside `spec.go`; `0 issues`, in particular `funlen`/`gocyclo` on the counter switch and `lll` on the fixture-heavy tests. |
| TC-I13 | build | `make build`. | Binary builds with CGO; note the size delta against `main` in the PR description and in the README note (T10). |

## Cross-cutting invariants (every fixture)

Iterate `internal/analyze/java/testdata/*.java`; do not list files.

| ID | Level | Case | Expectation |
|---|---|---|---|
| TC-X1 | unit | `TestOccurrencesAccountForEveryCount`: for every unit of every fixture, the sum of `Occurrences[i].Count` per metric equals `Counts[metric]`. | Passes. |
| TC-X2 | unit | `TestOccurrencesAreWellFormed`: `Line ≥ 1`, `Col ≥ 1`, `(EndLine, EndCol) > (Line, Col)`, `Count ≥ 1`, `Metric` is a known id. | Passes. |
| TC-X3 | unit | `TestOccurrencesAreSorted`: occurrences are non-decreasing by `(Line, Col)`. | Passes. |
| TC-X4 | unit | `TestNonCouplingOccurrencesAreInsideTheUnit`: every occurrence whose metric is not a coupling metric sits within the unit's own line range. | Passes. |
| TC-X5 | unit | `TestCouplingOccurrencesSitOnImports`: every `internal_coupling`/`external_coupling` occurrence's `Line` is strictly above the unit's `Line` and matches an `import` line of the file. | Passes for both coupling fixtures and vacuously elsewhere. |
| TC-X6 | unit | `TestEveryUnitCarriesEveryMetric`: each unit's `Counts` has exactly `len(config.Metrics())` keys — disabled metrics are the pipeline's business. | Passes. |
| TC-X7 | unit | Determinism: analyze each fixture twice with two analyzers. | `reflect.DeepEqual` on the results. |
| TC-X8 | unit | `-race` is on in `make test`; a small parallel test builds one analyzer per goroutine and analyzes concurrently. | No race reported; the shared grammar is read-only. |

## Coverage targets

- `internal/analyze/java` ≥ 90 % (statements). Run `make cover` and paste the
  per-function table for the package in the PR description.
- `internal/analyze/internal/jvm` ≥ 90 %.
- `internal/analyze/kotlin` and `internal/analyze/typescript` coverage must not
  fall below their pre-T1 values.
