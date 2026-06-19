# Changelog

## v0.1.2 - 2026-06-19

- 补齐 `StdlibHarness` 公开 Go API，支持生成选项、完整检查结果和 CLI 入口测试。
- 将 SPEC、traceability、boundary 和 format 门禁升级为结构化检查：23 节 SPEC、FR WHEN/THEN、AC/TC 可追溯、Go import/go.mod 禁止依赖、Markdown 空链接和表格列数漂移。
- 新增 Makefile 与 CI/CD 验收聚合：build、test、race、vet、100% coverage 和正反例 boundary gates。
- 发布工作流升级到 `xlibgate@v1.0.2` 并在 tag release 前执行同款本地验收。

### 验证

- `go test ./...`：PASS
- `go test ./... -race -count=1`：PASS
- `go vet ./...`：PASS
- `go test ./... -coverprofile=coverage.out -covermode=count`：PASS
- `go tool cover -func=coverage.out`：total `100.0%`
- `go test -bench=. ./...`：PASS，`BenchmarkGenerate` 约 `113809 ns/op`，`BenchmarkCheckFullProfile` 约 `733850 ns/op`
- `make ci`：PASS

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
