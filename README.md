# xlib-harness

> FoundationX 模块生成器与门禁执行器

## 发布状态

- 当前版本：`v0.1.1`
- 代码验收基线：`335eef9`
- 验收日期：2026-06-18

`v0.1.1` 是 `v0.1.0` 之后的补丁发布，重点补齐可机械复验的验收证据：单测、race、vet、覆盖率、生成与完整检查基准、CLI smoke、只读行为、负例门禁、安全边界和 `xlib-standard` Go import/module dependency 禁止。

## 职责

- **generate / scaffold**：基于 xlib-standard 模板生成新模块骨架
- **spec-lint**：SPEC.md 结构合规性检查
- **boundary-check**：模块边界守卫（testkitx 生产导入检测、依赖矩阵验证）
- **traceability-gate**：FR→AC→TC 追溯链完整性验证
- **format-check**：Go 代码格式一致性检查

## Go Module

本模块拥有独立 Go module：`github.com/ZoneCNH/xlib-harness`。

`xlib-standard` 仅作为标准源与模板来源被读取；xlib-harness 不允许引入 `github.com/ZoneCNH/xlib-standard` Go import 或 module dependency。

## 验收与性能基线

发布前验证命令：

```bash
go test ./...
go test ./... -race -count=1
go vet ./...
go test ./... -coverprofile=coverage.out
go test -bench=. ./...
```

2026-06-18 本地验收结果：

- `go test ./...`：PASS
- `go test ./... -race -count=1`：PASS
- `go vet ./...`：PASS
- 覆盖率：total `88.8%`，核心包 `89.2%`
- `BenchmarkGenerate`：约 `220281 ns/op`
- `BenchmarkCheckFullProfile`：约 `223926 ns/op`

## 相关文档

- 发布记录：[CHANGELOG.md](CHANGELOG.md)
- 完整规格：[SPEC.md](https://github.com/ZoneCNH/ZoneCNH/blob/main/module/xlib-harness/SPEC.md)
- Goal 定义：[goal.md](https://github.com/ZoneCNH/ZoneCNH/blob/main/module/xlib-harness/goal.md)
- 追溯矩阵：[TRACEABILITY.md](https://github.com/ZoneCNH/ZoneCNH/blob/main/module/xlib-harness/TRACEABILITY.md)
