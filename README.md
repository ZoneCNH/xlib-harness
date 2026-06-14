# xlib-harness

> FoundationX 模块生成器与门禁执行器

## 职责

- **generate / scaffold**：基于 xlib-standard 模板生成新模块骨架
- **spec-lint**：SPEC.md 结构合规性检查
- **boundary-check**：模块边界守卫（testkitx 生产导入检测、依赖矩阵验证）
- **traceability-gate**：FR→AC→TC 追溯链完整性验证
- **format-check**：Go 代码格式一致性检查

## Go Module

本模块与 [xlib-standard](https://github.com/ZoneCNH/xlib-standard) 共享 Go module (`github.com/ZoneCNH/xlib-standard`)。代码位于 xlib-standard 仓库的 `cmd/` 和 `pkg/` 目录下。

## 相关文档

- 完整规格：[SPEC.md](https://github.com/ZoneCNH/ZoneCNH/blob/main/module/xlib-harness/SPEC.md)
- Goal 定义：[goal.md](https://github.com/ZoneCNH/ZoneCNH/blob/main/module/xlib-harness/goal.md)
- 追溯矩阵：[TRACEABILITY.md](https://github.com/ZoneCNH/ZoneCNH/blob/main/module/xlib-harness/TRACEABILITY.md)
