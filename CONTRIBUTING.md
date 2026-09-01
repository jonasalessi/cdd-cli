# Contributing to cdd-cli

## Run the setup first

Clone the repo, then run this once:

```sh
make setup
```

It sets `core.hooksPath` to `.githooks/`, so git runs the `commit-msg` hook
checked into this repo. The hook rejects any commit whose message does not
match the format below. Skip the setup and git will happily accept a bad
message, and the reviewer will ask you to rewrite it.

## What the hooks do

`pre-commit` runs `gofmt` and `golangci-lint run --fix` on the staged Go
files, re-stages whatever they fixed, and blocks the commit only when an
issue cannot be fixed automatically. It refuses partially staged Go files,
since re-staging those would commit hunks you left out on purpose.

`commit-msg` checks the message format below.

## Commit message format

```
<type>: <description>
```

Allowed types: `feat`, `fix`, `refactor`, `perf`, `docs`, `test`, `build`, `ci`.

```
feat: add init command
fix: handle missing config file
docs: describe ICP calculation
```

The type is lowercase, followed by a colon and one space. Make one commit per
remediation batch. Do not add a `Co-Authored-By` trailer.

If the hook rejects your message, do not `git commit --amend`. Fix the message
and create a new commit.

## Code style

Follow [Effective Go](https://go.dev/doc/effective_go).
