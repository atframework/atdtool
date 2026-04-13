---
name: repo-implementation
description: Implement or refactor production code in this Go repository. Use when adding features, fixing bugs, changing CLI behavior, or modifying files under cmd/, cli/, internal/, pkg/, or logarchive/.
---

# Repo implementation

1. Read `AGENTS.md`, then inspect the smallest relevant package, command file, and nearby tests.
2. Preserve existing CLI flags, YAML keys, Helm/value-merge semantics, and output filenames unless the request explicitly changes them.
3. Reuse local patterns from neighboring code before adding new abstractions or dependencies.
4. If behavior changes, update tests near the changed package.
5. If behavior is user-visible, update `README.md` and the relevant `docs/` pages in the same change.
6. Verify with focused `go test` first; use `go test ./...` for shared logic or final verification.
7. For filesystem work, prefer `filepath` helpers and close files before cleanup.

## Hotspots

- `cmd/atdtool/`
- `cli/values/`
- `internal/pkg/util/`
- `internal/pkg/noncloudnative/`
