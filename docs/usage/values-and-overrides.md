# Values 与覆盖关系说明

本文档说明 `atdtool` 如何从多个来源构建最终 `.Values`，以及 `global.yaml`、charts 同名 yaml、`modules/*.yaml` 之间的真实关系。

## 配置来源清单

对于一个 chart，`atdtool` 可能会读取以下内容：

1. chart 自带 `values.yaml`
2. `values/<group>/global.yaml`
3. `values/<group>/<service>.yaml`
4. `values/<group>/modules/*.yaml`
5. 命令行 `--set`
6. `template` 模式下的实例运行时值

## 服务级 yaml 文件名如何确定

实际读取哪个 `<service>.yaml`，顺序如下：

1. 优先取 chart 默认值中的 `type_name`
2. 如果没有 `type_name`，取 `func_name`
3. 如果都没有，则取 chart 本身的名称

因此，服务级 yaml 的文件名不一定等于 chart 目录名。

## 多 path 的基本规则

命令支持一次传多个 values 路径，例如：

```bash
-p ./values/default,./values/dev
```

当前代码里：

- 后面的 path 优先级高于前面的 path
- 但优先级是按**同类来源**分别计算的

也就是说：

- `dev/global.yaml` 会覆盖 `default/global.yaml`
- `dev/example.yaml` 会覆盖 `default/example.yaml`
- `dev/modules/logging.yaml` 会覆盖 `default/modules/logging.yaml`

## 同 key 的真实优先级

对于同一个 key，当前实现的真实优先级从高到低为：

1. `template` 模式下无条件注入的 `type_id`（来自 `deploy.yaml`，不可被 `--set` 覆盖）
2. `--set`
3. `template` 模式下的实例运行时值（`world_id`、`zone_id`、`instance_id`、`bus_addr`、`atdtool_running_platform`）
4. 后 path 的 charts 同名 yaml
5. 前 path 的 charts 同名 yaml
6. chart 自带 `values.yaml`
7. 后 path 的 `global.yaml`
8. 前 path 的 `global.yaml`
9. 后 path 的已启用模块配置
10. 前 path 的已启用模块配置

这意味着：

- `global.yaml` 不会覆盖 chart 默认值
- 模块配置不会覆盖服务级同名 yaml
- 模块更像一个“默认补全层”- `type_id` 由 `deploy.yaml` 中的 `instance_type_id` 无条件注入，`--set` 无法覆盖。而 `world_id`、`zone_id` 等运行时值可通过 `--set global.world_id` 等覆盖

## 深度合并行为

合并使用的是 Helm 的 `chartutil.CoalesceTables`，它的语义是：

- `dest` 优先于 `src`
- map / object 按 key 深度合并
- 已存在于高优先级来源中的 key，不会被低优先级来源覆盖

例如：

### chart 默认值

```yaml
cache:
  enabled: true
  size: 100
  mode: chart
```

### global.yaml

```yaml
cache:
  size: 50
  timeout: 30s
```

### example.yaml

```yaml
cache:
  mode: service
```

最终结果会是：

```yaml
cache:
  enabled: true
  size: 100
  mode: service
  timeout: 30s
```

说明：

- `size` 来自 chart 默认值，因为 chart 默认值优先于 `global.yaml`
- `mode` 来自服务级同名 yaml，因为服务级优先于 chart 默认值
- `timeout` 来自 `global.yaml`，因为更高层没有定义它

## modules 的关系

模块文件会自动挂到 `.Values.<模块名>` 下。

例如：

```yaml
# modules/logging.yaml
enabled: true
log_path: /data/log
sinks:
  file: true
```

最终会变成：

```yaml
logging:
  enabled: true
  log_path: /data/log
  sinks:
    file: true
```

如果更高优先级来源已有：

```yaml
logging:
  enabled: true
  log_path: /custom/log
```

那么模块文件只会补上缺失项：

```yaml
logging:
  enabled: true
  log_path: /custom/log
  sinks:
    file: true
```

如果更高优先级来源显式设置：

```yaml
vector:
  enabled: false
```

则模块不会被注入。

## `--set` 的差异

### 在 `merge-values` 中

`--set` 会原样写入最终 `.Values`。

例如：

```bash
-s log_level=DEBUG
-s vector.enabled=false
```

会直接覆盖：

- `.Values.log_level`
- `.Values.logging.enabled`

### 在 `template` 中

当前实现会额外识别：

- `global.xxx`
- `<实例名>.xxx`

并在渲染前把它们扁平化到实例级顶层值中。

因此：

```bash
-s global.log_level=DEBUG
```

在 `template` 命令里能影响每个实例的顶层 `log_level`；但在 `merge-values` 命令里，这只是普通的 `.Values.global.log_level`。

## 推荐组织方式

推荐 values 目录按下面的方式组织：

```text
values/
  default/
    global.yaml
    example.yaml
    modules/
      logging.yaml
      cache.yaml
    non_cloud_native/
      deploy.yaml
  dev/
    global.yaml
    example.yaml
    modules/
      logging.yaml
```

这种结构可以同时覆盖：

- 公共默认值
- 服务级差异
- 环境差异
- 模块默认值
- 非云原生实例部署信息

## 相关阅读

- [`merge-values.md`](merge-values.md)
- [`template.md`](template.md)
- [`modules.md`](modules.md)
