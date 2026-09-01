# Project Overview
CLI (`cdd`) that measures code quality with Cognitive-Driven Development (CDD)
and Intrinsic Complexity Points (ICPs). Written in Go.

## Critical Rules
- Follow strict the "Effective Go" for writing clear, idiomatic Go code.

## Commit style
- Format <type>: <description>; prefixes feat|fix|refactor|perf|docs|test|build|ci.
- One commit per remediation batch.
- If a pre-commit hook fails, do not git commit --amend, fix the issue and create a new commit.
- Do not add Co-Author

## Definition of Done (MANDATORY)
IMPORTANT: NEVER report a feature, change, or fix as done — and NEVER commit —
until ALL of these pass:
- `make build`
- `make test`
- `make lint`
- `make fmt` (leaves no diff)

If any step fails, fix the problem and re-run every step. No exceptions,
even for "trivial" changes.
