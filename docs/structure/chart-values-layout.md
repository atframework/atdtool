# Chart 与 Values 目录约定

本文档描述 `atdtool` 推荐的输入目录结构，重点说明 chart 根目录、values 分层目录和非云原生部署清单之间的关系。

## 1. 总览

`atdtool` 主要消费两类输入：

1. **chart 目录**：定义模板、默认值和输出文件结构
2. **values 目录**：定义公共配置、服务级配置、模块配置和实例清单

一个通用的组织方式如下：

```text
project/
  charts/
    example/
      Chart.yaml
      values.yaml
      cfg/
        example.yaml.tpl
      bin/
        start.sh.tpl
      templates/
        _helpers.tpl
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

## 2. chart 根目录

### `merge-values` 的输入

`merge-values` 需要传入的是**单个 chart 目录**，例如：

- `./charts/example`

### `template` 的输入

`template` 需要传入的是**chart 根目录**，例如：

- `./charts`

因为当前实现会从 `deploy.yaml` 中读取 `chart_name`，再按下面的方式定位实例对应 chart：

- `<chart-root>/<chart_name>`

## 3. 单个 chart 目录建议结构

一个 chart 至少应包含：

- `Chart.yaml`
- `values.yaml`

常见补充目录包括：

- `cfg/`：配置模板文件，如 `*.yaml.tpl`
- `bin/`：脚本模板文件，如 `*.sh.tpl` / `*.bat.tpl`
- `templates/`：Helm helper、命名模板和依赖模板

### `values.yaml`

作用：

- 提供 chart 默认值
- 可选定义 `type_name` / `func_name`，用于决定服务级同名 yaml 的文件名解析

### `cfg/` / `bin/`

这些目录通常放的是 `atdtool template` 要输出的实际文件模板。

例如：

- `cfg/example.yaml.tpl`
- `bin/start.sh.tpl`

### `templates/`

这个目录更适合放：

- `_helpers.tpl`
- `_partials.tpl`
- 通过 `define` 声明的命名模板

这些 helper 一般不会直接作为最终输出文件，而是被其他模板通过 `include` / `template` / `tpl` 调用。

## 4. values 目录建议结构

`values/` 下建议按“配置组”或“环境”分层，例如：

- `default/`
- `dev/`
- `test/`
- `production/`

每个配置组内可以包含以下几类文件。

### `global.yaml`

作用：

- 作为所有 chart 的公共默认值来源

注意：

- 当前实现中，`global.yaml` 不会覆盖 chart 默认值的同名 key
- 更适合作为“补齐默认项”的公共层

### `<service>.yaml`

作用：

- 作为单个 chart / 服务的专属覆盖层

文件名解析规则：

1. 优先使用 chart 默认值中的 `type_name`
2. 如果没有 `type_name`，使用 `func_name`
3. 如果两者都没有，则回退为 chart 名

### `modules/*.yaml`

作用：

- 按模块名注入可选能力默认值

例如：

- `modules/logging.yaml`
- `modules/cache.yaml`

注入后会出现在：

- `.Values.logging`
- `.Values.cache`

模块是否真正生效，取决于：

- 更高优先级来源里的 `<module>.enabled`
- 模块文件自身的 `enabled`

### `non_cloud_native/deploy.yaml`

作用：

- 提供 `template` 模式所需的实例清单

典型字段包括：

- `world_id`
- `zone_id`
- `proc_desc`
  - `chart_name`
  - `instance_type_id`
  - `world_instance`
  - `instance_count`
  - `start_instance_id`

## 5. 输出目录的典型结构

执行 `template` 后，输出目录通常会按 chart 名和原始相对路径组织，例如：

```text
target/rendered/
  example/
    cfg/
      example_1.2.42.3.yaml
    bin/
      start_1.2.42.3.sh
```

文件名中的 `1.2.42.3` 即 `bus_addr` 后缀。

## 6. 推荐实践

1. chart 默认值尽量写稳定默认项，不要把环境特有值塞回 `values.yaml`
2. 公共配置放 `global.yaml`，服务差异放 `<service>.yaml`
3. 可选能力放 `modules/*.yaml`，并配合 `enabled` 控制
4. `template` 使用时始终显式指定输出目录
5. 模板作者应把 helper 放在 `templates/_*.tpl`，把最终输出模板放在 `cfg/`、`bin/` 等目录中

## 相关阅读

- [`project-structure.md`](project-structure.md)
- [`render-flow.md`](render-flow.md)
- [`../usage/template.md`](../usage/template.md)
- [`../usage/values-and-overrides.md`](../usage/values-and-overrides.md)
