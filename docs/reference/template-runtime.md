# 模板运行时参考

本文档说明 `atdtool` 在模板渲染时提供的运行时上下文、额外变量，以及本工程实际暴露给 chart 的模板接口边界。

## 1. Helm 原生对象与函数

`atdtool` 的模板渲染底层使用 Helm 引擎，因此模板语法仍然遵循 Helm / Go template / Sprig 的规则。

Helm 官方文档入口：

- Built-in Objects：<https://helm.sh/docs/chart_template_guide/builtin_objects/>
- Variables：<https://helm.sh/docs/chart_template_guide/variables/>
- Functions and Pipelines：<https://helm.sh/docs/chart_template_guide/functions_and_pipelines/>
- Function List：<https://helm.sh/docs/chart_template_guide/function_list/>
- Named Templates：<https://helm.sh/docs/chart_template_guide/named_templates/>

当文档里没有特别说明时，函数、对象和变量的解释以 Helm 官方文档为准，不在本仓库重复维护。

## 2. `atdtool` 传入的上下文边界

在 `atdtool template` 的当前实现中：

- `.Values`：由 `atdtool` 组装并传入，是最核心的输入
- `.Chart`、`.Files`、`.Template`、`.Subcharts`：由 Helm 渲染引擎按 chart 自动补充
- `.Release`、`.Capabilities`：当前不会像 `helm template` 那样完整填充，依赖它们的模板需要谨慎使用

因此：

- 配置文件 / 启停脚本类模板应优先依赖 `.Values`
- 如果模板需要完整的 Helm 发布上下文，优先考虑 Helm 标准渲染链路

## 3. `atdtool` 额外注入的 `.Values`

以下变量由 `atdtool` 在 `template` 模式下额外注入或改写：

| 变量 | 来源 | 说明 |
| --- | --- | --- |
| `.Values.world_id` | `deploy.yaml` 或 `--set global.world_id` | 当前实例所属 world |
| `.Values.zone_id` | `deploy.yaml` 或 `--set global.zone_id` | 当前实例所属 zone |
| `.Values.instance_id` | `deploy.yaml` 展开后的实例号 | 当前实例 ID |
| `.Values.bus_addr` | 由 world/zone/type/instance 组合生成 | 当前实例 bus 地址 |
| `.Values.atdtool_running_platform` | 运行时 `runtime.GOOS` | 当前运行平台 |
| `.Values.type_id` | `deploy.yaml` 中的 `instance_type_id` | 当前实例类型 ID |

补充说明：

- `global.*` 的命令行参数在 `template` 模式下会被扁平化到实例顶层 values 中
- `<实例名>.*` 的命令行参数只作用于对应实例
- chart 自带的 `type_name` / `func_name` 不属于额外运行时值，但会影响服务级同名 yaml 的文件名解析

## 4. Values 的组成来源

渲染一个实例时，最终 `.Values` 由以下来源组合而成：

1. chart 自带 `values.yaml`
2. values path 中的 `global.yaml`
3. values path 中的 charts 同名 yaml
4. values path 中的已启用模块 `modules/*.yaml`
5. 非云原生实例运行时值
6. 命令行 `--set`

其中同 key 的真实优先级，见：

- [`../usage/values-and-overrides.md`](../usage/values-and-overrides.md)

## 5. 本工程实际暴露给 chart 的模板接口

### 5.1 可直接访问的顶层对象

| 对象 | 来源 | 说明 |
| --- | --- | --- |
| `.Values` | `atdtool` 组装 | 当前 chart 最重要的输入值 |
| `.Chart` | Helm 渲染引擎 | 当前 chart 元数据 |
| `.Files` | Helm 渲染引擎 | 当前 chart 文件访问接口 |
| `.Template` | Helm 渲染引擎 | 当前模板名称与基础路径 |
| `.Subcharts` | Helm 渲染引擎 | 子 chart 视图 |
| `.Release` | Helm 渲染引擎 | 当前命令不会完整填充，谨慎依赖 |
| `.Capabilities` | Helm 渲染引擎 | 当前命令不会完整填充，谨慎依赖 |

### 5.2 命名模板接口

`atdtool` 本身**不会额外注册业务侧的命名模板**。如果 chart 需要可复用模板接口，应由 chart 自己通过 Helm 的标准机制提供，例如：

- `define`
- `include`
- `template`
- `tpl`

这些命名模板通常由 chart 放在：

- `templates/_*.tpl`

因此，chart 侧的命名模板接口清单应由对应 chart 自己维护；`atdtool` 负责把 Helm 渲染引擎跑起来，而不是提供一套固定业务 helper。

### 5.3 输出模板接口

`atdtool template` 当前会把 chart 文件区中符合以下条件的文件当作“可输出模板”：

- 文件名以 `.tpl` 结尾
- 文件名以 `*.template` 结尾（按当前实现判断）

这类文件通常用于：

- 配置文件输出，如 `cfg/*.yaml.tpl`
- 脚本输出，如 `bin/*.sh.tpl`、`bin/*.bat.tpl`

输出规则：

1. 去掉模板后缀 `.tpl`
2. 如果当前实例存在 `bus_addr`，则在文件名末尾追加 `_<bus_addr>` 后缀
3. 将渲染结果写入目标输出目录对应位置

例如：

- 输入模板：`cfg/example.yaml.tpl`
- 实例地址：`1.2.65.3`
- 输出文件：`cfg/example_1.2.65.3.yaml`

### 5.4 Chart 侧需要对齐的 values 接口

从 chart 作者视角，最重要的输入接口是：

- `global.yaml`：公共默认值
- `<service>.yaml`：服务级覆盖值
- `modules/*.yaml`：按模块名注入的可选能力默认值
- `non_cloud_native/deploy.yaml`：非云原生实例清单（`template` 模式）
- `--set`：命令行临时覆盖值

这些接口的优先级与行为细节见：

- [`../usage/values-and-overrides.md`](../usage/values-and-overrides.md)

## 6. 推荐实践

1. 配置类模板优先依赖 `.Values`，不要默认依赖 `.Release` 或 `.Capabilities`
2. 引用模块配置时始终加存在性保护，例如 `{{- if .Values.logging }}`
3. 新增 chart 自定义模板接口时，优先在 `_*.tpl` 中定义，再在 `cfg/*.tpl` 或 `bin/*.tpl` 中通过 `include` 复用
4. 如果模板能力本身来自 Helm / Sprig，请直接链接官方文档，不要在本仓库复制一份“半官方说明”

## 相关阅读

- [`../usage/template.md`](../usage/template.md)
- [`../usage/values-and-overrides.md`](../usage/values-and-overrides.md)
