## GoOne

GoOne 是一套基于 Go 实现的微服务分布式游戏服务器框架，核心思路是 **Reactor + CSP**（并发消息驱动）且继承了很多C++游戏架构的思想，并配套提供：服务治理、配置中心、消息总线、网络层、部署控制台等“工程化”能力，适用于中小型游戏、MMO 等游戏后端业务。

[![Go Version](https://img.shields.io/github/go-mod/go-version/Iori372552686/GoOne)](go.mod)
[![License](https://img.shields.io/github/license/Iori372552686/GoOne)](LICENSE)
[![Stars](https://img.shields.io/github/stars/Iori372552686/GoOne?style=flat)](https://github.com/Iori372552686/GoOne/stargazers)

---

## 1. 架构概览

![image](https://github.com/user-attachments/assets/991e2091-dbd9-4f8f-9e0b-5c24ed98bf3b)

- **网关服（connsvr）**：管理客户端连接，接入并转发消息
- **核心逻辑服（mainsvr）**：核心业务逻辑
- **信息服（infosvr）**：角色/信息等服务
- **数据服务（mysqlsvr/dbsvr 等）**：存储、持久化、数据访问
- **Web 管理（websvr）**：HTTP 管理接口/后台

更详细的架构说明见：
- **架构文档**：[`/doc/G1服务器技术架构文档.docx`](/doc/G1服务器技术架构文档.docx)

---

## 2. 优势

- **工程化**：提供统一的控制台入口 `main.sh`，把常用操作标准化（doctor/go/docker/build/deploy）
- **可扩展**：注册中心 / 配置中心 / 消息总线均采用接口 + 工厂抽象，便于替换实现
- **易落地**：提供 `deploy/`（Ansible）与 `env/`（Docker 依赖环境）开箱即用
- **高性能导向**：网络层支持 tcp/kcp/ws，多服务按 busId 路由通信

---

## 3. 功能模块

- **ssrpc 运行时**：`lib/service/ssrpc/*` -- IDL 驱动的统一 RPC 框架，四传输层对称（见下文）
- **网络层**：`lib/net/*`（tcp/kcp/ws）
- **服务注册发现**：`lib/contrib/registry/*`（默认 ZooKeeper，支持 etcd/consul/nacos/k8s 等）
- **配置中心**：`lib/contrib/config/*`（支持 apollo/etcd/consul/nacos/k8s 等，并提供本地 Manager 聚合/热更新）
- **消息总线（Bus）**：`lib/service/bus/*`（RabbitMQ / NSQ / NATS / Kafka / RocketMQ，统一 `IBus` 接口）
- **路由与服务编排**：`lib/service/router/*`、`lib/service/svrinstmgr/*`
- **部署与运维**：`deploy/*`（Ansible），`env/*`（依赖服务 Docker compose）
- **工具链**：`tools/protoc-gen-goone`（IDL 代码生成器）、`tools/cmd/genproto`（主仓 proto 生成入口）、`tools/cmd/scaffold`（脚手架）、`tools/cfgtool/*`（配置工具）

---

## 3.1 ssrpc -- IDL 驱动的统一 RPC 框架

GoOne 的 ssrpc 模块借鉴 CloudWeGo 的 IDL 驱动 + Kratos 的 middleware/transport 分层，嫁接到现有的 TransactionMgr + SSPacket 执行模型上。

**核心理念：定义一次 proto service，四条传输路径自动生成。**

| 传输层    | Wrap 函数           | 挂载点               | Proto option              |
|----------|--------------------|--------------------|--------------------------|
| SSPacket | `WrapUnary`        | `TransactionMgr`    | `cmd/cmd_enum/cmd_name`  |
| HTTP     | `WrapHTTPGin`      | `gin.IRoutes`       | `http_path`              |
| WS       | `WrapWS`           | `ssrpc.Dispatcher`  | `ws: true`               |
| gRPC     | `WrapGRPCUnary`    | `grpc.Server`       | `grpc: true`             |

**关键组件：**

- **`ssrpc.Dispatcher`** -- 统一注册中心，每个 service 会生成 `Register<Service>ToDispatcher(...)` 来一次注册多条传输路径
- **`ssrpc.Context`** -- 统一请求上下文，middleware 无需感知底层传输协议
- **`ssrpc.Middleware`** -- `func(next Handler) Handler` 链式中间件（recover/log/auth/sign/uid_lock/trace/metrics）
- **`protoc-gen-goone`** -- 从 proto service 定义自动生成 Server 注册 + Client Stub
- **`ssrpc.CallByCmd` / `CallByCmdWithRouter` / `SendByCmdSimple`** -- 生成的 Client Stub 使用的 helper，从 CMD 自动提取 svrType，并支持显式 `routerId` 与无 `IContext` 的 one-way send

**当前协议归属：**

- `game_protocol/`：共享消息、枚举、`CMD`，以及逐步迁入的 protocol-owned service proto
- `api/proto/goone/**`：GoOne 自己的 options / cmd enum 定义
- `api/proto/game/**`：当前仅保留主仓本地示例或尚未迁出的 proto
- `api/gen/**`：统一生成产物

**当前推荐生成命令：**

```bash
go run ./tools/cmd/genproto
```

或使用一键包装：

```bash
./scripts/proto_goone.sh
# Windows / PowerShell
./scripts/proto_goone.ps1
```

包装脚本会先生成 `game_protocol/protocol/**` 的消息 `pb.go`，再生成主仓 `api/gen/**`。

**校验生成物是否与工具链一致（推荐进 CI）：**

```bash
./main.sh check-genproto
./main.sh check-genproto --full
```

Windows PowerShell（不依赖 bash 时）：

```powershell
.\scripts\check_genproto.ps1
.\scripts\check_genproto.ps1 -Full
```

默认模式会先执行 `go run ./tools/cmd/genproto`，再检查 `api/gen/**` 是否仍有未提交的差异。`--full` / `-Full` 会改跑完整 `proto_goone` 流程，并额外检查 `game_protocol/protocol/**`。若你改的是 `game_protocol` 里的消息而非仅 service proto，建议直接使用 full 模式。

**示例：**

```proto
service MainC2SService {
  rpc Login(g1.protocol.LoginReq) returns (g1.protocol.LoginRsp) {
    option (goone.options.v1.ssrpc) = {
      cmd_name: "CMD_MAIN_LOGIN_REQ"
      ws: true
      grpc: true
    };
  }
}
```

```go
// Server 注册（一行搞定四条路径）
d := ssrpc.NewDispatcher()
mainsvrv1.RegisterMainC2SServiceToDispatcher(d, srv)
d.RegisterToTransactionMgr(globals.TransMgr)  // SSPacket
d.MountGin(router)                             // HTTP
d.MountGRPC(grpcSrv)                           // gRPC

// Client 调用（类型安全，无样板代码）
client := mainsvrv1.NewMainC2SServiceClient()
rsp, err := client.Login(ctx, &g1_protocol.LoginReq{...})
```
---

## 3.2 脚手架工具

快速创建新服务骨架：

```bash
go run tools/cmd/scaffold -name mysvr
```

生成 `src/mysvrsvr/` 目录（`main.go` / `app.go` / `globals` / `cmd_handler`），与 `infosvr` 结构一致。
当前脚手架仍会保留一个兼容用的 `cmd_handler/register.go` 骨架；如果新服务走 IDL-first，通常应在 `app.go` 里接生成的 `Register<Service>ToDispatcher(...)` / `Register<Service>ToTransactionMgr(...)`。

---

## 4. 快速开始（推荐：通过主控制台 main.sh）

> `main.sh` 是统一入口（类似 “框架控制台”）。在 Windows 上建议使用 **WSL2 或 Git-Bash** 来执行这些 bash 脚本。

### 4.1 自检（强烈推荐先跑）

```bash
./main.sh doctor
./main.sh help
```

### 4.2 安装/切换 Go 版本（可选，但推荐与 go.mod 对齐）

项目 `go.mod` 当前要求 Go `1.25.4`。

```bash
./main.sh go list
./main.sh go install 1.25.4
./main.sh go use 1.25.4
./main.sh go current
```

### 4.3 一键拉起依赖环境（MySQL / Redis / ZooKeeper / RabbitMQ）

**本地开发（推荐：直接 docker compose）**：

```bash
docker compose -f env/env_docker.yaml up -d
```

**远程/统一管理（通过 main.sh + Ansible）**：

```bash
# 需要先在部署机安装 ansible（只需做一次）
./main.sh install ansible

./main.sh docker install --env dev_local
./main.sh docker status  --env dev_local
```

> Docker 配置来自：[`env/env_docker.yaml`](env/env_docker.yaml)

### 4.4 编译

```bash
./main.sh build          # 默认编译一组核心服务
./main.sh build web      # 单独编译 websvr（target 由 build.sh 定义）
```

### 4.5 本地运行（IDE/调试）

各服务通过统一 flag 读取配置（`-svr_conf`），示例配置见：
- `env/server_conf_ide.yaml`

示例（按需启动）：

```bash
./build/connsvr  -svr_conf=./env/server_conf_ide.yaml
./build/mainsvr  -svr_conf=./env/server_conf_ide.yaml
./build/infosvr  -svr_conf=./env/server_conf_ide.yaml
./build/websvr   -svr_conf=./env/server_conf_ide.yaml
```

---

## 5. 部署（Ansible）

部署相关脚本在 `deploy/`，推荐仍通过 `main.sh` 调度：

### 5.1 安装 Ansible（部署机/控制机）

```bash
./main.sh install ansible
```

### 5.2 列出环境与角色

```bash
./main.sh env list
./main.sh role list
```

### 5.3 部署/重启服务

```bash
./main.sh deploy --env dev1 --action restart --role websvr
./main.sh deploy --env dev1 --action restart --roles websvr,mainsvr --dry-run
```

更多说明：
- 部署说明：[`deploy/README.md`](deploy/README.md)

---

## 6. 安装与环境（Linux / Windows）

- Linux 安装与依赖说明：[`doc/setup_linux.md`](doc/setup_linux.md)
- Windows 安装与依赖说明：[`doc/setup_win.md`](doc/setup_win.md)

> 提示：历史文档里部分 “手动安装 Go 1.13 / 手动装中间件” 已过时；当前更推荐使用 `main.sh go ...` 与 `main.sh docker ...` 统一管理。

---

## 7. 目录结构（Top Level）

```text
api/        IDL 定义 (api/proto/) 与生成代码 (api/gen/)
build/      编译后的可执行文件输出目录
common/     公共模块（配置/常量/工具/游戏数据等）
deploy/     Ansible 自动化部署脚本与 roles
doc/        文档（架构/环境搭建等）
env/        本地依赖环境（docker compose）与调试配置
game_protocol/ 协议子仓（共享消息 / CMD / protocol-owned service proto）
lib/        框架核心库
  lib/service/ssrpc/    ssrpc 运行时（Context/Dispatcher/Middleware/Wrap*/Client）
  lib/service/bus/      消息总线（RabbitMQ/NATS/Kafka/...）
  lib/service/router/   路由与服务编排
  lib/net/              网络层（tcp/kcp/ws）
  lib/contrib/          注册中心 / 配置中心
src/        业务服务（connsvr/mainsvr/infosvr/web_svr/roomcentersvr/mysqlsvr）
tools/      工具链
  tools/protoc-gen-goone/   IDL 代码生成器（protoc 插件）
  tools/cmd/scaffold/       脚手架工具（生成新服务骨架）
  tools/cmd/genproto/       proto 编译脚本
  tools/cfgtool/            配置工具
main.sh     主控制台脚本（推荐入口）
```

---

## 8. 常见问题（FAQ）

- **在 Windows 上怎么运行 main.sh？**
  - 建议使用 **WSL2** 或 **Git-Bash**；部署工具还需要 Python（用于安装 Ansible）。
- **本地 go test 会因为 MQ/注册中心没启动而失败吗？**
  - 某些包是集成测试性质，建议先启动 `env/env_docker.yaml` 中的依赖后再联调。
- **部署 inventory/账号密码在哪里配置？**
  - 在 `deploy/hosts/*` 与 `deploy/playbook_dev/*`；生产环境务必使用安全的凭据管理方式，避免明文提交。

---

## 9. 交流与贡献

- 联系 QQ：372552686
- QQ 群：767770895

---

## 10. License

GoOne 使用 MIT 协议发布，详见 [`LICENSE`](LICENSE)。
