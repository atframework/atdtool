# merge-values 使用说明

`atdtool merge-values` 用于针对**单个 chart** 计算最终 `.Values`，并将结果输出为一个 YAML 文件。

## 适用场景

适合在以下场景使用：

- 验证某个 chart 在多套 values 目录下的最终取值
- 检查 `global.yaml`、charts 同名 yaml、`modules/*.yaml` 和 `--set` 的叠加结果
- 生成一个可落盘的最终 `values.yaml` 供排障或审阅使用

## 输入

命令形态：

- `CHART`：**单个 chart 目录**，例如 `./charts/example`
- `-p, --values`：一个或多个 values 路径，后面的路径优先级更高
- `-s, --set`：命令行覆盖项，优先级最高
- `-o, --output`：输出文件路径；如果给的是目录，会自动写成 `<目录>/values.yaml`

## 服务级同名 yaml 的解析规则

服务级 yaml 文件名不是固定直接取 chart 目录名，而是按以下顺序解析：

1. 先看 chart 默认值里的 `type_name`
2. 如果没有 `type_name`，再看 `func_name`
3. 如果都没有，再退回 chart 目录名 / `Chart.yaml` 中的 chart 名

例如：

- chart 目录叫 `engine`
- `values.yaml` 中 `type_name: example`
- 则会读取 `values/<group>/example.yaml`

如果 chart 目录叫 `logic`，但 `values.yaml` 中写了 `type_name: example`，那么读取的仍然是 `example.yaml`。

## 同 key 合并顺序

对于同一个 key，当前代码的实际优先级从高到低为：

1. `--set`
2. 后出现 values 路径中的 charts 同名 yaml
3. 先出现 values 路径中的 charts 同名 yaml
4. chart 自带 `values.yaml`
5. 后出现 values 路径中的 `global.yaml`
6. 先出现 values 路径中的 `global.yaml`
7. 后出现 values 路径中的**已启用**模块配置
8. 先出现 values 路径中的**已启用**模块配置

要点：

- `global.yaml` 是“公共默认层”，不会覆盖 chart 自带 `values.yaml` 的同名 key
- 模块配置是“按需补齐层”，只会补充更高优先级来源没有定义的 key
- map / object 是深度合并，而不是整段替换

## modules 的合并规则

模块文件放在 `values/<group>/modules/*.yaml` 下，例如：

- `values/default/modules/logging.yaml`
- `values/dev/modules/cache.yaml`

模块文件会自动被挂到 `.Values.<模块名>` 下。

例如 `modules/logging.yaml`：

```yaml
enabled: true
log_path: /data/log
```

会出现在最终值里：

```yaml
logging:
  enabled: true
  log_path: /data/log
```

模块生效规则：

- 如果更高优先级来源里存在 `<module>.enabled: false`，模块不会注入
- 如果更高优先级来源里存在 `<module>.enabled: true`，模块会注入
- 如果更高优先级来源没有显式写 `<module>.enabled`，则以模块文件自身的 `enabled` 为准

## 常用示例

### 合并 default 与 dev 两层配置

```bash
atdtool merge-values ./charts/example \
  -p ./values/default,./values/dev \
  -o ./target/example.values.yaml
```

### 用命令行覆盖最终值

```bash
atdtool merge-values ./charts/example \
  -p ./values/default,./values/dev \
  -s log_level=DEBUG \
  -s logging.enabled=false \
  -o ./target/example.values.yaml
```

### 覆盖数组与嵌套对象

```bash
atdtool merge-values ./charts/example \
  -p ./values/default \
  -s http_client.timeout=15s \
  -s logging.sources[0].name=access \
  -o ./target/example.values.yaml
```

## 输出行为

- 指定 `-o ./result.yaml`：直接写入该文件
- 指定 `-o ./target`：写入 `./target/values.yaml`
- 不指定 `-o`：当前实现会直接覆盖 chart 目录下的 `values.yaml`

因此在日常使用中，建议**总是显式指定 `-o`**，避免误覆盖 chart 默认值。

## 注意事项

1. `merge-values` 不会像 `template` 命令那样注入 `instance_id`、`bus_addr` 等运行时实例值。
2. `--set` 在本命令中是**原样并入 `.Values`**，不会自动把 `global.xxx` 扁平化成顶层值。
3. 如果任一 `--values` 路径不存在，命令会返回错误。

## 相关阅读

- [`values-and-overrides.md`](values-and-overrides.md)
- [`modules.md`](modules.md)
- [`../reference/template-runtime.md`](../reference/template-runtime.md)
