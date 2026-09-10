# Feature 05 — Java Support: CDD Analyzer on Tree-sitter

## Goal

Give Java the analyzer its spec has been waiting for. After this feature,
`cdd check` in a Java project parses every `.java` file the configuration
matches, computes ICPs per top-level type, enforces `icp-limits`, and reports
through the four existing reporters — with the same pipeline, the same
reporters and the same `check` command TypeScript and Kotlin already use.

Java is the third analyzer and the second on the JVM, so it also settles the
seam feature 04 deliberately deferred: the import classification and
attribution that is about the JVM rather than about Kotlin moves into
`internal/analyze/internal/jvm` before Java uses it, and Kotlin is shown to be
unchanged by that move.

The measure of success is not "counts something": it is that a Java developer
running `cdd check` on a real codebase reads the report and finds it credible.
Every rule below was chosen for cross-language comparability with the shipped
TypeScript and Kotlin rules first, fidelity to `docs/cdd.md` second, and
Java-specific tuning only where the language genuinely differs.

[test-cases.md](test-cases.md) is the companion to this file: every constraint
below has a numbered test case there, and a task is done when its cases are
checked-in, passing tests.

## Current state (verified 2026-09-10)

- `internal/analyze/java/spec.go` exists, is registered in
  `internal/languages/languages.go` with `NewAnalyzer: nil`, and shares
  `internal/analyze/internal/jvm` package-prefix detection with Kotlin.
  `spec_test.go` and `testdata/project/` cover detection.
- `cdd check` on a Java file stops with `no analyzer for java yet`.
- `Spec()` hands out the package-level `extensions` slice uncloned, so a
  caller's mutation leaks into every later call. The Kotlin spec was fixed for
  the same reason in feature 04; Java is fixed here (FR-2).
- The Java spec's `Descriptions` hold only the two coupling rows and are
  correct as written; nothing about the metric vocabulary changes here, so no
  golden file moves.
- Kotlin's `internal/analyze/kotlin/imports.go` holds `isInternal`, the
  `module` type and the per-unit attribution loop. All three are JVM rules
  rather than Kotlin rules and are extracted first (FR-1).

## Scope

**In:**
- `internal/analyze/internal/jvm/imports.go` — the import classification,
  merging and attribution extracted from `internal/analyze/kotlin` with no
  behaviour change (FR-1).
- Spec correction: `Spec()` returns cloned slices and a copied map (FR-2).
- `internal/analyze/java` — the analyzer: units, metric counters, import
  coupling (FR-3 … FR-11).
- Registration: `NewAnalyzer: java.NewAnalyzer` in `languages.go` (FR-12).
- Hand-maintained docs: README language paragraph, binary-size note and known
  limitations, CONTRIBUTING note on the grammar's provenance.

**Out:**
- Per-method units for Java. A unit is a top-level type; a method of a type
  bills to it, exactly as TypeScript and Kotlin bill their members. The one
  exception is a Java 25 compact source file, whose top-level method has no
  enclosing type to bill to.
- Cross-file symbol resolution. Same-package references that need no `import`,
  and fully-qualified references written inline (`java.time.Instant.now()`),
  are invisible to a per-file analyzer and stay documented limitations.
- `module-info.java` semantics. `requires`, `exports` and `opens` are not
  coupling; the file parses and yields no units.
- Any change to the metric vocabulary, default selection or default weights.
  `lambda` stays opt-in for every language.
- Changes to `internal/analyze` pipeline types, reporters or `cmd/check.go`.
  If one turns out to be necessary it is a separate, justified commit.

## Dependencies (pinned)

| Module | Version | Notes |
|---|---|---|
| `github.com/tree-sitter/go-tree-sitter` | `v0.24.0` | Already pinned by feature 03. Do not bump. |
| `github.com/tree-sitter/tree-sitter-java` | `v0.23.5` | Import `bindings/go`; `tree_sitter_java.Language()` returns the parse table. Its `go.mod` requires `go-tree-sitter v0.24.0`, exactly the pin above, so no binding bump. Verified 2026-09-10 against `$GOMODCACHE`, where the module is already present, so `go get` is offline-safe. |

Provenance to state in CONTRIBUTING: unlike the Kotlin grammar, this one comes
from the official `tree-sitter/` GitHub organisation, the same source as the
TypeScript pair, so the paragraph feature 04 added about community forks needs
one sentence saying Java is not one of them. The mitigation is unchanged and
still applies: every node kind, field and anonymous token the analyzer relies
on is resolved by name once at startup (`IdForNodeKind`, `FieldIdForName`) and
pinned by `TestGrammarResolvesEveryKind`, so a grammar bump that renames
something fails loudly in `make test` instead of silently counting zero.

Consequences: the binary grows by the Java parse table; measure the delta
against `main` and record it in the README binary-size note (T10). CGO and the
C-compiler prerequisite are unchanged.

## Design decisions (settled)

| Decision | Choice | Rejected |
|---|---|---|
| Unit granularity | Top-level type declarations, mirroring TypeScript FR-1 and Kotlin FR-4. Nested types, methods, constructors and initializers bill to the enclosing type. A compact source file's top-level method is a unit because there is no type to bill it to. | Per-method units. Java methods are small by convention and the limits in `icp-limits` are calibrated per type across languages; per-method units would make every Java project pass and break comparability. |
| Unit `Kind` labels | Distinct: `class`, `interface`, `enum`, `record`, `annotation`, `method`. The grammar already gives each its own node kind, so the label is read from the node, not from a keyword scan. | Everything type-shaped as `"class"` — the report would call a record a class, which is the first thing a Java reader distrusts. |
| Visibility | No filter. Package-private and `final` top-level types are units. | Mirroring TypeScript's "exported only" rule. That rule is about module surface; a package-private class still carries complexity someone maintains. |
| Shared code location | `internal/analyze/internal/jvm`, extended in a preparatory `refactor:` commit that leaves Kotlin byte-identical. | Copying the import logic into `internal/analyze/java` (two places to fix one classification bug); putting it in `treesitter` (that package is grammar- and metric-agnostic and must not import `config`). |
| Switch arms | +1 per `switch_block_statement_group` (old style) and per `switch_rule` (arrow style) that carries at least one non-`default` label. `case 1: case 2: stmt` is one group and one point, matching Kotlin's `2, 3 ->` and TypeScript's one-point-per-`switch_case`. | +1 per `switch_label` — a multi-label arm is one decision the reader follows, not two. |
| `default` arm | 0. | +1 — the same reasoning that gives Kotlin's `else ->` and a plain `else` block zero when they carry no test of their own. Note the asymmetry with `if/else`, where the final `else` is +1: an `if` chain's `else` is the second half of a binary decision, a `default` is the fallthrough of an already-counted set. |
| Ternary | `ternary_expression` +1, like a one-line `if`. | 0 — it is the same decision written shorter, and TypeScript already counts its conditional expression. |
| `local_variable` per declarator | `int a, b;` is 2, on each `variable_declarator`. | 1 per declaration — TypeScript counts per declarator, and the doc calls it a "method-level temporary variable", singular per name. Kotlin has no multi-declarator form, so nothing diverges. |
| Record components | Counted: `record Money(int amount, String currency)` is 2. | 0 — a record component is a field with a shorter spelling, exactly the case Kotlin already charges for `val` constructor parameters. Plain method and constructor parameters stay 0 in all three languages. |
| Enhanced-for binding | `for (Integer x : xs)` is +1 on `x`. | 0 — TypeScript charges `for…of` bindings and Kotlin charges `for (x in xs)` for the same doc reason. |
| Resource declarations | A `resource` that declares a name (`try (var in = open(p))`) is +1; a resource that only names an existing variable (`try (existing)`) is 0. | Counting both — the second declares nothing. |
| Pattern variables | 0: `o instanceof String s`, record deconstruction patterns and `case Circle c ->` bindings. | +1 — they are the reading of a value the branch already charged for, and counting them would tax pattern matching over an `if`-plus-cast that costs less. Documented, revisit if Java code stops reading that way. |
| Anonymous classes | `inheritance` +1, on the `type` field of the `object_creation_expression`, mirroring Kotlin's `object : Iface {}`. Not a `lambda`. | `lambda` +1 — an anonymous class implements a type as much as a named one does, and a reader must follow the supertype to know what it is. |
| `permits` | 0. | +1 per permitted subtype — `permits` narrows who may extend, it does not make the sealed type depend on anything; the subtype's own `implements` is where the edge is already charged. |
| Bitwise `&`, `\|`, `^` | 0 for `condition`. | Counting them — they are arithmetic on bits, and counting them would make `flags & MASK` a Boolean clause. Documented as a limitation: the non-short-circuit Boolean forms `a & b` on `boolean` operands are indistinguishable without type resolution and are therefore not counted. |
| Method references | `method_reference` +1 for `lambda`, covering `String::valueOf`, `ArrayList::new` and `this::m`. | 0 — it is a function value passed to somebody, the same thing the lambda form denotes. Java's grammar produces one node for all of them, so Kotlin's `Type::method` gap does not repeat here. |
| Same-package and fully-qualified references | Documented blind spots. A class used without an `import` — same package, or written out as `java.time.Instant.now()` — is not charged. | Heuristic detection (a dotted capitalised identifier is "probably a type") — wrong often enough to discredit the number, and the fully-qualified form is rare in formatted code. |

## Functional requirements

| # | Rule | Where the rule lives |
|---|---|---|
| FR-1 | The JVM import rules move out of `internal/analyze/kotlin/imports.go` into `internal/analyze/internal/jvm/imports.go` with no behaviour change: `Module{Path, Bindings, Internal, Star, At}`, `IsInternal(path, prefixes)`, `Imports` with `NewImports(prefixes)`, `Bind(path, binding, at)`, `Star(path, at)` and `Modules()`, plus `Module.UsedBy(refs)` and `Module.Metric()`. Kotlin keeps only the grammar walk that reads a path, an alias and a star from an `import` node, and a `countCoupling` that loops over `Modules()` charging `m.Metric()` at `m.At` when `m.UsedBy(c.refs)`. The `isInternal` tests move to `jvm/imports_test.go`. Kotlin's JSON reports over its fixtures are identical before and after. | `internal/analyze/internal/jvm`, `internal/analyze/kotlin` |
| FR-2 | `Spec()` returns cloned slices and a copied map, as the TypeScript and Kotlin specs do; today `Extensions: extensions` hands out the package variable. Descriptions, `ID`, `DisplayName`, `DefaultExcludes`, `PackageExample` and `LimitExamples` are unchanged, so no golden file moves. | `internal/analyze/java/spec.go` |
| FR-3 | A Java **unit** is each direct named child of `program` that is a `class_declaration` (kind `class`), an `interface_declaration` (`interface`), an `enum_declaration` (`enum`), a `record_declaration` (`record`), an `annotation_type_declaration` (`annotation`) or a `method_declaration` (`method`, the compact source file case). `Name` is the `name` field. `Line`/`Col` point at the declaration's first child that is not `modifiers` — the `class` / `interface` / `enum` / `record` / `@interface` keyword, or a method's return type. Nothing nested inside a unit is a unit. `package_declaration`, `import_declaration`, `module_declaration` and top-level statements are not units, and their complexity is invisible, like Kotlin's `val n = 4`. | `internal/analyze/java/units.go` |
| FR-4 | ICPs are counted per unit over the unit's whole subtree using the metric→node mapping below. The analyzer counts every metric; the pipeline drops the disabled ones. Every charge records an `analyze.Occurrence`; `Counts` equals the sum of `Occurrences` per metric, and every unit carries a key for every `config.Metrics()` entry. | `internal/analyze/java/metrics.go` |
| FR-5 | A file whose root `HasError()` yields no units and one warning `syntax error at L:C` naming the first `ERROR` or `MISSING` node, falling back to `1:1` when the root is flagged but no such node exists. Only `.java` is accepted, case-insensitively; any other extension returns an error naming the path and the extension, and no units. Same shape as Kotlin FR-6. | `internal/analyze/java/analyzer.go` |
| FR-6 | Import classification: every `import_declaration` at the top of `program` names one qualified path — the text of its `scoped_identifier` or `identifier` child, stopping before `.*` on a star import. It is **internal** when the path equals one of `Options.InternalPrefixes` or starts with prefix + `.`; otherwise **external**. Prefixes come from `internal_coupling.packages` and `jvm.Prefixes` as today. Java has no relative import and no import alias. | `internal/analyze/java/imports.go` |
| FR-7 | Import attribution: a named import binds one local name — the last segment of the path, which is the `name` field of the `scoped_identifier`. For `import static a.b.C.m` that name is `m`, so a static member import is attributed by the member. An import is charged to every unit whose subtree mentions that name, once per module however many mentions. A star import (`import a.b.*;`, `import static a.b.C.*;`, recognised by an `asterisk` child) binds no name and is charged to every unit of the file. Two imports of the same path are one module. The occurrence points at the `import_declaration` node. | `internal/analyze/java/imports.go` |
| FR-8 | `condition` counts Boolean **clauses**, not operators: the leaf operands of a chain of `binary_expression` nodes whose `operator` is `&&` or `\|\|`, flattened through `parenthesized_expression` and unary `!` (the `operand` field) exactly as Kotlin `clauses()` does. `a && b` = 2, `a && b \|\| c` = 3, `!(a \|\| b) && x` = 3, `x > 1` = 0, `a & b \| 3` = 0. Nested chains are marked consumed so an inner `&&` is not counted twice. Java has no `??`, no `?:` elvis and no `&&=`. | `internal/analyze/java/metrics.go` |
| FR-9 | `if / else if / else` is 3, not 4: an `if_statement` is +1; the node in its `alternative` field is +1 only when it is not itself an `if_statement`. The `else` occurrence spans from the anonymous `else` token to the end of the alternative, so a reader sees the branch and not the whole statement. | `internal/analyze/java/metrics.go` |
| FR-10 | Switch arms: `switch_block_statement_group` and `switch_rule` are +1 each when at least one of their `switch_label` children is not the anonymous `default` token. `case 1: case 2: stmt` is one group and one point; `case 2, 3 ->` is one rule and one point; `default:` and `default ->` are 0. `switch_expression` covers both statement and expression forms, so there is one rule for both. | `internal/analyze/java/metrics.go` |
| FR-11 | Every node kind, field and anonymous token the analyzer uses is resolved by name once, when the shared grammar is built, and a test asserts each resolves to a non-zero id. `IdForNodeKind(name, false)` is used for the anonymous tokens (`else`, `default`). Two analyzers share one immutable grammar and never share a parser. | `internal/analyze/java/parser.go` |
| FR-12 | `languages.go` gains `NewAnalyzer: java.NewAnalyzer`; the analyzer implements `io.Closer` and the pipeline closes it. No file outside `internal/analyze/java`, `internal/analyze/internal/jvm`, `internal/analyze/kotlin`, `languages.go`, `cmd/check_java_test.go` and the docs listed below changes. `make check-literals` stays green: the string `"java"`, the extension `".java"` and the description strings appear only in `spec.go`. | `internal/languages/languages.go` |

## Metric → Tree-sitter node mapping

Node kinds and fields are the exact names from
`github.com/tree-sitter/tree-sitter-java v0.23.5`, verified against
`src/node-types.json` and the parse tree on 2026-09-10. Quoted strings are
anonymous tokens, reachable through `IdForNodeKind(name, false)` or by scanning
`Child(i)` for `!IsNamed()`.

| MetricID | Counted nodes / rule |
|---|---|
| `code_branch` | `if_statement` +1; its `alternative` +1 unless that node is itself an `if_statement` (FR-9). `switch_block_statement_group` +1 and `switch_rule` +1 when at least one `switch_label` child is not `"default"` (FR-10). `ternary_expression` +1. `for_statement`, `enhanced_for_statement`, `while_statement`, `do_statement` +1 each. Nothing for `try_statement`, `return`, `break`, `continue`, `throw`, `yield`, `instanceof`, `assert` or labelled statements. |
| `condition` | Leaf clauses of `binary_expression` chains whose `operator` is `"&&"` or `"\|\|"` (FR-8), flattened through `parenthesized_expression` and `unary_expression` with operator `"!"` via the `operand` field. Each clause is one occurrence on the operand node. Bitwise `"&"`, `"\|"`, `"^"` are 0. |
| `exception_handling` | `try_statement` and `try_with_resources_statement` +1, on the node in their `body` field (the guarded block, not the whole statement, so the range does not swallow the catches). Each `catch_clause` +1 — a multi-catch `catch (A \| B e)` is one clause and one point. `finally_clause` +1. `try { } catch { } finally { }` = 3. `throws` and `throw` are 0. |
| `internal_coupling` | +1 per import classified internal (FR-6) that the unit uses (FR-7). Occurrence on the `import_declaration` node. |
| `external_coupling` | +1 per import classified external, same attribution. |
| `inheritance` | +1 for the type under a `superclass` (`extends Base`). +1 per type in the `type_list` of a `super_interfaces` (`implements A, B`) and of an `extends_interfaces` (`interface I extends A, B`), so `: A, B` is two occurrences a reader can tell apart. Applies to `class_declaration`, `interface_declaration`, `enum_declaration` and `record_declaration` alike, nested or not — a nested type's heritage bills to the enclosing unit like everything else nested. +1 per anonymous class: an `object_creation_expression` that has a `class_body` child, occurrence on its `type` field. `permits` is 0. |
| `local_variable` | +1 per `variable_declarator` under a `local_variable_declaration`, a `field_declaration` or a `constant_declaration` (`int a, b;` = 2; occurrence on the declarator). +1 per `resource` that has a `name` field. +1 on the `name` of an `enhanced_for_statement`. +1 per `formal_parameter` under a `record_declaration`'s `parameters` (a record component). **Not** counted: method, constructor, lambda and `catch_clause` parameters; pattern variables; `enum_constant`; `spread_parameter`. |
| `lambda` | `lambda_expression` +1, `method_reference` +1 (`String::valueOf`, `ArrayList::new`, `this::m`). Anonymous classes are `inheritance`, not `lambda`. Java has no property units, so there is no unit-body exemption and the counter needs no `skipLambda`/`skipDeclaration`. |

Occurrence ranges follow TypeScript and Kotlin: `treesitter.SpanOf` from
`StartPosition()`/`EndPosition()`, 1-based, end exclusive.

## Worked fixtures (acceptance values)

Encode each as a fixture in `internal/analyze/java/testdata/` with a test
asserting the exact per-unit counts. Totals below assume every metric enabled at
weight 1.0, and the coupling fixtures assume `InternalPrefixes = ["com.acme"]`.

```java
// cdd_examples.java — docs/cdd.md section 2, verbatim rules
package com.acme.app;

class Examples {
    boolean check(int a, int b, int c, int d) {
        if (a > b && c < d) {              // code_branch 1, condition 2
            return true;
        }
        return false;
    }

    void guarded() {
        try {                              // exception_handling 1 (on the guarded block)
            first();
        } catch (RuntimeException e) {     // exception_handling 1
            second();
        } finally {                        // exception_handling 1
            third();
        }
    }

    int ifElse(int x) {
        if (x > 0) {                       // code_branch 1
            return 1;
        } else {                           // code_branch 1 — the alternative is not an if
            return 2;
        }
    }
}
// unit Examples: code_branch 3, condition 2, exception_handling 3, local_variable 0
```

```java
// branches.java
package com.acme.app;

import java.util.List;

class Branches {
    String oldSwitch(int x) {
        switch (x) {
            case 1:                        // code_branch 1 — a group with one label
                return "one";
            case 2:                        // same group as case 3
            case 3:                        // code_branch 1 — one group, two labels
                return "few";
            default:                       // 0
                return "many";
        }
    }

    String arrowSwitch(int x) {
        return switch (x) {
            case 1 -> "one";               // code_branch 1
            case 2, 3 -> "few";            // code_branch 1 — one rule
            default -> "many";             // 0
        };
    }

    int chain(boolean a, boolean b, boolean c) {
        if (a) {                           // code_branch 1
            return 1;
        } else if (b) {                    // code_branch 1 — the if; its else is an if, so 0
            return 2;
        } else if (c) {                    // code_branch 1
            return 3;
        } else {                           // code_branch 1 — the final alternative
            return 4;
        }
    }

    int ternary(int x) {
        return x > 0 ? 1 : 2;              // code_branch 1
    }

    void loops(List<Integer> xs, boolean flag) {
        for (int i = 0; i < 3; i++) {      // code_branch 1, local_variable 1 (i)
            noop();
        }
        for (Integer x : xs) {             // code_branch 1, local_variable 1 (x)
            noop();
        }
        while (flag) {                     // code_branch 1
            noop();
        }
        do {                               // code_branch 1
            noop();
        } while (flag);
    }
}
// unit Branches: code_branch 13, local_variable 2, condition 0
```

```java
// conditions.java
package com.acme.app;

class Conditions {
    boolean both(boolean a, boolean b) {
        return a && b;                     // condition 2
    }

    boolean either(boolean a, boolean b, boolean c) {
        return a && b || c;                // condition 3
    }

    boolean negated(boolean a, boolean b, boolean x) {
        return !(a || b) && x;             // condition 3 — flattened through ! and ( )
    }

    boolean compare(int x) {
        return x > 1;                      // condition 0 — a comparison is not a clause
    }

    int bits(int a, int b) {
        return a & b | 3;                  // condition 0 — bitwise operators are not clauses
    }
}
// unit Conditions: condition 8, code_branch 0
```

```java
// exceptions.java
package com.acme.app;

import java.io.Closeable;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;

class Exceptions {
    void guard(Path p, Closeable existing) {
        try (var in = Files.newInputStream(p); existing) {
            // exception_handling 1 (the guarded block), local_variable 1 (in);
            // `existing` declares no name, so it is 0
            in.read();
        } catch (IOException | RuntimeException e) {
            // exception_handling 1 — multi-catch is one clause; `e` is 0
            throw new IllegalStateException(e);   // 0 — throw is neither branch nor handling
        }
    }
}
// unit Exceptions: exception_handling 2, local_variable 1, code_branch 0
```

```java
// inheritance.java
package com.acme.app;

class Ledger extends Base implements Auditable, Printer {
    // inheritance 3 — Base, Auditable, Printer, one occurrence each
}

interface Auditable extends Named, Timestamped {
    // inheritance 2
}

enum Level implements Auditable {
    LOW, HIGH                              // inheritance 1, local_variable 0 (enum constants)
}

record Money(int amount, String currency) implements Comparable<Money> {
    // inheritance 1 (type arguments do not add), local_variable 2 (the components)
    public int compareTo(Money other) {
        return amount - other.amount;
    }
}

class Factory {
    Runnable r = new Runnable() {          // inheritance 1 (anonymous class, on `Runnable`),
        public void run() {                // local_variable 1 (the field r), lambda 0
        }
    };
}

sealed interface Shape permits Circle {
    // inheritance 0 — `permits` is not an edge; Circle lives elsewhere in the package
}
```

```java
// lambdas.java
package com.acme.app;

import java.util.ArrayList;
import java.util.List;
import java.util.function.Function;
import java.util.function.IntUnaryOperator;
import java.util.function.Supplier;

class Lambdas {
    void wire() {
        IntUnaryOperator doubled = x -> x * 2;              // lambda 1, local_variable 1
        Function<Integer, String> shown = String::valueOf;  // lambda 1, local_variable 1
        Supplier<List<String>> made = ArrayList::new;       // lambda 1, local_variable 1
        Runnable noop = () -> {};                           // lambda 1, local_variable 1
        run(doubled, shown, made, noop);
    }
}
// unit Lambdas: lambda 4, local_variable 4
```

```java
// locals.java
package com.acme.app;

class Locals {
    private int a, b;                      // local_variable 2 — one per declarator
    static final int MAX = 3;              // local_variable 1

    void body(Object o) {
        int x = 1, y = 2;                  // local_variable 2
        var z = x;                         // local_variable 1
        if (o instanceof String s) {       // code_branch 1, local_variable 0 (pattern variable)
            z = s.length();
        }
        try {                              // exception_handling 1
            check(y + z + MAX + a + b);
        } catch (Exception e) {            // exception_handling 1, local_variable 0 (catch param)
            return;
        }
    }

    private void check(int v) {
    }
}
// unit Locals: local_variable 6, code_branch 1, exception_handling 2

interface Consts {
    int LIMIT = 10;                        // local_variable 1 — a constant_declaration
}
// unit Consts: local_variable 1

enum Color {
    RED, GREEN                             // local_variable 0 — enum constants are not variables
}
// unit Color: local_variable 0
```

```java
// units.java — expected units in order, with kinds
package com.acme.app;

public class A {}                          // unit 1 — class, position at `class`
interface B {}                             // unit 2 — interface
enum C { X, Y }                            // unit 3 — enum
record D(int v) {}                         // unit 4 — record, local_variable 1 (component v)
@interface E {}                            // unit 5 — annotation, position at `@interface`
abstract class F {}                        // unit 6 — class, position at `class`, not `abstract`

final class G {                            // unit 7 — class; everything below bills to it
    static class Inner {}                  // not a unit
    class Nested {}                        // not a unit
    interface Shape {}                     // not a unit
    enum Kind { ONE }                      // not a unit
    record R(int v) {}                     // not a unit; local_variable 1 on G
    void m() {                             // not a unit — a member method
        class Local {}                     // not a unit
    }
}
// units in order: A, B, C, D, E, F, G — seven, no Inner, Nested, Shape, Kind, R, m or Local
```

```java
// compact.java — a Java 25 compact source file
int count = 1;                             // not a unit; its complexity is invisible

void main() {                              // unit `main` — kind method, position at `void`
    if (count > 0) {                       // code_branch 1
        System.out.println(count);
    }
}
// units: main (method); code_branch 1, local_variable 0
```

```java
// coupling.java — with InternalPrefixes = ["com.acme"]
package com.acme.billing;

import com.acme.shared.Money;              // internal, binds Money
import com.acme.shared.Ledger;             // internal, binds Ledger
import static com.acme.shared.Rates.rate;  // internal, binds rate (the member)
import java.time.Instant;                  // external, binds Instant
import java.util.*;                        // external, star: charged to every unit

class Invoice {
    Money amount;                          // internal_coupling +1 (Money), local_variable 1
    Instant at() {                         // external_coupling +1 (Instant)
        return Instant.ofEpochMilli(rate() + amount.cents());  // internal_coupling +1 (rate)
    }
}
// unit Invoice: internal_coupling 2, external_coupling 2 (Instant + star), local_variable 1

class Note {
    Ledger l;                              // internal_coupling 1, local_variable 1
}
// unit Note: internal_coupling 1, external_coupling 1 (the star only)

class Plain {
}
// unit Plain: internal_coupling 0, external_coupling 1 — the star charges every unit
```

```java
// coupling_no_star.java — the same file without the star import
package com.acme.billing;

import com.acme.shared.Money;
import com.acme.shared.Ledger;
import static com.acme.shared.Rates.rate;
import java.time.Instant;

class Invoice {
    Money amount;
    Instant at() {
        return Instant.ofEpochMilli(rate() + amount.cents());
    }
}
// unit Invoice: internal_coupling 2, external_coupling 1

class Note {
    Ledger l;
}
// unit Note: internal_coupling 1, external_coupling 0

class Plain {
}
// unit Plain: internal_coupling 0, external_coupling 0
```

```java
// broken.java — FR-5
class Oops { void f( { }
// expected: no units, one warning "syntax error at L:C", err == nil
```

```java
// empty.java — zero bytes: no units, no warnings, no error
```

```java
// header_only.java — a package line and imports and nothing else
package com.acme.app;

import java.util.List;
import java.util.Map;
// expected: no units, no warnings, no error
```

## Implementation guidance

- Follow feature 03's binding discipline verbatim: no finalizers, `Close()`
  every `Parser`/`Tree`/`TreeCursor`, never let a `*Node` outlive its tree, one
  parser and cursor per worker, kinds resolved to `uint16` once.
- Copy the shape of `internal/analyze/kotlin/parser.go`, not its values: a
  fresh `kind` enum for Java, `kindNames`, `fields`, `tokens`, `newGrammar`,
  `kindOf`, and one `sharedGrammar` behind a `sync.Once`. Java has one grammar,
  so there is no `grammars.forPath`; `.java` is the only extension and any other
  is an error, as Kotlin does for `.kts`.
- Unlike Kotlin, this grammar has the fields the analyzer wants. Use
  `alternative` for the `else` branch, `body` for the guarded block of a `try`,
  `operand` for the operand of a unary `!`, and `name` for a unit's name, a
  resource's binding and an `enhanced_for_statement`'s binding. Do not scan by
  child position where a field exists — `TestFieldsResolve` pins them.
- The anonymous tokens are `else` and `default`, both resolved with
  `IdForNodeKind(name, false)`. The `else` token is only needed to anchor the
  occurrence's start; the alternative itself comes from the field.
- `switch_expression` is the node for both `switch (x) { … }` as a statement and
  `switch (x) { … }` as an expression, so the arm rules need no second case.
  The old-style body is a `switch_block` of `switch_block_statement_group`s;
  the arrow body is a `switch_block` of `switch_rule`s.
- Reference collection for FR-7 must gather both `identifier` and
  `type_identifier` text: this grammar splits value and type names, unlike
  Kotlin's single `identifier`. Same by-name test, same documented shadowing
  caveat.
- Import path and star: `import_declaration` is `import` `static`? `_name`
  (`.` `asterisk`)? `;`. Read the path from the `scoped_identifier` or
  `identifier` child, the binding from that node's `name` field, and the star
  from the presence of an `asterisk` child. There is no alias to handle.
- Keep `funlen` (80 lines / 60 statements) and `gocyclo` (20) in mind when
  writing the counter's `switch`: split it into `countControlFlow` and
  `countDeclaration` the way TypeScript and Kotlin do, with `countElse`,
  `countSwitchArm`, `countTryBody`, `countCondition`/`clauses`,
  `countInheritance` and `countLoopBinding` as their own small functions.
- Reference smoke test before the docs commit: run `cdd check` over a few
  hundred files of a public, conventionally formatted Java project and confirm
  the syntax-warning rate is zero or attributable. Record any grammar gap in the
  README limitations rather than working around it in the counter.

## Deliverables

```
internal/analyze/internal/jvm/
    imports.go, imports_test.go     Module, IsInternal, Imports, UsedBy, Metric (FR-1)
internal/analyze/kotlin/
    imports.go, imports_test.go     grammar walk only; behaviour identical (FR-1)
internal/analyze/java/
    spec.go, spec_test.go           clone-on-return, whole-struct TestSpec (FR-2)
    parser.go                       kind enum, kindNames, fields, tokens, sharedGrammar (FR-11)
    analyzer.go                     NewAnalyzer, Analyze, Close, parse (FR-5)
    units.go                        units, declaredUnit, declarationStart (FR-3)
    metrics.go                      counter, visit, countControlFlow, countDeclaration,
                                    countElse, countSwitchArm, countTryBody,
                                    countCondition/clauses, countInheritance,
                                    countLoopBinding (FR-4, FR-8 … FR-10)
    imports.go                      grammar walk into jvm.Imports, countCoupling (FR-6, FR-7)
    analyzer_test.go, units_test.go, metrics_test.go, imports_test.go,
    occurrences_test.go, helper_test.go
    testdata/*.java                 the worked fixtures above (testdata/project/ stays)
internal/languages/languages.go     FR-12, one line
cmd/check_java_test.go              e2e, mirroring cmd/check_kotlin_test.go
go.mod, go.sum                      tree-sitter-java v0.23.5
README.md                           language paragraph, binary-size note, known limitations
CONTRIBUTING.md                     grammar provenance sentence
docs/features/05-java-support/test-cases.md   case ids cited from the tests (`// TC-…`)
```

## Tasks

Commit per task; the task title is the commit message, verbatim and alone — no
trailers. Each task's acceptance criterion is shorthand for the cases listed
under the same task heading in [test-cases.md](test-cases.md); those cases are
the contract.

- **T0 — `docs: add Java support feature spec`**. This file and
  [test-cases.md](test-cases.md). *Accept:* the rules, fixtures and case ids
  below are the ones the following ten commits implement.

- **T1 — `refactor: share JVM import attribution between analyzers`** (FR-1).
  *Accept:* `make test` green; `git diff --stat` touches only
  `internal/analyze/kotlin/*` and the new `jvm` files; a Kotlin fixture run
  before and after produces identical JSON reports (TC-R3).

- **T2 — `fix: hand out copies of the Java spec`** (FR-2). Clone-on-return,
  whole-struct `TestSpec`, mutation test. *Accept:* TC-S1 … TC-S4 pass with the
  detection expectation unchanged.

- **T3 — `feat: parse Java with tree-sitter`** (FR-5, FR-11).
  `go get github.com/tree-sitter/tree-sitter-java@v0.23.5`, `parser.go`,
  `analyzer.go` returning zero units. *Accept:* TC-P1 … TC-P11 pass; a broken
  file yields the warning and no error.

- **T4 — `feat: extract Java units`** (FR-3). *Accept:* `units.java` yields
  exactly the seven listed units, kinds, names and positions, in order;
  `compact.java` yields the `main` method unit.

- **T5 — `feat: count Java branches and conditions`** (FR-8, FR-9, FR-10).
  *Accept:* `cdd_examples.java`, `branches.java` and `conditions.java` totals
  reproduce exactly.

- **T6 — `feat: count Java exceptions, inheritance and locals`**. *Accept:*
  `exceptions.java`, `inheritance.java` and `locals.java` reproduce.

- **T7 — `feat: count Java lambdas`**. *Accept:* `lambdas.java` reproduces,
  including the zero on the anonymous class in `inheritance.java`.

- **T8 — `feat: attribute Java imports to the units that use them`** (FR-6,
  FR-7). *Accept:* `coupling.java` and `coupling_no_star.java` reproduce.

- **T9 — `feat: register the Java analyzer`** (FR-12). One line in
  `languages.go`, `cmd/check_java_test.go`, occurrence invariants over every
  fixture. *Accept:* `cdd check` on a Java fixture project works end-to-end in
  all four formats; TC-X1 … TC-X6 run over every `testdata/*.java`.

- **T10 — `docs: describe Java support`**. README installation paragraph
  ("TypeScript, Kotlin and Java analyzers embed Tree-sitter"), measured
  binary-size note, "Language support" paragraph (Java joins; only Go stays
  `init`-only) with a Java rules summary and known limitations: same-package and
  fully-qualified references invisible, bitwise `&`/`|` not conditions,
  top-level statements of compact source files not units, and anything the smoke
  test turned up. CONTRIBUTING: the Java grammar comes from the `tree-sitter/`
  org. *Accept:* README and CONTRIBUTING reviewed against the shipped behaviour;
  the config template comments already cover Java and need no edit.

## Definition of done

- [ ] `make build`
- [ ] `make test` (race detector on)
- [ ] `make lint` (including `check-literals`)
- [ ] `make fmt` leaves no diff
- [ ] Coverage ≥ 90 % for `internal/analyze/java` and
      `internal/analyze/internal/jvm`; `internal/analyze/kotlin` and
      `internal/analyze/typescript` coverage does not drop
- [ ] Every worked fixture above is a checked-in test with the stated totals
- [ ] Every case in [test-cases.md](test-cases.md) is a checked-in test, citing
      its id, and passes; cross-cutting invariants TC-X1 … TC-X6 run over every
      fixture under `testdata/`
- [ ] The T1 refactor commit produces byte-identical Kotlin reports on the
      existing fixtures
- [ ] CI dogfood gate green
- [ ] Nothing outside `internal/analyze/java`, `internal/analyze/internal/jvm`,
      `internal/analyze/kotlin/imports*.go`, `languages.go`,
      `cmd/check_java_test.go` and the docs listed above changed — verified with
      `git diff --stat main`

## Suggested order

T0 → T1 → T2 → T3 → T4 → T5 → T6 → T7 → T8 → T9 → T10. T2 is independent of T1
and may land first. T5–T8 each depend on T4; they are independent in principle
but all touch `metrics.go`, `imports.go` and `parser.go`, so they land one at a
time in the order listed. T9 depends on all of them.
