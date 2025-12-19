## GoOne

GoOne 是一套基于 Go 实现的分布式游戏服务器框架，核心思路是 **Reactor + CSP**（并发消息驱动），并配套提供：服务治理、配置中心、消息总线、网络层、部署控制台等“工程化”能力，适用于中小型游戏、MMO 等服务端架构。

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

- **网络层**：`lib/net/*`（tcp/kcp/ws）
- **服务注册发现**：`lib/contrib/registry/*`（默认 ZooKeeper，支持 etcd/consul/nacos/k8s 等）
- **配置中心**：`lib/contrib/config/*`（支持 apollo/etcd/consul/nacos/k8s 等，并提供本地 Manager 聚合/热更新）
- **消息总线（Bus）**：`lib/service/bus/*`（RabbitMQ / NSQ / NATS / Kafka / RocketMQ，统一 `IBus` 接口）
- **路由与服务编排**：`lib/service/router/*`、`lib/service/svrinstmgr/*`
- **部署与运维**：`deploy/*`（Ansible），`env/*`（依赖服务 Docker compose）
- **工具链**：`tools/cfgtool/*`（配置工具）、`lib/util/xlstrans/*`（表格转换相关）

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
build/      编译后的可执行文件输出目录
common/     公共模块（配置/常量/工具/游戏数据等）
deploy/     Ansible 自动化部署脚本与 roles
doc/        文档（架构/环境搭建等）
env/        本地依赖环境（docker compose）与调试配置
lib/        框架核心库（网络/服务治理/Bus/工具等）
src/        业务示例服务（connsvr/mainsvr/infosvr/web_svr/...）
tools/      工具（cfgtool 等）
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
