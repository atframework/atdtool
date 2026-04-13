# 渲染流程说明

本文档用步骤化的方式说明 `merge-values` 和 `template` 的主要执行链路，方便排查“为什么最终值是这个结果”。

## 1. `merge-values` 流程

### merge-values 步骤 1：解析命令行

读取：

- `CHART`
- `--values/-p`
- `--set/-s`
- `--output/-o`

### merge-values 步骤 2：规范化 values 路径

由 `cli/values.Options.MergePaths()` 完成：

- 处理绝对路径 / 相对路径
- 处理 `~` 开头的 home 路径
- 检查路径是否存在

### merge-values 步骤 3：解析命令行覆盖值

由 `cli/values.Options.MergeValues()` 完成，使用 Helm `strvals` 语法把 `--set` 转成嵌套 map。

### merge-values 步骤 4：加载 chart 默认值

`internal/pkg/util.MergeChartValues()` 会先加载 chart 自带的 `values.yaml`。

### merge-values 步骤 5：读取配置组目录

对每个 values path：

1. 读取 `global.yaml`
2. 读取 charts 同名 yaml（按 `type_name` / `func_name` / chart 名解析）
3. 记录 `modules/*.yaml`

### merge-values 步骤 6：计算最终 values

当前实现的同 key 优先级：

1. `--set`
2. 服务级同名 yaml（后 path 优先）
3. chart 默认值
4. `global.yaml`（后 path 优先）
5. 已启用模块（后 path 优先）

### merge-values 步骤 7：输出合并结果

最终 values 会被序列化为 YAML 并写到目标路径。

## 2. `template` 流程

### template 步骤 1：解析命令行

读取：

- `CHART`（这里是 chart 根目录）
- `--values/-p`
- `--set/-s`
- `--output/-o`

### template 步骤 2：加载非云原生部署清单

调用 `internal/pkg/noncloudnative.LoadConfig()`：

- 递归扫描 values path
- 找到 `deploy.yaml`
- 加载 `world_id`、`zone_id` 和 `proc_desc`

### template 步骤 3：遍历实例清单

对 `proc_desc` 里的每个实例：

- 按 `instance_count` 展开实例
- 计算实例号 `instance_id`
- 计算 `bus_addr`

### template 步骤 4：构造实例级 values

对每个实例，会组装：

- 实例专属 `type_id`
- `global.*` 命令行覆盖（扁平化后作用于所有实例）
- `<实例名>.*` 命令行覆盖（只作用于对应实例）
- 由 `deploy.yaml` 派生的 `world_id` / `zone_id` / `instance_id` / `bus_addr`

### template 步骤 5：调用 `MergeChartValues()`

继续叠加：

- chart 默认值
- `global.yaml`
- 服务级同名 yaml
- 已启用模块
- 实例级运行时值
- 命令行覆盖值

### template 步骤 6：筛选可输出模板文件

`template` 命令不会直接输出 `templates/_*.tpl` 这样的 helper。

当前实现主要会把 chart 中的普通 `.tpl` 文件作为输出目标，常见是：

- `cfg/*.yaml.tpl`
- `bin/*.sh.tpl`
- `bin/*.bat.tpl`

这些模板文件内部可以继续 `include` 其他 helper。

### template 步骤 7：渲染与落盘

每个实例会：

- 生成独立的输出目录
- 按 `bus_addr` 给输出文件加后缀
- 写出配置文件和脚本

## 3. 调试顺序建议

如果发现最终输出与预期不一致，推荐按这个顺序查：

1. `--set` 是否写到了正确路径
2. 服务级同名 yaml 文件名是否与 `type_name` / `func_name` 对齐
3. chart 默认值是否已经在 `values.yaml` 中定义了更高优先级的同名 key
4. 模块是否真的被启用，且模板里是否做了存在性保护
5. `deploy.yaml` 是否来自你预期的那一个 path
6. `template` 输出目录和文件名是否带上了正确的 `bus_addr`

## 4. 常见误区

### 误区一：认为 `global.yaml` 会覆盖 chart 默认值

不是。当前实现中，chart 默认值优先于 `global.yaml` 的同名 key。

### 误区二：认为模块配置会强覆盖服务级配置

不是。模块更像默认补齐层。

### 误区三：认为 `template` 的 `CHART` 是单个 chart 目录

不是。当前实现里它是 chart 根目录，实例名会再拼接到这个路径下面。

### 误区四：认为 `template` 和 `helm template` 的上下文完全一致

不是。`atdtool template` 当前主要围绕 `.Values` 工作，`.Release` / `.Capabilities` 不会完整模拟 Helm 发布上下文。

## 相关阅读

- [`../usage/merge-values.md`](../usage/merge-values.md)
- [`../usage/template.md`](../usage/template.md)
- [`../usage/values-and-overrides.md`](../usage/values-and-overrides.md)
