# atdtool

atframework deploy tool , 用于atframework的部署工具。

支持云上和云原生模式。

[![Build](https://github.com/atframework/atdtool/actions/workflows/main.yml/badge.svg)](https://github.com/atframework/atdtool/actions/workflows/main.yml)
![GitHub License](https://img.shields.io/github/license/atframework/atdtool)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/atframework/atdtool/total)
![GitHub Release](https://img.shields.io/github/v/release/atframework/atdtool)
![GitHub Downloads (all assets, latest release)](https://img.shields.io/github/downloads/atframework/atdtool/latest/total)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/atframework/atdtool)
![GitHub forks](https://img.shields.io/github/forks/atframework/atdtool)
![GitHub Repo stars](https://img.shields.io/github/stars/atframework/atdtool)

## 快速开始

详细命令说明请参考 `atdtool --help`，也可以直接查看下方文档索引。

| 命令                   | 说明                                                                 |
| :--------------------- | :------------------------------------------------------------------- |
| `atdtool version`      | 查看 `atdtool` 版本信息                                              |
| `atdtool merge-values` | 针对**单个 chart** 合并 `values.yaml`、配置组目录和命令行覆盖项      |
| `atdtool template`     | 针对**实例清单** 渲染配置模板，输出每个实例对应的配置与脚本          |
| `atdtool guid`         | 生成唯一 ID（雪花算法）                                              |
| `atdtool watch`        | 监听文件变化并执行相关命令                                           |

## 文档索引

- 使用说明
  - [`docs/usage/merge-values.md`](docs/usage/merge-values.md)
  - [`docs/usage/template.md`](docs/usage/template.md)
  - [`docs/usage/values-and-overrides.md`](docs/usage/values-and-overrides.md)
  - [`docs/usage/modules.md`](docs/usage/modules.md)
- 模板运行时参考
  - [`docs/reference/template-runtime.md`](docs/reference/template-runtime.md)
- 结构文档
  - [`docs/structure/project-structure.md`](docs/structure/project-structure.md)
  - [`docs/structure/chart-values-layout.md`](docs/structure/chart-values-layout.md)
  - [`docs/structure/render-flow.md`](docs/structure/render-flow.md)

## 配置来源与作用域

`atdtool` 当前会从以下来源构建最终 `.Values`：

1. chart 自带的 `values.yaml`
2. 配置组目录中的 `global.yaml`
3. 配置组目录中的 **charts 同名 yaml**（优先取 chart 的 `type_name`，其次 `func_name`，最后是 chart 目录名）
4. 配置组目录中的 `modules/*.yaml`
5. 命令行 `--set`
6. `template` 模式下额外注入的运行时变量（如 `world_id`、`zone_id`、`instance_id`、`bus_addr`）

作用域约定如下：

- `global.yaml`：为所有进程 / chart 提供公共默认值
- `<chart-name>.yaml`：只作用于对应 chart（或对应 `type_name` / `func_name`）
- `modules/*.yaml`：以模块名为 key 注入，例如 `modules/logging.yaml` 会合并到 `.Values.logging`
- chart 自带 `values.yaml`：当前 chart 的默认值
- `--set`：命令行临时覆盖

## 同 key 覆盖关系

对于同一个 key，当前代码的真实优先级从高到低为：

1. `template` 模式下无条件注入的 `type_id`（来自 `deploy.yaml`，不可被 `--set` 覆盖）
2. `--set`
3. `template` 模式下按实例注入的运行时值（`world_id`、`zone_id`、`instance_id`、`bus_addr`、`atdtool_running_platform`）
4. 后出现配置组路径中的 charts 同名 yaml
5. 先出现配置组路径中的 charts 同名 yaml
6. chart 自带 `values.yaml`
7. 后出现配置组路径中的 `global.yaml`
8. 先出现配置组路径中的 `global.yaml`
9. 后出现配置组路径中的**已启用**模块配置
10. 先出现配置组路径中的**已启用**模块配置

例如：

```bash
atdtool merge-values ./charts/example -p ./values/default,./values/dev -s log_level=DEBUG
```

需要特别注意：

- `global.yaml` 并不会覆盖 chart 自带 `values.yaml` 的同名 key，它更适合作为“公共默认层”。
- `modules/*.yaml` 更偏向“按需补齐层”：如果更高优先级来源已经写入同名 key，则模块不会再覆盖它。
- `type_id` 由 `deploy.yaml` 中的 `instance_type_id` 无条件注入，`--set` 无法覆盖。而 `world_id`、`zone_id` 等运行时值可通过 `--set global.world_id` 等覆盖。

## Modules 使用约定

- 模块配置需要放在 `modules` 目录下
- 模块文件内容会自动挂到 `.Values.<模块名>` 下
- 模块是否生效，取决于：
  - 更高优先级来源里是否显式设置 `<module>.enabled`
  - 模块文件自身是否声明 `enabled: true`

例如：

```yaml
# values/default/modules/etcd.yaml
enabled: true
endpoints:
  - 127.0.0.1:2379
```

```yaml
# values/dev/example.yaml
logging:
  enabled: false
```

模板引用模块时需要做好保护：

```gotemplate
{{- if .Values.etcd }}
...
{{- end }}
```

更多细节见 [`docs/usage/modules.md`](docs/usage/modules.md)。

## 覆盖配置的写法建议

- 只覆盖需要修改的 key，不要整段复制完整配置。
- 修改模块内部配置时，**不要再额外套一层模块名**。
- 修改进程自身配置时，**不要再额外套一层进程名**。

示例：

```yaml
# 错误示例：重复包了一层模块名
ds:
  pre_alloc_ds_count: 1
  disabled_pre_alloc_alias: []
```

```yaml
# 正确示例：直接写模块内部 key
pre_alloc_ds_count: 1
disabled_pre_alloc_alias: []
```

## `merge-values` 与 `template` 的区别

- `merge-values`
  - 处理对象是**单个 chart**
  - 输出合并后的 `values.yaml`
  - 适合检查某个 chart 在多组 values 下的最终取值
- `template`
  - 处理对象是**chart 根目录**（例如 `./charts`）
  - 依赖 `values` 路径中的 `non_cloud_native/deploy.yaml`
  - 按实例展开并输出每个实例的配置文件和脚本
  - 当前实现中 `-o/--output` 为必填项

更多细节见：

- [`docs/usage/merge-values.md`](docs/usage/merge-values.md)
- [`docs/usage/template.md`](docs/usage/template.md)

## 模板运行时与 Helm 原生能力

`atdtool` 在 Helm 模板能力之上，额外注入了部分运行时值；chart 侧的命名模板与输出模板约定见参考文档。

- `atdtool` 额外变量、模板上下文边界、chart 侧命名模板与输出模板约定：见 [`docs/reference/template-runtime.md`](docs/reference/template-runtime.md)
- Helm 内置对象：<https://helm.sh/docs/chart_template_guide/builtin_objects/>
- Helm 变量：<https://helm.sh/docs/chart_template_guide/variables/>
- Helm 函数与管道：<https://helm.sh/docs/chart_template_guide/functions_and_pipelines/>
- Helm 函数清单：<https://helm.sh/docs/chart_template_guide/function_list/>
- Helm 命名模板：<https://helm.sh/docs/chart_template_guide/named_templates/>
