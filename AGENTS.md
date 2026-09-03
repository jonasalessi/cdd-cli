# Project Overview
CLI (`cdd`) that measures code quality with Cognitive-Driven Development (CDD)
and Intrinsic Complexity Points (ICPs). Written in Go.

## Critical Rules
- Follow strict the "Effective Go" for writing clear, idiomatic Go code.
- Plan tests around command behavior and package contracts: use unit tests for pure logic, and integration tests for CLI boundaries such as arguments, flags, stdin/stdout/stderr, exit codes, filesystem access, and interaction between internal packages. Keep tests deterministic, avoid excessive mocking, and verify observable CLI behavior rather than internal function calls.
- Apply Single Responsibility at the package and type level: each package should represent one clear capability, and each struct or function should have one clear reason to change. Prefer small focused interfaces, short functions, and composition over large “manager” or “service” types that mix parsing, validation, filesystem access, formatting, and CLI orchestration.
- Commit when you reach a logical checkpoint that you could explain in one sentence. If you are implementing a task and it contains FR then commit by FR.
- When adding support for a new language, follow the "Adding a language" section in CONTRIBUTING.md.

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
