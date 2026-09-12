# Standard-library classification

An analyzer decides which of its imports are the standard library, and it
does so with a predicate it hands to the classifier it shares:
`jvm.NewImports(prefixes, stdlib)` for the JVM languages, its own `classify`
for TypeScript and for Go.

## Where the knowledge lives

The list that predicate reads lives in the language package's `stdlib.go`,
except for the JDK table Java and Kotlin share, which lives in
`internal/analyze/internal/jvm/stdlib.go`.

Go needs no list at all. `internal/analyze/golang/stdlib.go` holds a rule
instead: an import path whose first element carries no dot is resolved
against GOROOT, so a package added next release classifies correctly the
first time it is imported.

## Two rules to keep

`internal/analyze/*/stdlib.go` is exempt from the literal check the way
`spec.go` is, because module names such as Node's `console` collide with
vocabulary ids.

A configured project prefix always wins over the standard library, so a
project that lists `java` or `path` among its own packages keeps counting it
as internal coupling.
