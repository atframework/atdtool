---
name: helm-values-precedence-review
description: Review or change Helm/chart values precedence in this repository. Use when working on global.yaml, chart-name yaml, modules, runtime values, template overrides, or --set merge behavior.
---

# Helm values precedence review

1. Start from the implementation, not from old docs. Inspect `internal/pkg/util/chart.go` first, then the nearest tests.
2. Confirm the effective merge order for chart defaults, `global.yaml`, chart-name yaml, `modules/*.yaml`, runtime/template values, and `--set`.
3. Check module enable/disable behavior and whether disabled modules are removed or merely ignored.
4. For template-related changes, also inspect `cmd/atdtool/template.go` and related tests.
5. Keep deep-merge semantics and alias mapping (`type_name`, `func_name`, chart name) explicit in both code and tests.
6. If precedence changes, update tests and docs together:
   - `internal/pkg/util/chart_test.go`
   - `cmd/atdtool/template_test.go` when runtime/template values are affected
   - `docs/usage/values-and-overrides.md`
   - `docs/usage/modules.md`
   - `docs/usage/merge-values.md` or `docs/usage/template.md` when command behavior changes
7. Verify with focused tests first, then `go test ./...`.
