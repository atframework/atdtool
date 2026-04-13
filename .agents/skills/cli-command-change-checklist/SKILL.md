---
name: cli-command-change-checklist
description: Review or implement user-visible CLI command changes in this repository. Use when changing subcommands, flags, help text, defaults, output paths, or command behavior.
---

# CLI command change checklist

1. Inspect command wiring in `cmd/atdtool/atdtool.go` and the specific command file before editing behavior.
2. Preserve existing flags, defaults, positional arguments, output naming, and help structure unless the request explicitly changes them.
3. Keep error messages actionable and specific about missing paths, invalid values, or unsupported combinations.
4. Update nearby tests for changed command behavior, especially when flags alter merge inputs, render outputs, or filesystem layout.
5. Update user-facing docs in the same change:
   - `README.md`
   - matching `docs/usage/*.md` pages
6. If a command affects render or merge semantics, also review the related values-precedence docs and tests.
7. Verify with focused package tests first; finish with `go test ./...` when the task is complete.
