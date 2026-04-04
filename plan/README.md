# GoOne 架构评审与演进计划

## 目的

本目录用于沉淀对 `GoOne` 项目的系统性分析与后续改进计划，输出深度采用：

- **B：架构评审 + 模块逐项优化建议**
- 比较对象侧重：**C：通用游戏后端平台能力**
  - 参考能力模型包括但不限于：Nakama、Photon/Realtime、PlayFab、GameLift、Agones/K8s 游戏托管体系，以及常见商业化游戏后端平台的 LiveOps/可观测/会话治理能力。

## 仓库定位

结合 `AGENTS.md` 与当前代码：

- `GoOne` 是一个 **Go 微服务化游戏后端框架**；
- 主体服务位于 `src/`：`connsvr`、`infosvr`、`mainsvr`、`mysqlsvr`、`roomcentersvr`、`web_svr`；
- 共享底层位于 `lib/`，其中核心运行时包括：
  - `lib/service/application`：主循环与信号处理
  - `lib/service/bootstrap`：服务装配、管理端口、初始化阶段
  - `lib/service/router`：跨服务消息收发与路由
  - `lib/service/transaction`：串行事务执行模型
  - `lib/service/ssrpc`：统一 RPC/HTTP/WS/gRPC 运行时
- 配置在 `common/gconf/config.go`，目前处于“旧平铺字段 -> 新分组结构”的迁移阶段。

## 重点证据入口

建议先读以下文件，再读各专题文档：

- 服务主循环：`lib/service/application/app.go`
- 服务装配：`lib/service/bootstrap/app.go`
- 管理端口：`lib/service/bootstrap/admin.go`
- 配置模型：`common/gconf/config.go`
- 服务路由：`lib/service/router/router.go`
- 事务模型：`lib/service/transaction/transaction_mgr_impl.go`
- 事务等待/超时：`lib/service/transaction/transaction_impl.go`
- 服务发现与实例管理：`lib/service/svrinstmgr/svr_inst_mgr.go`
- 统一 RPC 运行时：`lib/service/ssrpc/dispatcher.go`
- HTTP 封装：`lib/service/ssrpc/http.go`
- gRPC 封装：`lib/service/ssrpc/grpc.go`
- Web 服务入口：`src/web_svr/app.go`
- 主逻辑服入口：`src/mainsvr/app.go`
- 构建脚本：`main.sh`、`build.sh`

## 文档索引

| 文件 | 作用 |
|---|---|
| `01-architecture-review.md` | 从整体架构、运行时模型、风险点和保留原则出发做总评审 |
| `02-module-recommendations.md` | 按模块/服务逐项给出优化建议 |
| `03-platform-capability-comparison.md` | 对照通用游戏后端平台能力，分析当前缺项与差距 |
| `04-operations-and-quality-plan.md` | 围绕构建、测试、发布、观测、可运维性给出治理方案 |
| `05-prioritized-roadmap.md` | 将建议转成可执行的分阶段路线图 |
| `06-execution-priority-plan.md` | 将全部分析整合成单独执行计划，按 P0-P5 给出当前形态、改动内容与落地方案 |

## 使用建议

建议阅读顺序：

1. `01-architecture-review.md`
2. `02-module-recommendations.md`
3. `03-platform-capability-comparison.md`
4. `04-operations-and-quality-plan.md`
5. `05-prioritized-roadmap.md`
6. `06-execution-priority-plan.md`

## 文档状态

| 文档 | 状态 |
|---|---|
| `README.md` | Draft |
| `01-architecture-review.md` | Draft |
| `02-module-recommendations.md` | Draft |
| `03-platform-capability-comparison.md` | Draft |
| `04-operations-and-quality-plan.md` | Draft |
| `05-prioritized-roadmap.md` | Draft |
| `06-execution-priority-plan.md` | Draft |

