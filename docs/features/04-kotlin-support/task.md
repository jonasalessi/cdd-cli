# Feature 04 — Kotlin Support: CDD Analyzer on Tree-sitter

## Goal

Give Kotlin the analyzer its spec has been waiting for. After this feature,
`cdd check` in a Kotlin project parses every `.kt` file the configuration
matches, computes ICPs per top-level declaration, enforces `icp-limits`, and
reports through the four existing reporters — with the same pipeline, the same
reporters and the same `check` command TypeScript already uses.

Kotlin is the second analyzer, so it also proves the seam feature 03 left
implicit: the tree-sitter mechanics that are not about any one grammar move
into a shared package before Kotlin is written, and TypeScript is shown to be
unchanged by that move.

The measure of success is not "counts something": it is that a Kotlin
developer running `cdd check` on a real codebase reads the report and finds it
credible. Every rule below was chosen for cross-language comparability with the
shipped TypeScript rules first, fidelity to `docs/cdd.md` second, and
Kotlin-specific tuning only where the language genuinely differs.

[test-cases.md](test-cases.md) is the companion to this file: every
constraint below has a numbered test case there, and a task is done when its
cases are checked-in, passing tests.

## Current state (verified 2026-09-07)

- `internal/analyze/kotlin/spec.go` exists, is registered in
  `internal/languages/languages.go` with `NewAnalyzer: nil`, and shares
  `internal/analyze/internal/jvm` package-prefix detection with Java.
  `spec_test.go` and `testdata/project/` cover detection.
- `cdd check` on a Kotlin file stops with `no analyzer for kotlin yet`.
- The Kotlin spec's `code_branch` description says
  `if/when, loops, safe calls (?.), elvis (?:)`; the README metric table and
  `internal/config/templates/cdd.config.yaml.tmpl` line 30 repeat
  `?: kotlin` under `code_branch`. Feature 03 FR-10 ruled that TypeScript's
  `??` — the same operator — is a `condition`. These three places are wrong
  and are fixed here (FR-2).
- `Extensions` lists `.kt` and `.kts`. `.kts` is dropped here (FR-3).

## Scope

**In:**
- `internal/analyze/internal/treesitter` — the grammar-agnostic tree-sitter
  helpers extracted from `internal/analyze/typescript` (FR-1).
- Spec corrections: elvis classification and `.kts` removal (FR-2, FR-3).
- `internal/analyze/kotlin` — the analyzer: units, metric counters, import
  coupling (FR-4 … FR-12).
- Registration: `NewAnalyzer: kotlin.NewAnalyzer` in `languages.go` (FR-13).
- Hand-maintained docs: README language paragraph and metric table, config
  template comments, CONTRIBUTING note on the grammar's provenance.

**Out:**
- Java analyzer. Its `Descriptions` and idioms differ enough that bundling it
  would make both worse; the JVM counting seam is extracted when Java lands,
  not speculatively now.
- Kotlin scripts (`.kts`), Kotlin Multiplatform expect/actual pairing, KDoc.
- Cross-file symbol resolution. Same-package references that need no `import`
  are invisible to a per-file analyzer and stay a documented limitation.
- Any change to the metric vocabulary, default selection or default weights.
  `lambda` stays opt-in for every language.
- Changes to `internal/analyze` pipeline types, reporters or `cmd/check.go`.
  If one turns out to be necessary it is a separate, justified commit.

## Dependencies (pinned)

| Module | Version | Notes |
|---|---|---|
| `github.com/tree-sitter/go-tree-sitter` | `v0.24.0` | Already pinned by feature 03. Do not bump. |
| `github.com/tree-sitter-grammars/tree-sitter-kotlin` | `v1.1.0` | Import `bindings/go`; `Language()` returns the parse table. Verified 2026-09-07 to build against the binding above and to parse classes, delegation, `when`, safe calls, elvis, lambdas, destructuring, getters/setters and imports without error. |

Provenance to state in CONTRIBUTING: this is the first grammar outside the
`tree-sitter/` GitHub organisation. `tree-sitter-grammars/` is the community
collective that maintains the fork of `fwcd/tree-sitter-kotlin` (`v0.3.2`,
one release behind and less active). Accepted risk; the mitigation is that
every node kind and field the analyzer relies on is resolved by name once at
startup (`IdForNodeKind`, `FieldIdForName`) and pinned by
`TestGrammarResolvesEveryKind`, so a grammar bump that renames something fails
loudly in `make test` instead of silently counting zero.

Consequences: binary grows by the Kotlin parse table (roughly a third of the
TypeScript pair); CGO and the C-compiler prerequisite are unchanged.

## Design decisions (settled)

| Decision | Choice | Rejected |
|---|---|---|
| Unit granularity | Top-level declarations only, mirroring TypeScript FR-1. Nested classes, companion objects, members and local functions bill to the enclosing unit. | Per-function units (breaks comparability with TS limits); companion object as its own unit (an implementation detail of the enclosing class). |
| Unit `Kind` labels | Distinct: `class`, `interface`, `enum`, `object`, `typealias`, `function`, `property`. The grammar folds interface/enum/sealed/annotation/data into `class_declaration`, so the keyword token is read once per unit. | Everything class-shaped as `"class"` — the report would call an interface a class, which is the first thing a Kotlin reader distrusts. |
| Visibility | No filter. `private`/`internal` top-level declarations are units. | Mirroring TypeScript's "exported arrow consts only" rule. That rule is about module surface; Kotlin has no `export`, and a private top-level function still carries complexity someone maintains. |
| Elvis `?:` | `condition`, two clauses like TS `??`. | Keeping the spec's `code_branch` wording — same operator, different metric across languages. |
| `!!` | Not counted. | `code_branch` — it is an assertion, not a decision the reader follows; and it is not in the spec's description. |
| `when` | +1 per `when_entry` that has a condition; the `else` entry is 0. `1, 2 ->` is one entry, one point. | +1 per `condition:` field — a multi-value arm is one decision, and `switch_case` in TS is one per case. |
| Lambdas | Count uniformly: `lambda_literal`, `anonymous_function`, `callable_reference`. Scope functions (`let`, `apply`, `run`, `also`, `with`, …) are **not** exempted. | Name-based allowlist — unsound without type resolution (any receiver can declare its own `apply`), and `lambda` is already off in `config.DefaultSelection()`, so the tax only exists for teams that opt in and can weight it. |
| Coupling attribution | Per unit, by binding reference, like TS FR-6. A star import binds unknown names and charges every unit, like a TS side-effect import. An alias attributes by the alias. | Charging every import to every unit (a file with twelve imports and three classes would triple-count). |
| Same-package references | Documented blind spot. | Heuristic detection (a capitalised identifier with no import is "probably internal") — wrong often enough to discredit the number. |
| `inheritance` | +1 per `delegation_specifier`, `explicit_delegation` (`: I by d`) included. The charge is for `: I`, the `implements` analogue TS already counts, not for `by`. | Exempting `by` — would make `class A : I by d` cheaper than `class A : I`, which it is not for the reader of the type. |
| `local_variable` destructuring | One per declaration: `val (a, b) = p` is 1. | One per name — TS counts one per declarator (`const {a, b} = x` is 1), and the doc calls it a "method-level temporary variable", singular. |
| `local_variable` loop bindings | `for (i in xs)` is 1, on the binding. | 0 — TS charges `for…of` bindings (`countLoopBinding`) for the same doc reason. |
| Plain constructor params | 0. `class P(val a: Int, b: Int)` charges `a` and not `b`. | Counting both — parameters are not counted in TS either. |
| Shared code location | `internal/analyze/internal/treesitter`, extracted in a preparatory `refactor:` commit that leaves TS byte-identical. | Extracting inline with the Kotlin work (a regression in TS would bisect to a 40-file commit); naming it `tsutil` (reads as "TypeScript util"). |

## Functional requirements

| # | Rule | Where the rule lives |
|---|---|---|
| FR-1 | The grammar-agnostic tree-sitter helpers move out of `internal/analyze/typescript` into `internal/analyze/internal/treesitter` with no behaviour change: the parse budget and deadline-to-timeout conversion, the cursor `walk`, `namedChildren`/`namedChildrenInField`/`firstNamedChild`/`text`, `srcSpan`/`spanOf`/`position`, `firstErrorNode` and the syntax-warning text, and the source-order occurrence sort. The `kind` table, `grammar` struct, every metric rule and every unit rule stay in the language package. | `internal/analyze/internal/treesitter`, `internal/analyze/typescript` |
| FR-2 | Elvis `?:` is a `condition`. The Kotlin spec's `code_branch` description becomes `if/when, loops, safe calls (?.)` and its `condition` description becomes `&&, \|\| and ?: clauses`. The README metric table row and the config template comment stop listing `?:` under `code_branch`. Golden files that render the description are regenerated. | `internal/analyze/kotlin/spec.go`, `README.md`, `internal/config/templates/cdd.config.yaml.tmpl` |
| FR-3 | `Extensions` is `[".kt"]`. `cdd check` never opens a `.kts` file. `jvm.Prefixes` receives the same slice, so package detection stops reading build scripts too (they declare no package; the detection test's expectation is unchanged). | `internal/analyze/kotlin/spec.go` |
| FR-4 | A Kotlin **unit** is each direct named child of `source_file` that is a `class_declaration` (kind by keyword token: `interface` → `interface`; `class` with `class_modifier` `enum` → `enum`; every other `class` → `class`, `sealed`/`data`/`annotation`/`abstract`/`open`/`inner`/`value` included), an `object_declaration` (`object`), a `function_declaration` (`function`, extension functions included), a `type_alias` (`typealias`), or a `property_declaration` (`property`) whose initializer is a `lambda_literal` or `anonymous_function`, or which carries a `getter` or `setter` with a `function_body`, or which has a `property_delegate`. A top-level property holding any other value is not a unit. Nothing nested inside a unit is a unit. `Name` is the `name:` field (`type:` for `type_alias`; the `variable_declaration` identifier for a property; for an extension function, the bare function name without the receiver). `Line`/`Col` point at the declaration's first token, `modifiers` excluded. | `internal/analyze/kotlin/units.go` |
| FR-5 | ICPs are counted per unit over the unit's whole subtree using the metric→node mapping below. The analyzer counts every metric; the pipeline drops the disabled ones. Every charge records an `analyze.Occurrence`; `Counts` equals the sum of `Occurrences` per metric. | `internal/analyze/kotlin/metrics.go` |
| FR-6 | A file whose root `HasError()` yields no units and one warning `syntax error at L:C` naming the first `ERROR` or `MISSING` node, falling back to `1:1` when the root is flagged but no such node exists (the Kotlin grammar does this on some unusual one-line layouts). Same shape as TS FR-5. | `internal/analyze/kotlin/analyzer.go` |
| FR-7 | Import classification: every `import` node at the top of `source_file` names one qualified path. It is **internal** when the path equals one of `Options.InternalPrefixes` or starts with prefix + `.`; otherwise **external**. Prefixes come from `internal_coupling.packages` and `jvm.Prefixes` as today. There is no "relative import" notion. | `internal/analyze/kotlin/imports.go` |
| FR-8 | Import attribution: an import binds one local name — the last `identifier` of its `qualified_identifier`, or the alias after `as` when present — and is charged to every unit whose subtree mentions that identifier, once per import however many mentions. A star import (anonymous `*` child) binds no name and is charged to every unit of the file. Two imports of the same path are one module. The occurrence points at the `import` node. | `internal/analyze/kotlin/imports.go` |
| FR-9 | `condition` counts Boolean **clauses**, not operators: the leaf operands of a chain of `binary_expression` nodes whose `operator` is `&&`, `\|\|` or `?:`, flattened through `parenthesized_expression` and prefix `!` exactly as TS `clauses()` does. `a && b` = 2, `a && b \|\| c` = 3, `x ?: y` = 2, `if (x > 1)` = 0. Kotlin has no `&&=`/`\|\|=`, so there is no logical-assignment rule. | `internal/analyze/kotlin/metrics.go` |
| FR-10 | `if / else if / else` is 3, not 4: an `if_expression` is +1; its `else` branch is +1 only when the node after the anonymous `else` token is not itself an `if_expression`. | `internal/analyze/kotlin/metrics.go` |
| FR-11 | The unit's own body is never one of its lambdas: a `property` unit whose initializer is the lambda or anonymous function that made it a unit does not charge that node as `lambda`, mirroring TS `skipLambda`. A `property` unit does not charge its own declaration as `local_variable`, mirroring `skipDeclarator`. | `internal/analyze/kotlin/metrics.go` |
| FR-12 | Every node kind, field and anonymous token the analyzer uses is resolved by name once, when the shared grammar is built, and a test asserts each resolves to a non-zero id. `IdForNodeKind(name, false)` is used for anonymous tokens (`?.`, `*`, `else`, `val`, `var`, `interface`, `enum`). | `internal/analyze/kotlin/parser.go` |
| FR-13 | `languages.go` gains `NewAnalyzer: kotlin.NewAnalyzer`; the analyzer implements `io.Closer` and the pipeline closes it. No other file outside `internal/analyze/kotlin` and the docs changes. `make check-literals` stays green: string literals for `"kotlin"` and metric ids appear only in `spec.go`. | `internal/languages/languages.go` |

## Metric → Tree-sitter node mapping

Node kinds and fields are the exact names from
`tree-sitter-grammars/tree-sitter-kotlin v1.1.0`, verified against the parse
tree on 2026-09-07. Quoted strings are anonymous tokens, reachable through
`IdForNodeKind(name, false)` or by scanning `Child(i)` for `!IsNamed()`.

| MetricID | Counted nodes / rule |
|---|---|
| `code_branch` | `if_expression` +1, plus +1 for its `else` branch unless that branch is an `if_expression` (FR-10). `when_entry` +1 when it has at least one `condition:` field; a `when_entry` whose first child is the `"else"` token is 0. `for_statement`, `while_statement`, `do_while_statement` +1 each. `navigation_expression` whose anonymous child is `"?."` +1 (a plain `"."` is 0; `a?.b?.c` is 2). Nothing for `"!!"`, `try`, `return`, `break`, `continue`, `throw`. |
| `condition` | Leaf clauses of `binary_expression` chains with `operator:` `"&&"`, `"\|\|"`, `"?:"` (FR-9). Each clause is one occurrence on the operand node. Nested chains are marked consumed so an inner `&&` is not counted again. |
| `exception_handling` | `try_expression` +1 on its first `block` child (the guarded body, not the whole expression); each `catch_block` +1; `finally_block` +1. `try { } catch { } finally { }` = 3. |
| `internal_coupling` | +1 per import classified internal (FR-7) that the unit uses (FR-8). Occurrence on the `import` node. |
| `external_coupling` | +1 per import classified external, same attribution. |
| `inheritance` | +1 per `delegation_specifier` anywhere in the unit's subtree — under a class, an object, a companion, an enum, or an anonymous `object : Iface { }` expression — whatever its shape: `constructor_invocation` (`: Base()`), bare `user_type` (`: Iface`), `explicit_delegation` (`: Iface by d`). Occurrence on the specifier's `user_type`, so `: A, B` is two occurrences a reader can tell apart. A nested declaration's specifiers bill to the enclosing unit like everything else nested. |
| `local_variable` | `property_declaration` inside the unit (locals in a `block`, members in a `class_body`) +1 each, `multi_variable_declaration` included (one per declaration, not per name). `class_parameter` +1 only when its first anonymous child is `"val"` or `"var"`. `for_statement` +1 on its `variable_declaration` or `multi_variable_declaration` binding. **Not** counted: `enum_entry`; a `property_declaration` in an `interface` unit's `class_body` that has no initializer, no `property_delegate` and no accessor `function_body` (a shape, not a variable, like a TS interface property signature); function parameters; lambda parameters; the `property` unit's own declaration (FR-11). |
| `lambda` | `lambda_literal` +1 (trailing form under `annotated_lambda` included), `anonymous_function` +1, `callable_reference` +1 (`::name`, `Type::name` when the grammar produces the node). The `property` unit's own body is excluded (FR-11). Known gap: `String::trim` parses as `navigation_expression` in v1.1.0 and is not counted; document, do not heuristically match `::` in text. |

Occurrence ranges follow TS: `srcSpan` from `StartPosition()`/`EndPosition()`,
1-based, end exclusive.

## Worked fixtures (acceptance values)

Encode each as a fixture in `internal/analyze/kotlin/testdata/` with a test
asserting the exact per-unit counts. Totals below assume every metric enabled
at weight 1.0.

```kotlin
// cdd_examples.kt — docs/cdd.md section 2, verbatim rules
fun check(a: Int, b: Int, c: Int, d: Int): Boolean {
    if (a > b && c < d) {     // code_branch 1, condition 2
        return true
    }
    return false
}                             // total 3

fun guarded() {
    try { a() } catch (e: E) { b() } finally { c() }   // exception_handling 3
}                             // total 3

fun ifElse(x: Int) = if (x > 0) 1 else 2               // code_branch 2 (if-else = 2)
```

```kotlin
// branches.kt
fun describe(x: Int): String = when (x) {
    1 -> "one"                // code_branch 1
    2, 3 -> "few"             // code_branch 1 (one entry)
    else -> "many"            // 0
}                             // total 2

fun chain(a: Boolean, b: Boolean, c: Boolean): Int =
    if (a) 1 else if (b) 2 else if (c) 3 else 4        // code_branch 4 (3 ifs + final else)

fun safe(s: String?): Int = s?.trim()?.length ?: 0
// code_branch 2 (two ?.), condition 2 (clauses of ?:) — total 4

fun loops(xs: List<Int>, m: Map<String, Int>) {
    for (x in xs) { }         // code_branch 1, local_variable 1
    for ((k, v) in m) { }     // code_branch 1, local_variable 1
    while (a) { }             // code_branch 1
    do { } while (b)          // code_branch 1
    x!!.y                     // 0
}                             // code_branch 4, local_variable 2
```

```kotlin
// inheritance.kt
class Ledger(private val repo: Repo, clock: Clock) : Base(), Auditable, Printer by ConsolePrinter()
// inheritance 3, local_variable 1 (repo; clock is a plain parameter)

interface Auditable : Named, Timestamped                 // inheritance 2
object Registry : Auditable                              // inheritance 1
```

```kotlin
// lambdas.kt
fun total(xs: List<Int>): Int {
    val doubled = xs.map { it * 2 }        // lambda 1, local_variable 1
    val shown = xs.map(::render)           // lambda 1, local_variable 1
    val anon = fun(a: Int): Int = a        // lambda 1, local_variable 1
    return xs.let { it.sum() }             // lambda 1
}                                          // lambda 4, local_variable 3

val onEvent: (Int) -> Unit = { println(it) }   // unit "onEvent" kind property; lambda 0 (FR-11), local_variable 0
val plain = 3                                  // not a unit
```

```kotlin
// units.kt — expected units in order, with kinds
package com.acme.app

class A                        // class
interface B                    // interface
enum class C { X, Y }          // enum (enum_entry: local_variable 0)
sealed class D                 // class
data class E(val v: Int)       // class (local_variable 1)
annotation class F             // class
object G                       // object
fun h() {}                     // function
fun String.shout() = uppercase()   // function, name "shout"
typealias I = (Int) -> Unit    // typealias
val j = { 1 }                  // property
val k: Int get() = 2           // property
val l by lazy { 3 }            // property (lambda inside `lazy { }` is +1: it is not the unit's own body)
private fun m() {}             // function (no visibility filter)
val n = 4                      // not a unit
class O {                      // class — one unit; everything below bills to it
    inner class P
    companion object { fun q() {} }
    object R
    fun s() { fun local() {} }
}
```

```kotlin
// coupling.kt — with InternalPrefixes = ["com.acme"]
package com.acme.billing

import com.acme.shared.Money
import com.acme.shared.Ledger as L
import java.time.Instant
import kotlinx.coroutines.*

class Invoice(val amount: Money) {    // internal 1 (Money), external 2 (Instant, star), local_variable 1
    fun at(): Instant = Instant.now()
}
class Note { val l = L() }            // internal 1 (alias L → Ledger), external 1 (star), local_variable 1
class Plain                           // internal 0, external 1 (star charges every unit)
```

```kotlin
// broken.kt — FR-6
class Oops {
    fun f( {
}
// expected: no units, one warning "syntax error at L:C"
```

## Implementation guidance

- Follow feature 03's binding discipline verbatim: no finalizers, `Close()`
  every `Parser`/`Tree`/`TreeCursor`, never let a `*Node` outlive its tree,
  one parser and cursor per worker, kinds resolved to `uint16` once.
- `kind` for Kotlin is a fresh enum in `internal/analyze/kotlin/parser.go`.
  Copy the shape of TS's `kindNames`/`newGrammar`/`kindOf`, not the values.
  Kotlin has one grammar, so there is no `grammars.forPath`; `.kt` is the
  only extension and any other is an error like TS does for unknown ones.
- Reading the unit kind: the `class_declaration` node's anonymous children
  include exactly one of `"class"` or `"interface"`; the `modifiers` child,
  when present, holds `class_modifier` nodes whose text is `enum`, `sealed`,
  `data`, `annotation`, … . Only `interface` and `enum` change the label.
- `?.` detection: `navigation_expression` has no `operator` field. Scan its
  direct children for the anonymous token whose `KindId()` equals the id
  resolved for `"?."`, exactly as TS `countOptionalCall` does for its
  anonymous `?.`.
- `else` detection on `if_expression`: the branch after the anonymous
  `"else"` token is the next sibling of that token. Reading it by index is
  fine; the grammar has no `alternative` field (verified: `FieldIdForName`
  returns 0 for `consequence`, `alternative`, `body`, `receiver`,
  `delegate`). Fields that do exist: `condition`, `name`, `type`,
  `operator`, `left`, `right`, `argument`.
- `when_entry` "has a condition": `ChildByFieldId(fields.condition) != nil`.
- Identifier references for FR-8: record the text of every `identifier`
  node the unit's subtree contains. Types are `user_type > identifier` in
  this grammar, so one node kind covers values and types (there is no
  `type_identifier`). Same by-name test TS uses, same documented shadowing
  caveat.
- Reference smoke test before opening the PR: run `cdd check` over a few
  hundred files of a public, idiomatically formatted Kotlin project (e.g. a
  ktlint-formatted library) and confirm the syntax-warning rate is zero or
  attributable to genuinely unusual layouts. The grammar flags
  `interface I { val x: Int\n fun g() }` (member on the brace line, bodiless
  function before a same-line brace) while parsing every conventional layout
  of the same interface cleanly.
- Keep `funlen` (80 lines / 60 statements) and `gocyclo` (20) in mind when
  writing the counter's `switch`; TS split it into `countControlFlow` and
  `countDeclaration` for that reason.

## Deliverables

```
internal/analyze/internal/treesitter/
    budget.go       parse budget / deadline → timeout micros
    walk.go         walk, namedChildren, namedChildrenInField, firstNamedChild, text
    span.go         srcSpan, spanOf, position
    errors.go       firstErrorNode, syntax-warning text
    occurrences.go  source-order stable sort
    *_test.go       moved with the code
internal/analyze/typescript/
    (imports the package above; behaviour byte-identical; goldens untouched)
internal/analyze/kotlin/
    spec.go         FR-2 descriptions, FR-3 extensions
    parser.go       kind enum, grammar resolution, shared grammar (FR-12)
    analyzer.go     NewAnalyzer, Analyze, Close, parse (FR-6)
    units.go        unit extraction (FR-4)
    metrics.go      counters (mapping table, FR-5, FR-9 … FR-11)
    imports.go      import classification and attribution (FR-7, FR-8)
    *_test.go, testdata/*.kt   the worked fixtures above, one file per metric
internal/languages/languages.go    FR-13, one line
docs/features/04-kotlin-support/test-cases.md   case ids cited from the tests (`// TC-…`)
README.md                          language paragraph, metric table row, binary note
internal/config/templates/cdd.config.yaml.tmpl   line 30 comment (FR-2)
CONTRIBUTING.md                    grammar provenance note
cmd/testdata/golden/*, internal/config/testdata/golden/*   regenerated if a description renders
```

## Tasks

Commit per task; the task title is the commit message body's first line.
Each task's acceptance criterion is shorthand for the cases listed under the
same task heading in [test-cases.md](test-cases.md); those cases are the
contract.

- **T1 — `refactor: extract tree-sitter helpers shared by analyzers`** (FR-1).
  Move the listed helpers into `internal/analyze/internal/treesitter`; TS
  imports them. No TS test changes other than import paths and receiver
  names. *Accept:* `make test` green; `git diff --stat` touches only
  `internal/analyze/typescript/*.go` and the new package; a TS fixture run
  before and after produces identical JSON reports.

- **T2 — `fix: classify Kotlin elvis as condition and drop .kts`** (FR-2,
  FR-3). Spec strings, extensions, README row, template comment; regenerate
  goldens. *Accept:* CI dogfood gate green; `kotlin` spec tests pass with
  the unchanged detection expectation.

- **T3 — `feat: parse Kotlin with tree-sitter`** (FR-6, FR-12). Dependency,
  `parser.go`, `analyzer.go` returning zero units, grammar-resolution test,
  `broken.kt` warning test. *Accept:* `TestGrammarResolvesEveryKind` passes;
  a broken file yields the warning and no error.

- **T4 — `feat: extract Kotlin units`** (FR-4). *Accept:* `units.kt` yields
  exactly the listed units, kinds, names and positions, in order.

- **T5 — `feat: count Kotlin branches and conditions`** (`code_branch`,
  `condition`, FR-9, FR-10). *Accept:* `cdd_examples.kt` and `branches.kt`
  totals reproduce exactly.

- **T6 — `feat: count Kotlin exceptions, inheritance and locals`**
  (`exception_handling`, `inheritance`, `local_variable`). *Accept:*
  `inheritance.kt` and the loop/local lines of `branches.kt` reproduce.

- **T7 — `feat: count Kotlin lambdas`** (`lambda`, FR-11). *Accept:*
  `lambdas.kt` reproduces, including the zero on the `property` unit's own
  body.

- **T8 — `feat: attribute Kotlin imports to the units that use them`**
  (FR-7, FR-8). *Accept:* `coupling.kt` reproduces with and without the
  alias and star lines.

- **T9 — `feat: register the Kotlin analyzer`** (FR-13). One line in
  `languages.go`; an e2e test in `cmd/check_test.go` following the TS
  fixture pattern: over-limit Kotlin unit ⇒ exit 1 naming it; clean project
  ⇒ exit 0. *Accept:* `cdd check` on a Kotlin fixture project works
  end-to-end in all four formats.

- **T10 — `docs: describe Kotlin support`**. README "Language support"
  paragraph (Kotlin joins TypeScript; Go and Java still `init`-only), binary
  size note, CONTRIBUTING provenance note, and a "Known limitations"
  subsection: same-package references, `Type::method` references, scope
  functions counted as lambdas. *Accept:* README and CONTRIBUTING reviewed
  against the shipped behaviour.

## Definition of done

- [ ] `make build`
- [ ] `make test` (race detector on)
- [ ] `make lint` (including `check-literals`)
- [ ] `make fmt` leaves no diff
- [ ] Coverage ≥ 90 % for `internal/analyze/kotlin` and
      `internal/analyze/internal/treesitter`; `internal/analyze/typescript`
      coverage does not drop
- [ ] Every worked fixture above is a checked-in test with the stated totals
- [ ] Every case in [test-cases.md](test-cases.md) is a checked-in test,
      citing its id, and passes; cross-cutting invariants TC-X1 … TC-X6 run
      over every fixture under `testdata/`
- [ ] The T1 refactor commit produces byte-identical TS reports on the
      existing fixtures
- [ ] CI dogfood gate green
- [ ] Nothing outside `internal/analyze/kotlin`, the new shared package,
      `languages.go` and the docs listed above changed — verified with
      `git diff --stat main`

## Suggested order

T1 → T2 → T3 → T4 → T5 → T6 → T7 → T8 → T9 → T10. T2 is independent of T1
and may land first. T5–T8 each depend on T4 and are independent of each
other; T9 depends on all of them.
