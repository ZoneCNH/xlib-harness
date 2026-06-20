# xlib-harness

> FoundationX 模块生成器与门禁执行器

## 发布状态

- 当前版本：`v0.1.5`
- 代码验收基线：`xlib-harness` 分支 v0.1.5 候选提交（随发布 tag 固化）
- 验收日期：2026-06-20

`v0.1.5` 是代码仓库同步文档与发布合约补齐发布：仓库本体新增 `FEATURES.md` 与 `ACCEPTANCE.md`，CI/CD 和 Release workflow 将这两份文档纳入非空合约检查，并继续保持 Makefile 聚合验收和 100% 语句覆盖率门槛。

## 职责

- **generate / scaffold**：生成带 23 节 SPEC、追溯矩阵、Goal、计划和任务文件的新模块骨架
- **spec-lint**：SPEC.md 结构、FR WHEN/THEN、AC 可验证性和测试命令检查
- **boundary-check**：模块边界守卫（Go import、go.mod 禁止依赖检测）
- **traceability-gate**：FR→AC→TC 追溯链完整性验证
- **format-check**：Markdown 文档格式、空链接、表格列数和模板残留检查

## Go Module

本模块拥有独立 Go module：`github.com/ZoneCNH/xlib-harness`。

`xlib-standard` 仅作为标准源与模板来源被读取；xlib-harness 不允许引入 `github.com/ZoneCNH/xlib-standard` Go import 或 module dependency。

## 验收与性能基线

发布前验证命令：

```bash
go test ./...
go test ./... -race -count=1
go vet ./...
go test ./... -coverprofile=coverage.out -covermode=count
go tool cover -func=coverage.out
go test -bench=. ./...
make ci
```

2026-06-20 本地验收结果：

- `go test ./...`：PASS
- `go test ./... -race -count=1`：PASS
- `go vet ./...`：PASS
- `go test ./... -coverprofile=coverage.out -covermode=count`：PASS
- `go tool cover -func=coverage.out`：total `100.0%`
- `go test -bench=. ./...`：PASS，`BenchmarkGenerate` 约 `439979 ns/op`，`BenchmarkCheckFullProfile` 约 `4802818 ns/op`
- `make ci`：PASS

## 相关文档

- 发布记录：[CHANGELOG.md](CHANGELOG.md)
- 功能清单：[FEATURES.md](FEATURES.md)
- 验收证据：[ACCEPTANCE.md](ACCEPTANCE.md)
- 完整规格：[SPEC.md](https://github.com/ZoneCNH/ZoneCNH/blob/main/module/xlib-harness/SPEC.md)
- Goal 定义：[goal.md](https://github.com/ZoneCNH/ZoneCNH/blob/main/module/xlib-harness/goal.md)
- 追溯矩阵：[TRACEABILITY.md](https://github.com/ZoneCNH/ZoneCNH/blob/main/module/xlib-harness/TRACEABILITY.md)
