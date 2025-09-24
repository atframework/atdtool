# atdtool

atframework deploy tool , 用于atframework的部署工具。

支持云上和云原生模式

## 配置工具atdtool使用说明

详细使用说明请参考 `atdtool --help` 命令

| 命令                     | 说明                     |
| :----------------------- | :----------------------- |
| `atdtool version``      | 查看atdtool信息          |
| `atdtool template``     | 渲染配置模板             |
| `atdtool merge-values`` | 合并Values               |
| `atdtool guid``         | 生成唯一ID（雪花算法）   |
| `atdtool watch``        | 监听文件变化执行相关命令 |

## 配置作用域说明

- Global配置作用于所有进程
- Modules中的配置可以被所有进程所引用
- Chart中的values只作用于当前进程

## 配置优先级说明

- Global的优先级高于modules
- Chart中的values的优先级要高于global
- Values指定目录的优先级按照从左到右依次递增

```shell
--values default,default_tgf,tgf_dev # 优先级关系 default < default_tgf < tgf_dev
```

## Module使用说明

- 模块配置需要放到modules目录
- 模块需要指定是否默认启用

```shell
# 通用模块可以默认启用
# 通用模块示例
# modules/tbuspp.yaml
enabled: true

# 非通用模块建议关闭默认启用，按需引入模块
# 非通用模块示例
# modules/ace.yaml
enabled: false
```

- 服务启用和关闭模块方式

```shell
# gamesvr启用apollo模块示例
# charts/gamesvr/values.yaml
apollo:
  enabled: true

# tconnd关闭hpa模块示例
# charts/tconnd/values.yaml
hpa:
  enabled: false
```

- 模块配置模板时需要加上保护（服务不启用某个模块的情况下不会加载对应的配置）

```shell
# 模板引用模块保护示例
# libapp/templates/_atapp.logic.yaml.tpl
{{- if .Values.tbuspp }}
...
{{- end }}
```

## 配置覆盖说明

- 配置修改只需指定待覆盖的配置项（切勿再拷贝一份完整的配置）

```shell
# 源配置
cache_cfg:
  watcher:
    heartbeat_interval: 15m # 自动更新watcher的心跳间隔
    expired_timeout: 32m # 订阅自动清理时间，由于有些模块是按分钟的精度。最好大于 watcher.heartbeat_interval 的2倍
    check_interval: 18m # Watcher的检查间隔
    max_number: 2000000 #
    max_recycle_count_per_tick: 1000 # 每个tick最大回收缓存数量
  data:
    cache_data_expired_timeout: 60m # 如果未主动标记过期，缓存的被动刷新周期
    cache_expired_timeout: 35m # 缓存对象长时间未访问自动清理时间，最好大于 watcher.heartbeat_interval 的2倍
    cache_check_interval: 10m # 缓存对象的生命周期检查间隔
    cache_fallback_expired_timeout: 5m # 缓存服务未开启时本地模块的缓存过期时间
    max_recycle_count_per_tick: 100 # 每个tick最大回收缓存数量
    max_user_cache_number: 200000 # 最大玩家数据缓存数量，超出后会强制回收最老的数据块
    gc_user_cache_number: 100000 # 开始主动执行GC的玩家缓存数量

# 正确覆盖示例
cache_cfg:
   watcher:
     check_interval: 18m
```
- 修改modules相关配置Key无需再加上对应的模块名称
```shell
# 错误覆盖示例
# ds.yaml
ds:
  pre_alloc_ds_count: 1
  disabled_pre_alloc_alias: []

# 正确覆盖示例
# ds.yaml
pre_alloc_ds_count: 1
disabled_pre_alloc_alias: []
```
- 修改对应进程配置Key无需再加上对应的进程名
```shell
# 错误示例
# gamesvr.yaml
gamesvr:
  ace:
    enable_sdk: false

# 正确示例
# gamesvr.yaml
ace:
  enable_sdk: false
```

## 配置生成

1.1. 配置生成命令

```shell
Usage:
  atdtool template [CHART] [flags]

Flags:
      --devel             enable develop mode
  -h, --help              help for template
  -o, --output string     specify templates rendered result save path
  -s, --set stringArray   set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)
      --non-cloud-native  enable non cloud-native mode
  -p, --values strings    set values path on the command line (can specify multiple paths with commas:path1,path2)
```

1.2. Flag详细说明

- 通过--values可以指定配置源加载路径
- 通过--set可以指定单个配置项的值

```shell
# 替换某一个结构成员值
--set battlesvr.ds.default.max_fps=0

# 指定数组中某个元素值
--set battlesvr.ds.default.oss_log_server_list[0]="11.152.245.181:7788"

# 指定整个数组值
--set battlesvr.ds.default.pre_alloc_ds_maps="{1,5}"
```