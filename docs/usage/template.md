# template 使用说明

`atdtool template` 用于根据实例部署信息和 values 配置，渲染出每个实例对应的配置文件与脚本。

它不是 Helm 的 `helm template` 替代品，而是针对本工程“配置文件 / 启停脚本生成”场景的本地渲染入口。

## 与 merge-values 的区别

- `merge-values`：面向**单个 chart**，输出合并后的 values
- `template`：面向**实例集合**，输出每个实例的渲染结果

## 输入约定

### `CHART` 参数

`template` 命令的 `CHART` 参数应当是**chart 根目录**，而不是单个 chart 目录。

例如应传：

- `./charts`

而不是：

- `./charts/example`

原因是当前实现会读取部署清单中的 `chart_name`，再拼出：

- `<CHART>/<chart_name>`

去寻找实际 chart。

### `--values` 路径

`--values` 需要至少包含一套能被递归扫描到 `deploy.yaml` 的目录，例如：

- `values/default/non_cloud_native/deploy.yaml`

同一组 values 中，还可以包含：

- `global.yaml`
- `<chart-name>.yaml`
- `modules/*.yaml`

### `--output`

当前实现中 `-o, --output` 是**必填项**。如果不传，命令会直接报错。

## 实例展开流程

1. 读取 `--values` 指定的多个配置组路径
2. 递归查找并加载 `deploy.yaml`
3. 遍历 `proc_desc` 中的每个实例定义
4. 按 `world_id.zone_id.type_id.instance_id` 生成 `bus_addr`
5. 将 values 与运行时值合并
6. 渲染 chart 中的 `.tpl` 文件并输出到目标目录

## 运行时额外注入的值

在 `template` 模式下，除了常规 `.Values` 之外，还会额外注入实例相关值：

| 值 | 说明 |
| --- | --- |
| `.Values.world_id` | 当前实例所属 world |
| `.Values.zone_id` | 当前实例所属 zone |
| `.Values.instance_id` | 当前实例号 |
| `.Values.bus_addr` | 当前实例的 bus 地址 |
| `.Values.atdtool_running_platform` | 当前运行平台，例如 `windows` / `linux` |
| `.Values.type_id` | 当前实例的 `instance_type_id` |

更完整的模板接口清单见：

- [`../reference/template-runtime.md`](../reference/template-runtime.md)

## `--set` 的作用方式

`template` 命令对 `--set` 做了额外处理：

- `global.xxx`：会被视为“应用到所有实例的顶层值”
- `<实例名>.xxx`：只应用到对应实例

例如：

```bash
atdtool template ./charts \
  -p ./values/default,./values/dev \
  -o ./target/rendered \
  -s global.log_level=DEBUG \
  -s global.world_id=10 \
  -s example.listen.port=7101
```

其中：

- `global.log_level=DEBUG` 会作用于所有实例的顶层 `log_level`
- `global.world_id=10` 不仅会覆盖 `world_id`，还会参与 `bus_addr` 计算
- `example.listen.port=7101` 只作用于 `example` 实例

## 输出文件规则

当前实现会渲染 chart 中的普通 `.tpl` 文件，典型位置包括：

- `cfg/*.yaml.tpl`
- `bin/*.sh.tpl`
- `bin/*.bat.tpl`

输出文件名会自动带上实例的 `bus_addr` 后缀。

例如模板文件：

- `cfg/example.yaml.tpl`

当实例地址为 `1.2.65.3` 时，输出文件可能为：

- `cfg/example_1.2.65.3.yaml`

## 非云原生 deploy.yaml 的当前语义

当传入多个 values 路径时，`deploy.yaml` 当前不是字段级 merge，而是：

- **后扫描到的文件整体替换前面的文件**

这点和 `global.yaml`、同名 yaml、modules 的深度合并语义不同，文档和测试都按当前实现解释。

## 注意事项

1. 当前渲染顶层上下文主要依赖 `.Values`；Helm 的 `.Release`、`.Capabilities` 等对象并不会像 `helm template` 那样完整填充。
2. 如果模板依赖大量 Kubernetes 发行期上下文，请优先使用 Helm 标准渲染链路，而不是 `atdtool template`。
3. chart 中真正被输出的是普通文件区的 `.tpl` 文件；`templates/_*.tpl` 更常用于定义可复用片段。

## 相关阅读

- [`merge-values.md`](merge-values.md)
- [`values-and-overrides.md`](values-and-overrides.md)
- [`../structure/render-flow.md`](../structure/render-flow.md)
