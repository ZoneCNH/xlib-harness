# Changelog

## v0.1.1 - 2026-06-18

- 将验收门禁补强为可机械复验：partial pass、静默失败、只读检查、负例门禁、安全边界和 `xlib-standard` import/module 禁止均纳入测试。
- 增加 `BenchmarkGenerate` 与 `BenchmarkCheckFullProfile`，固化 BR/NFR 延迟预算证据。
- 保持公开 CLI 行为、Go module 路径和 `xlib-standard` 依赖边界兼容；无破坏性变更。

### 验证

- `go test ./...`：PASS
- `go test ./... -race -count=1`：PASS
- `go vet ./...`：PASS
- `go test ./... -coverprofile=coverage.out`：PASS，total `88.8%`
- `go test -bench=. ./...`：PASS，`BenchmarkGenerate` 约 `220281 ns/op`，`BenchmarkCheckFullProfile` 约 `223926 ns/op`
- CLI smoke：build、dependency-boundary、template-validate、generate、check-full、readonly、negative-gates、explicit-xlib-standard-rejected、security-boundary 全部 PASS
- Secret scan：PASS

## v0.1.0 - 2026-06-14

- 初始发布 xlib-harness：模块生成器与门禁执行器。
- 提供 generate/scaffold、spec-lint、boundary-check、traceability-gate 与 format-check 基础能力。
