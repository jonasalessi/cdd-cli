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