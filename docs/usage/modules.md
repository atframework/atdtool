# Modules 使用说明

`atdtool` 支持从 `values/<group>/modules/*.yaml` 中加载模块配置，并按模块名合并到最终 `.Values` 中。

## 目录结构

典型写法：

```text
values/
  default/
    modules/
      logging.yaml
      cache.yaml
  dev/
    modules/
      logging.yaml
```

模块文件名会成为最终 values 的 key：

- `modules/logging.yaml` -> `.Values.logging`
- `modules/cache.yaml` -> `.Values.cache`

## 文件内容写法

模块文件内部**不要再额外套一层模块名**。

正确示例：

```yaml
# modules/logging.yaml
enabled: true
log_path: /data/log
```

错误示例：

```yaml
# modules/logging.yaml
logging:
  enabled: true
  log_path: /data/log
```

原因是当前实现会自动把整个文件挂到 `.Values.logging` 下；如果文件里再包一层，会变成 `.Values.logging.logging`。

## 启用与禁用规则

### 方式一：模块文件默认启用

```yaml
enabled: true
```

如果更高优先级来源没有明确关闭它，那么模块会被自动注入。

### 方式二：由服务级或公共配置显式启用

如果模块文件没有 `enabled: true`，也可以通过更高优先级来源显式启用：

```yaml
logging:
  enabled: true
```

### 方式三：由服务级配置显式关闭

```yaml
logging:
  enabled: false
```

一旦更高优先级来源显式关闭模块，模块文件就不会再被注入。

## 与 global.yaml / 同名 yaml 的关系

模块层是低优先级补齐层，不是强覆盖层。

也就是说，当同一模块 key 同时出现在：

- `global.yaml`
- charts 同名 yaml
- `modules/<module>.yaml`

最终规则是：

1. 服务级同名 yaml 优先
2. chart 默认值次之
3. `global.yaml` 再次之
4. 模块文件最后补齐缺失 key

例如：

```yaml
# global.yaml
logging:
  enabled: true
  log_path: /global/log
```

```yaml
# example.yaml
logging:
  log_path: /service/log
```

```yaml
# modules/logging.yaml
enabled: true
log_path: /module/log
sinks:
  file: true
```

最终结果是：

```yaml
logging:
  enabled: true
  log_path: /service/log
  sinks:
    file: true
```

说明：

- `log_path` 由服务级同名 yaml 决定
- `sinks.file` 来自模块文件补齐

## 模板中的推荐写法

由于模块可能未启用，模板中应始终做存在性保护：

```gotemplate
{{- if .Values.logging }}
log_path: {{ .Values.logging.log_path }}
{{- end }}
```

访问内部字段时也建议建立在模块存在的前提下：

```gotemplate
{{- if and .Values.logging .Values.logging.enabled }}
log_path: {{ .Values.logging.log_path }}
{{- end }}
```

## 多 path 场景

当同时传入多个 values 路径时：

```bash
-p ./values/default,./values/dev
```

模块文件也遵循“后 path 优先”的规则：

- `dev/modules/logging.yaml` 会覆盖 `default/modules/logging.yaml` 中的同名 key
- 但它仍然不会覆盖更高优先级来源（服务级同名 yaml / chart 默认值 / `--set`）

## 典型用途

适合放入模块层的内容包括：

- 可选能力的默认配置（如 `logging`、`cache`、`feature-flags`）
- 多 chart 共享但并非所有实例都启用的配置
- 希望由 `enabled` 开关统一控制的配置块

## 相关阅读

- [`values-and-overrides.md`](values-and-overrides.md)
- [`../reference/template-runtime.md`](../reference/template-runtime.md)
