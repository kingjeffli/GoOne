# Config中间件 抽象层（本地 Source/Watcher/KeyValue）

本目录提供一套**本地**的 `Source/Watcher/KeyValue` 配置中间件抽象，并在其上实现多种配置中心 Source：

- 各配置中心的 `Source` 实现（`apollo/consul/etcd/kubernetes`）
- 一个 **统一工厂**（`config/factory`）：用一条 `scheme://...` 字符串即可创建 Source
- 更清晰的本地封装面（`lib/contrib/config/config.go`）：`Client(Source + Close)`

## 目录结构

- `config.go`
  - `Client`：`Source + Close()` 的轻量封装（便于资源回收）
- `factory/`
  - `ParseConfig` / `NewFromAddr`：解析 URI + 创建具体 Source
  - 避免 import cycle：根包 `config` 不依赖任何 backend
- `apollo/`、`consul/`、`etcd/`、`kubernetes/`
  - 各配置中心实现（多数为 Kratos contrib 风格移植）

## 快速使用（工厂）

### Consul

`consul://127.0.0.1:8500?path=goone/config&timeout=5s`

```go
cli, cfg, err := factory.NewFromAddr("consul://127.0.0.1:8500?path=goone/config")
_ = cfg
defer cli.Close()
```

### Kubernetes ConfigMap

`k8s://?namespace=default&label=app=goone&kubeconfig=/path/to/kubeconfig`

```go
cli, _, err := factory.NewFromAddr("k8s://?namespace=default&label=app=goone")
defer cli.Close()
```

### Apollo

`apollo://?appid=goone&endpoint=http://127.0.0.1:8080&cluster=dev&namespace=application.yaml,demo.json&backup=true`

```go
cli, _, err := factory.NewFromAddr("apollo://?appid=goone&endpoint=http://127.0.0.1:8080&cluster=dev&namespace=application.yaml")
defer cli.Close()
```

### Nacos

`nacos://127.0.0.1:8848?dataid=application.yaml,demo.json&group=DEFAULT_GROUP&namespace_id=public&timeout=5s`

```go
cli, _, err := factory.NewFromAddr("nacos://127.0.0.1:8848?dataid=application.yaml&group=DEFAULT_GROUP")
defer cli.Close()
```

### etcd（可选启用）

`etcd://127.0.0.1:2379?path=/goone/config&prefix=true&timeout=5s`

默认构建 **不启用 etcd**，需要：

`-tags config_etcd`

原因与 registry 类似：etcd 依赖链可能触发 protobuf extension 冲突（在部分依赖组合下会 panic），因此把 etcd 做成显式启用更安全。

## 与上层配置系统的组合

`Client` 返回的是本地 `Source`，你可以自行在上层实现“合并多 Source / 结构体反序列化 / 热更新回调”等能力。

## 兼容性 / 健壮性约定

- 工厂只负责：**解析字符串 + 组装选项 + 创建 Source**
- `Normalize()` 会补齐默认超时与必填项校验，避免在运行时隐式失败
- Apollo 原 `NewSource` 会 panic；已新增 `NewSourceE`（返回 error），工厂使用 `NewSourceE`


