# copier/bench — 本地 copier vs 主流库性能对比

本目录是**独立**的 Go 模块，用于将本地 `github.com/charlienet/go-misc/copier`
与主流拷贝库做性能对比。主模块保持零依赖，本目录通过 `replace` 指令引用本地路径：

- 本地：`github.com/charlienet/go-misc/copier`（`Copy(&dst, &src)`，深拷贝）
- [jinzhu/copier](https://github.com/jinzhu/copier) v0.4.0（`Copy(&dst, &src)` 默认浅拷贝；
  深拷贝用 `CopyWithOption(&dst, &src, Option{DeepCopy: true})`）
- [tiendc/go-deepcopy](https://github.com/tiendc/go-deepcopy) v1.7.2（`Copy(&dst, &src)`，深拷贝）

## 运行

```bash
cd copier/bench
go test -run '^$' -bench . -benchmem -count=3
```

首次运行前需要拉取外部依赖（需网络）：

```bash
go mod tidy
```

## 对比场景

| 场景 | 说明 |
|------|------|
| BenchmarkStructSameType | 同类型 struct→struct（10 字段混合类型，对齐主模块 `benchMedium`） |
| BenchmarkStructCrossType | 跨类型 struct→struct（同名字段 + 可转换类型：int→int64、float32→float64） |
| BenchmarkStructNested | 带嵌套 struct / 指针 / 切片 / map 的 struct（体现深拷贝成本） |

每个场景包含 4 个子基准：`local`（本地）、`jinzhu`（浅拷贝）、`jinzhu-deep`、
`tiendc`。统一 `-benchmem`。

## 公平性约定

- 每个迭代内 `var dst ...` 新建目标，避免复用 dst 时库之间对 slice/map
  字段行为不一致（如 append 而非覆盖）。
- 拷贝结果写入 package 级 `sink` 变量，防止编译器优化。
- 三库使用完全相同的调用模式与目标类型。
