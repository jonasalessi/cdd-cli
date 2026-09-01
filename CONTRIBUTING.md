# Contributing to cdd-cli

Thanks for your interest in contributing!

## Before you start (required)

After cloning the repository, run the setup once:

```sh
make setup
```

This points git at the versioned hooks in `.githooks/`, which enforce the
commit message format below. **Commits made without running `make setup`
will not be validated locally and may be rejected during review.**

## Commit message format

Every commit message must follow:

```
<type>: <description>
```

Allowed types: `feat`, `fix`, `refactor`, `perf`, `docs`, `test`, `build`, `ci`.

Examples:

```
feat: add init command
fix: handle missing config file
docs: describe ICP calculation
```

Rules:

- The type is lowercase and followed by a colon and a single space.
- One commit per remediation batch.
- Do not add a `Co-Authored-By` trailer.
- If a hook fails, do not `git commit --amend`; fix the issue and create a new commit.

## Code style

Follow [Effective Go](https://go.dev/doc/effective_go) for clear, idiomatic Go code.
