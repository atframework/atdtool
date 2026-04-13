# 项目结构说明

本文档描述 `atdtool` 仓库中与配置生成、模板渲染和测试最相关的目录职责。

## 顶层目录

| 路径 | 作用 |
| --- | --- |
| `cmd/` | CLI 命令入口与命令实现 |
| `cli/` | 命令行参数解析与共享选项 |
| `internal/` | 内部实现细节，不作为公共 API 暴露 |
| `pkg/` | 可复用公共包 |
| `cmake/` | CMake / Go 构建辅助脚本 |
| `vendor/` | vendored 第三方依赖 |
| `target/` | 本地构建产物目录 |

## `cmd/atdtool/`

该目录是主命令实现入口。

核心文件：

- `atdtool.go`
  - 创建根命令并注册子命令
- `merge_values.go`
  - 实现 `merge-values` 命令
- `template.go`
  - 实现 `template` 命令
  - 负责实例展开、运行时值注入和模板输出
- `guid.go`
  - 实现雪花 ID 生成命令
- `watch.go`
  - 实现文件监听相关命令
- `exec.go`
  - 实现外部命令执行相关逻辑
- `version.go`
  - 输出版本信息

## `cli/values/`

`cli/values/options.go` 负责解析：

- `--values/-p`
- `--set/-s`

其中：

- `MergePaths()`：做路径归一化与存在性检查
- `MergeValues()`：使用 Helm `strvals` 语法解析命令行覆盖项

## `internal/pkg/util/`

与配置合并和文件输出最相关的目录。

### `chart.go`

是本项目 values 合并的核心实现：

- 读取 chart 自带 `values.yaml`
- 读取 `global.yaml`
- 读取服务级同名 yaml
- 按模块开关注入 `modules/*.yaml`
- 叠加命令行值与非云原生运行时值

### 其他辅助文件

- `file.go`：文件与路径相关工具
- `chart.go` 中的 `mergeEnabledModuleValues()`：模块注入逻辑核心

## `internal/pkg/noncloudnative/`

该目录负责处理非云原生 / 云下部署所需的实例信息。

核心职责：

- 加载 `deploy.yaml`
- 解析实例列表 `proc_desc`
- 计算 `world_id`、`zone_id`、`instance_id`、`bus_addr`
- 将实例信息转换成模板可消费的 `.Values`

关键文件：

- `noncloudnative.go`
- `deploy.go`

## `pkg/`

该目录放置可以独立复用的公共包。

当前已有测试的子包包括：

- `pkg/compress/`
- `pkg/snowflake/`

此外还有：

- `pkg/confparser/yaml/`：YAML 配置读取

## 与业务 chart / values 输入目录的关系

`atdtool` 自身提供的是：

- CLI 命令
- values 合并能力
- 非云原生实例展开能力
- 基于 Helm 引擎的模板渲染能力

它并不强绑定某一个外部样例目录。实际使用时，需要由调用方提供：

- chart 根目录
- 单个 chart 的 `values.yaml`
- 分层组织的 values 目录（如 `global.yaml`、`<service>.yaml`、`modules/*.yaml`）
- 非云原生场景下的 `non_cloud_native/deploy.yaml`

推荐输入结构见：

- [`chart-values-layout.md`](chart-values-layout.md)

## 调试建议

当出现“值不对”或“模板输出不符合预期”时，通常按以下顺序排查最有效：

1. `cli/values/options.go`：确认 `--values` / `--set` 是否被正确解析
2. `internal/pkg/util/chart.go`：确认同名 yaml、`global.yaml`、module 的优先级
3. `internal/pkg/noncloudnative/*.go`：确认实例信息与 `bus_addr` 生成
4. `cmd/atdtool/template.go`：确认最终传给渲染器的值与输出路径
5. 目标 chart 中的 `*.tpl`：确认模板是否对可选模块做了保护

## 相关阅读

- [`render-flow.md`](render-flow.md)
- [`../reference/template-runtime.md`](../reference/template-runtime.md)
