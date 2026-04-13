---
applyTo: "**/*_test.go,**/testdata/**"
---

- Keep tests deterministic and platform-neutral.
- Prefer table-driven tests for merge precedence, parser variants, and configuration matrices.
- Use `t.TempDir()`, `filepath.Join`, and isolated temp files instead of shared paths such as `/tmp/...`.
- Keep fixtures compact, readable, and scoped to the package under test.
- For bug fixes, reproduce with a failing or missing test first, then implement the code change, then rerun focused tests before `go test ./...`.
