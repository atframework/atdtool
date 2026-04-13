# atdtool 执行计划

> 说明：本文件仅用于规划后续工作，当前阶段**不执行代码修改**。待你确认本计划无误后，再按计划逐项实施。

## 1. 背景与目标

本工程 `atdtool` 是一个基于 Helm 库实现的配置合并与模板渲染工具，用于打通云上 / 云原生与云下 / 非云原生两类部署场景。根据当前代码与 `D:\workspace\git\github\atsf4g-go\install\cloud-native` 目录分析，后续工作的重点应集中在两大方向：

1. **补充各类使用文档和结构文档**，让使用者能清晰理解命令入口、配置目录结构、覆盖优先级、模块启停规则，以及云原生 / 非云原生两种模式下的数据流。
2. **补全单元测试**，重点覆盖 values 合并、配置组 path 的优先级、`global` 与 `module` 的覆盖关系、模块启停、非云原生实例展开、模板渲染输出等核心行为。

## 2. 现状分析

## 2.1 `atdtool` 当前代码结构分析

结合现有代码，核心链路如下：

- `cmd/atdtool/atdtool.go`
  - 定义 CLI 根命令。
  - 暴露 `template`、`merge-values`、`guid`、`watch`、`exec` 等子命令。
- `cli/values/options.go`
  - 负责解析 `--values/-p` 与 `--set/-s`。
  - `MergePaths()` 负责路径归一化（绝对路径、相对路径、`~`）。
  - `MergeValues()` 负责将命令行 `--set` 解析成嵌套 map。
- `internal/pkg/util/chart.go`
  - `MergeChartValues()` 是配置合并核心：
    1. 按 path 顺序读取 `global.yaml`
    2. 按 chart 名 / `type_name` / `func_name` 读取服务级同名 yaml（如 `lobbysvr.yaml`）
    3. 叠加 chart 自带 `values.yaml`
    4. 叠加 `global` 值
    5. 若为非云原生模式，再叠加 `RenderValue`
    6. 最后叠加命令行 `--set`
    7. 再执行 `mergeEnabledModuleValues()` 处理模块配置
  - `mergeEnabledModuleValues()` 按 `modules/*.yaml` 扫描模块，并根据默认启用状态、服务侧启停开关决定是否注入。
  - 按当前代码真实语义（`chartutil.CoalesceTables` 的 `dest` 优先原则），需要在文档和测试中明确区分：
    - 服务级同名 yaml 对同 key 优先于 `global.yaml`
    - chart 自带 `values.yaml` 对同 key 也优先于 `global.yaml`
    - `modules/*.yaml` 更偏向“按需补齐与默认注入层”，优先级低于已经合入的服务级 / chart 默认 / 命令行值
- `internal/pkg/noncloudnative/noncloudnative.go`
  - `LoadConfig()` 递归查找 `deploy.yaml` 并加载。
  - 多 path 下的非云原生配置目前表现为：**后遇到的 `deploy.yaml` 会整体替换前者**，不是字段级 merge。
  - `ToRenderValues()` 将 `bus_addr` 转换成 `world_id / zone_id / instance_id` 等模板变量。
- `cmd/atdtool/template.go`
  - 通过 `noncloudnative.LoadConfig()` 获取部署实例信息。
  - 针对每个实例组合 `bus_addr` 与运行时 values。
  - 调用 `util.MergeChartValues()` 后渲染 chart 模板。
  - 输出目标目录下的实例化配置文件。

## 2.2 `cloud-native` 目录内容分析

`D:\workspace\git\github\atsf4g-go\install\cloud-native` 目录提供了真实的样例结构，可直接作为文档示例来源与测试夹具参考：

- `charts/`
  - `libapp/`：library chart，包含 `_atapp.*.tpl`、`_vector.yaml.tpl` 等公共模板。
  - `lobbysvr/`：application chart，依赖 `libapp`，包含 `values.yaml`、Kubernetes 资源模板、`cfg/*.tpl`、`bin/*.tpl`。
- `values/`
  - `default/global.yaml`
  - `default/modules/*.yaml`：如 `autoscaling.yaml`、`redis.yaml`、`resources.yaml`、`vector.yaml`
  - `default/non_cloud_native/deploy.yaml`
  - `dev/global.yaml`
  - `dev/modules/vector.yaml`
  - 体现了“默认层 + 环境层 + 模块层 + 非云原生部署层”的典型组织方式。
  - 当前 `cloud-native` 示例更偏向展示 `global.yaml` / `modules` / `deploy.yaml`；而代码还支持 `default/<chart-name>.yaml`、`dev/<chart-name>.yaml` 这类服务级同名文件，后续文档与测试夹具需显式补齐这部分说明与样例。
- `docker/` 与 `images/`
  - 对应容器启动脚本与镜像目录，适合在结构文档中说明其与 chart / values 的协作关系。

## 2.3 当前缺口判断

当前仓库中只有如下测试：

- `pkg/compress/compress_test.go`
- `pkg/snowflake/snowflake_test.go`

这意味着：

- **主流程没有测试**：`values` 解析、path 合并、模块合并、非云原生实例展开、模板渲染等均缺少保障。
- **文档不足**：README 已有基础说明，但尚未系统描述：
  - 目录结构
  - 云原生 / 非云原生模式差异
  - `global.yaml` / charts 同名 yaml / `modules/*.yaml` / `--set` 的覆盖关系
  - `atdtool` 额外注入的模板变量、样例 chart 自定义模板接口（`define` / `include`）及其功能说明
  - Helm 内置对象 / 函数与 `atdtool` 自定义内容之间的边界，以及应链接的 Helm 官方文档入口
  - 真实示例目录如何映射到工具输入输出
  - README 中“优先级说明”和代码真实行为之间可能存在的偏差，需通过测试与文档统一口径

## 3. 本次计划的执行范围

待确认后，本次执行将聚焦以下范围：

1. **补充使用文档**
2. **补充结构文档**
3. **补全单元测试与测试夹具**
4. **验证测试通过，并确保文档与代码行为一致**

本次**暂不计划**：

- 直接调整业务模板内容
- 改造配置语义或覆盖规则
- 改动 `cloud-native` 外部仓库中的内容

如在测试过程中暴露出明确缺陷，会先记录为问题点，并在你确认后决定是否纳入修复范围。

## 4. 分阶段执行方案

## 4.1 阶段一：补充文档

### 文档阶段目标

建立“看文档即可上手、看结构即可理解覆盖关系”的文档体系。

### 文档交付物

建议补充 / 调整以下文档（文件名可在执行时微调）：

- `README.md`
  - 补充快速开始导航
  - 补充文档索引
  - 保留命令总览，避免 README 过重
- `docs/usage/merge-values.md`
  - 说明 `merge-values` 输入、输出、优先级、典型命令
- `docs/usage/template.md`
  - 说明 `template` 的输入结构、输出目录、非云原生实例展开逻辑
- `docs/usage/values-and-overrides.md`
  - 专门说明配置来源与优先级：`chart values`、`global.yaml`、charts 同名 yaml、`modules/*.yaml`、`--set`
  - 说明多 path（如 `default,dev`）下 `global.yaml`、同名 yaml、modules 的覆盖顺序与关系
- `docs/usage/modules.md`
  - 说明模块默认启用、服务侧禁用、模块命名约定、模板保护写法
- `docs/reference/template-runtime.md`
  - 汇总 `atdtool` 额外注入 / 覆盖的模板变量（如 `.Values.world_id`、`.Values.zone_id`、`.Values.instance_id`、`.Values.bus_addr`、`.Values.atdtool_running_platform` 等）
  - 汇总样例 chart 中的自定义模板接口（如 `atapp.yaml`、`atapp.logic.yaml`、`vector.yaml`、`libapp.util.merge`、`libapp.logicID`、`libapp.busAddr` 等）及功能说明
  - 对 Helm 内置对象、函数、Sprig 函数等不重复造文档，统一补充官方链接
- `docs/structure/project-structure.md`
  - 说明 `cmd/`、`cli/`、`internal/pkg/util`、`internal/pkg/noncloudnative`、`pkg/` 的职责划分
- `docs/structure/cloud-native-layout.md`
  - 说明 `install/cloud-native` 样例目录中 `charts/`、`values/`、`docker/`、`images/` 的角色
- `docs/structure/render-flow.md`
  - 用步骤图式文字说明从 `--values` / `--set` 到最终渲染文件的完整链路

### 文档重点内容

文档中会明确以下关键点：

1. **配置优先级**（按代码实际行为描述，而不是按经验猜测）
2. **多 path 叠加规则**（例如 `default -> dev`）
3. **模块配置的注入条件**
4. **服务配置文件命名如何由 chart 名、`type_name`、`func_name` 决定**
5. **非云原生 `deploy.yaml` 如何驱动多实例渲染**
6. **模板内访问模块配置时为何需要 `if .Values.xxx` 保护**
7. **`atdtool` 额外注入 / 改写的模板变量清单，以及它们在云上 / 云下模式下的来源**
8. **`cloud-native` 样例中自定义模板接口（`define` / `include`）的清单、输入与用途**
9. **Helm / Sprig 原生对象与函数的官方文档链接，避免与 `atdtool` 自定义内容混淆**
10. **charts 同名 yaml 与 `global.yaml`、`modules/*.yaml` 作用于同一 key 时的真实合并关系**

## 4.2 阶段二：搭建测试夹具（testdata）

### 测试夹具阶段目标

构建一组**最小但覆盖关键分支**的测试样例，避免测试直接耦合外部仓库路径。

### 测试夹具交付物

在仓库内新增若干 `testdata/` 目录（具体位置执行时根据包路径确定），内容包括：

- 简化版 chart 样例
  - 最小 `Chart.yaml`
  - 最小 `values.yaml`
  - 1~2 个 `.tpl` 模板
- 多组 values path 样例
  - `default/global.yaml`
  - `default/<service>.yaml`
  - `default/modules/*.yaml`
  - `dev/global.yaml`
  - `dev/<service>.yaml`
  - `dev/modules/*.yaml`
  - `non_cloud_native/deploy.yaml`
- 必要时抽取 `cloud-native` 中的典型配置模式，做成更小的测试样例
- 针对同一 key 同时出现在 `global.yaml`、charts 同名 yaml、`modules/*.yaml` 的场景，设计专门夹具验证真实覆盖关系

### 原则

- **不依赖外部目录运行测试**，避免 CI 和本地环境路径差异。
- **测试数据尽量小**，但要保留真实结构特征。
- **优先保留语义，不保留冗长业务字段**，避免测试臃肿。
- **必须包含 charts 同名 yaml 场景**，不能只验证 `global.yaml` 与 modules。

## 4.3 阶段三：补全单元测试

### 单元测试阶段目标

围绕核心行为建立测试矩阵，优先覆盖“最容易回归、最难靠肉眼发现”的逻辑。

### 计划补充的测试文件

建议新增以下测试（名称可在执行时微调）：

- `cli/values/options_test.go`
- `internal/pkg/util/chart_test.go`
- `internal/pkg/noncloudnative/deploy_test.go`
- `internal/pkg/noncloudnative/noncloudnative_test.go`
- `cmd/atdtool/template_test.go`
- 如有必要：`cmd/atdtool/merge_values_test.go`

### 测试矩阵

| 目标模块 | 关键测试点 | 说明 |
| --- | --- | --- |
| `cli/values.Options.MergeValues` | 标量、嵌套对象、数组、字符串、非法 `--set` | 覆盖 `strvals.ParseInto` 的常见输入 |
| `cli/values.Options.MergePaths` | 绝对路径、相对路径、`~` 路径、不存在路径 | 验证路径归一化与错误返回 |
| `util.MergeChartValues` | chart 默认值加载 | 验证 `values.yaml` 能作为基础值 |
| `util.MergeChartValues` | charts 同名 yaml 加载 | 验证按 chart 名 / `type_name` / `func_name` 读取服务级同名文件 |
| `util.MergeChartValues` | `global.yaml` 与 chart 默认值的实际关系 | 验证当前代码中 chart 默认值对同 key 优先，`global.yaml` 主要补齐缺失项 |
| `util.MergeChartValues` | charts 同名 yaml 与 `global.yaml` 的关系 | 验证服务级同名 yaml 对同 key 优先 |
| `util.MergeChartValues` | 服务 yaml 覆盖 chart 默认值 | 验证服务级覆盖 chart 默认值 |
| `util.MergeChartValues` | 多 path 下 `default -> dev` 的覆盖顺序 | 验证后 path 优先 |
| `util.MergeChartValues` | 多 path 下 charts 同名 yaml 的覆盖顺序 | 验证 `default/<service>.yaml` 与 `dev/<service>.yaml` 的优先级 |
| `util.MergeChartValues` | `type_name` / `func_name` 对服务文件命名的影响 | 覆盖 chart 名别名场景 |
| `util.MergeChartValues` | `--set` 的最高优先级 | 验证命令行覆盖最终值 |
| `mergeEnabledModuleValues` | 模块默认启用时自动注入 | 覆盖 `enabled: true` |
| `mergeEnabledModuleValues` | 模块默认关闭时不注入 | 覆盖 `enabled: false` |
| `mergeEnabledModuleValues` | 服务侧显式 `enabled: false` 时禁用模块 | 验证服务配置可关闭默认模块 |
| `mergeEnabledModuleValues` | 多 path 下 module 覆盖关系 | 覆盖 `default/modules` 与 `dev/modules` |
| `mergeEnabledModuleValues` | 同一模块 key 在 `global.yaml` / 服务同名 yaml / module 中同时出现 | 验证模块层作为低优先级补齐层时的真实行为 |
| `mergeEnabledModuleValues` | 模块值合并后不要求再次套模块名 | 对应 README 中的使用约束 |
| `noncloudnative.parseBusAddr` | 合法 bus addr / 非法 bus addr | 覆盖格式与类型错误 |
| `noncloudnative.LoadConfig` | 递归查找 `deploy.yaml` | 验证目录扫描行为 |
| `noncloudnative.LoadConfig` | 多 path 下后者整体替换前者 | 固化当前真实语义 |
| `noncloudnative.ToRenderValues` | `world_id / zone_id / instance_id / bus_addr` 输出 | 验证模板输入值 |
| `templateOptions.run` | 多实例渲染输出 | 覆盖 `instance_count`、`start_instance_id` |
| `templateOptions.run` | `global.world_id` / `global.zone_id` 支持 `string/int/uint` 覆盖 | 覆盖当前类型转换逻辑 |
| `templateOptions.run` | 未提供 `outPath` 报错 | 覆盖失败分支 |
| `renderTemplate` / `render` | 仅输出 `.tpl` 模板、输出文件命名带 bus addr 后缀 | 覆盖渲染输出行为 |

## 4.4 阶段四：校验与回归

### 回归阶段目标

保证文档与代码行为一致，测试可以稳定运行。

### 计划执行的验证动作

确认计划后，执行阶段会至少完成：

- 运行全量单元测试
- 按新增测试检查关键路径是否稳定
- 对照文档逐项确认示例命令、优先级与实际行为一致

### 验收标准

满足以下条件视为本阶段完成：

1. `README.md` 与新增 `docs/` 文档能覆盖日常使用与结构说明
2. 文档中明确列出 `atdtool` 额外模板变量、自定义模板接口清单，并为 Helm / Sprig 原生对象与函数补充官方链接
3. 至少为核心链路补齐单元测试：`values`、`util.MergeChartValues`、`module merge`、`noncloudnative`、`template`
4. 测试覆盖多 path、`global.yaml`、charts 同名 yaml、module、`--set` 的组合关系
5. 针对同 key 同时出现在 `global.yaml`、charts 同名 yaml、`modules/*.yaml` 的场景，有明确测试固化当前行为
6. 测试数据可独立在当前仓库内运行，不依赖外部目录
7. 文档描述与当前代码语义一致，尤其是同名 yaml / global / module 的优先级，以及非云原生配置替换规则

## 5. 实施顺序建议

建议按如下顺序实施：

1. 先补结构文档草稿，统一术语
2. 再创建测试夹具并补核心单元测试
3. 根据测试验证结果修正文档中的优先级与边界说明
4. 最后补 README 导航与示例命令

这样做的原因是：

- 测试能先把“真实语义”钉住，避免文档写成理想行为
- README 放最后补，能直接引用已落地的 `docs/` 内容

## 6. 风险与注意事项

1. **当前代码行为与 README 文字描述可能存在细微偏差**
   - 实施时应以测试验证后的代码语义为准。
2. **`noncloudnative.LoadConfig()` 当前是“整体替换”而非深度合并**
   - 文档必须写清楚，避免误解。
3. **Windows 与类 Unix 的路径行为略有差异**
   - 路径测试要避免写成平台强绑定。
4. **模板渲染依赖文件系统结构**
   - 需要用最小 chart 夹具保证测试稳定。
5. **Helm 官方对象 / 函数并非全部由 `atdtool` 主动注入**：文档中要区分“`atdtool` 补充的模板变量”和“Helm 引擎 / Sprig 提供的能力”，避免误导使用者。

## 7. 预期修改清单（确认后执行）

确认后，预计会涉及以下类型的文件：

- 文档文件
  - `README.md`
  - `docs/**/*.md`
- 测试文件
  - `cli/values/options_test.go`
  - `internal/pkg/util/chart_test.go`
  - `internal/pkg/noncloudnative/*_test.go`
  - `cmd/atdtool/*_test.go`
- 测试数据
  - `**/testdata/**`

## 8. 待你确认

如果你确认本 `Plan.md` 的范围、阶段划分和产出形式没有问题，我下一步将按此计划开始实施：

1. 先补文档框架与内容
2. 再补测试夹具与单元测试
3. 最后运行验证并整理结果

在你确认之前，我不会执行这些代码与文档改动。
