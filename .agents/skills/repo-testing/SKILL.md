---
name: repo-testing
description: Write, repair, or stabilize tests in this Go repository. Use when adding unit tests, reproducing bugs with tests, fixing failing or flaky tests, updating testdata, or improving coverage for changed behavior.
---

# Repo testing

1. Reproduce with the smallest scope possible first: package, file, or `-run` target.
2. For bug fixes, prefer adding or updating the failing test before the code fix.
3. Prefer table-driven tests for precedence, parser variants, and configuration matrices.
4. Use `t.TempDir()`, `filepath.Join`, isolated temp files, and deterministic fixtures.
5. Avoid shared temp paths, random sleeps, and hidden global state.
6. Keep `testdata/` compact, readable, and close to the package under test.
7. Close files before cleanup; Windows failures often come from open handles.
8. After focused tests pass, widen verification as needed and finish with `go test ./...` when appropriate.
