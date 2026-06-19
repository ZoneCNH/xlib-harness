# Changelog

## v0.1.3 - 2026-06-20

- `generate` 现在产出完整的标准模块资产集：README、SPEC、TRACEABILITY、goal、IMPLEMENTATION-PLAN、ACCEPTANCE、FEATURES、tasks/TASK-001，外加 `Makefile` 与 `.github/workflows/ci.yml` 桩——脚手架模块开箱即 CI-ready（兑现 FR-001）。
- `check --profile full` 新增 `ci-reference` 门禁：验证模块根 `Makefile` 含 `ci` 目标且 `.github/workflows/` 存在工作流（兑现 FR-004）。full profile 现为 15 项检查。
- `writeResult` 传播 JSON 编码与流写入错误（BR-004 / SPEC §11）：`--json` 管道提前关闭时不再静默产出残缺输出，改为非零退出并在 stderr 报错。
- `countHeadings` 改为 fence 感知：代码块内的 `#` 不再虚增 spec-section-depth 计数，杜绝门禁绕过。
- CI/CD：`runs-on` 由不存在的 `sre/gate`/`sre/deploy` self-hosted 标签改为 `ubuntu-latest`（远端 CI 此前因无 runner 领取而排队 24h 后取消），`setup-go` 显式 `cache: true`。

### 验证

- `go test ./...`：PASS
- `go test ./... -race -count=1`：PASS
- `go vet ./...`：PASS
- `go test ./... -coverprofile=coverage.out -covermode=count`：PASS
- `go tool cover -func=coverage.out`：total `100.0%`
- `go test -bench=. ./...`：PASS
- `make ci`：PASS（compliant full 15 项、bad-dependency/broken-trace 负例如预期失败）

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
