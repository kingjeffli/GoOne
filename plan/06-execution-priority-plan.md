# 06. GoOne 详细执行计划（P0-P5）

## 1. 目的

本文是对以下文档的执行化整合：

- `plan/01-architecture-review.md`
- `plan/02-module-recommendations.md`
- `plan/03-platform-capability-comparison.md`
- `plan/04-operations-and-quality-plan.md`
- `plan/05-prioritized-roadmap.md`

目标不是继续做“分析”，而是直接输出一份可执行计划，明确：

1. **P0-P5 优先级划分**
2. **当前业务/系统的真实形态**
3. **每项改动落地后会影响什么**
4. **每项改动应该如何落地**
5. **建议修改的文件、模块、脚本与运行时边界**
6. **每项任务的依赖关系与验收标准**

---

## 2. 当前业务与系统整体形态

## 2.1 当前业务形态

从当前仓库结构看，GoOne 目前更偏向一套“**已经能承载真实游戏业务的微服务后端框架**”，其业务承载形态如下：

- `connsvr`：客户端接入层，负责 TCP / WS 连接与客户端消息转发；
- `mainsvr`：玩家主逻辑层，承担角色状态、玩家业务、较核心的角色域逻辑；
- `infosvr`：轻量信息/缓存查询层；
- `mysqlsvr`：数据持久化访问层；
- `roomcentersvr`：房间中心、房间 tick、房间列表与部分玩法调度；
- `web_svr`：HTTP / gRPC 管理接口与对外 API 层。

当前业务的主特点是：

1. **核心链路是玩家与房间驱动型**，而不是纯无状态 API 驱动；
2. **消息驱动 + 串行事务** 是主模型；
3. 当前更像“业务框架 + 服务底座”，还不是一个成熟的“产品化游戏平台”；
4. 业务主链已经可以跑，但平台能力层不够完整。

## 2.2 当前系统形态

系统当前的主运行模型是：

- 服务从 `src/<service>/main.go` 启动；
- 统一经由 `application.Init(newApp())` + `application.Run()` 进入运行态；
- `bootstrap.NewServiceApp(...)` 负责配置加载、依赖初始化、处理器注册和 runtime 启动；
- `router + svrinstmgr + bus` 负责服务间路由；
- `transaction` 负责按 `uid/routerId` 串行执行；
- `ssrpc` 负责 SSPacket / HTTP / WS / gRPC 统一处理模型。

## 2.3 当前系统最重要的现实判断

### 已经具备的优势

- 有清晰的服务切分；
- 有统一 bootstrap 层；
- 有 router + transaction 主干；
- 有 `ssrpc` 多 transport 统一抽象；
- 有基础 metrics / health / pprof；
- 有一定测试基础。

### 当前最主要的问题

- `application.Run()` 忙轮询；
- 生命周期关闭能力不完整；
- 构建脚本和目录结构漂移；
- CI / lint / pipeline 缺失；
- transaction timeout 等关键参数硬编码；
- tracing 只有接口，没有成品；
- session/presence/room orchestration/RBAC/audit/LiveOps 等平台能力缺失。

---

## 3. 优先级总览

| 优先级 | 目标 | 核心主题 | 时间建议 |
|---|---|---|---|
| P0 | 先把系统跑稳 | 主循环、关闭、脚本一致性、CI 最小闭环、事务关键硬编码 | 立即开始 |
| P1 | 补齐基础平台层 | tracing、bootstrap 增强、配置 reload 边界、admin 扩展、服务角色矩阵 | P0 后 |
| P2 | 形成中层业务平台能力 | session/presence、房间生命周期、web 管理面分层、MySQL/Info 服务边界治理 | P1 后 |
| P3 | 完善运营与治理能力 | RBAC、审计、feature flags、基础 LiveOps | P2 后 |
| P4 | 补齐平台高级能力 | async jobs、analytics、risk control、anti-cheat hooks | P3 后 |
| P5 | 长期平台产品化 | 弹性调度、托管、成本治理、区域化与多环境平台能力 | 长期 |

---

# 4. P0：必须先落地的稳定性改造

## P0-1 `application.Run()` 忙轮询治理

### 当前业务形态

当前所有业务服务都共享同一主循环。业务侧并没有真正依赖一个高频空转的轮询框架，大多数服务的 `OnProc()` 只是简单 `return true`。

### 当前系统形态

涉及文件：

- `lib/service/application/app.go`
- `src/connsvr/app.go`
- `src/infosvr/app.go`
- `src/mainsvr/app.go`
- `src/mysqlsvr/app.go`
- `src/roomcentersvr/app.go`
- `src/web_svr/app.go`

当前问题：

- `for {}` 忙轮询；
- 空闲期依然反复执行；
- Tick 依赖轮询判断；
- 空闲 CPU 存在浪费。

### 改动内容

1. 重构 `application.Run()` 为 **ticker + signal + optional work loop** 模型；
2. 重新定义 `OnProc()` 语义；
3. 明确哪些服务根本不需要 `OnProc()`；
4. 若保留 `OnProc()`，其内部必须具备阻塞等待语义或有限工作语义。

### 落地方案

#### 方案说明

- 用 `time.Ticker` 代替当前 tick 的忙判断；
- 用信号 channel 统一处理退出与 reload；
- `OnProc()` 若无实际业务，则可以只在 ticker 或事件驱动下触发，不能继续空转；
- 保持现有 `AppInterface` 尽量兼容，优先做“内部实现替换”，而不是大改外层调用约定。

#### 建议改动文件

- `lib/service/application/app.go`
- `lib/service/application/sig.go`
- `lib/service/application/sig_windows.go`

### 验收标准

- 服务空闲 CPU 明显下降；
- 功能行为不变；
- reload / exit 行为保持可用；
- 现有服务入口无需大面积改写。

---

## P0-2 生命周期关闭协议（shutdown contract）

### 当前业务形态

当前业务服务可以启动并工作，但停止过程更多依赖进程结束，而不是组件级优雅收口。

### 当前系统形态

关键缺口位于：

- `router` 无统一 `Close()`；
- `transaction` 无统一 `Shutdown(ctx)`；
- `bus` 无 `Close()` 契约；
- `svrinstmgr` 虽有 `Close()`，但未统一挂入服务退出路径；
- 后台 goroutine 停止语义不统一。

### 改动内容

1. 给 router 增加关闭能力；
2. 给 transaction 增加优雅停机能力；
3. 给 bus 抽象补生命周期；
4. 将 watcher / goroutine / listener 纳入统一退出流程；
5. 让 `bootstrap.ServiceApp.OnExit()` 有统一收口职责。

### 落地方案

#### 方案说明

先做“最小一致性版本”：

- 停止接新流量；
- 停止接新消息；
- 让在途事务有机会结束；
- 超时后快速失败退出；
- 所有可关闭组件有统一接口。

#### 建议改动文件

- `lib/service/router/router.go`
- `lib/service/transaction/transaction_mgr_impl.go`
- `lib/service/transaction/transaction_i.go`
- `lib/service/bus/bus_i.go`
- 各 bus 实现文件
- `lib/service/bootstrap/app.go`
- 相关服务 `app.go`

### 验收标准

- 服务收到退出信号后能优雅停止；
- 无明显 goroutine 泄漏；
- 路由、事务、watcher 能正常结束；
- 发布/重启可靠性提升。

---

## P0-3 `main.sh` / `build.sh` / `etc/env` 一致性修复

### 当前业务形态

当前业务服务集已集中在 `src/` 的几个活跃服务中，但构建与环境脚本还保留历史状态。

### 当前系统形态

问题包括：

- `main.sh` 部分路径仍以 `env/*` 为假设；
- 实际目录在 `etc/env/*`；
- `build.sh` 保留了大量历史服务；
- `build.sh` 未体现 `roomcentersvr` 等当前活跃服务；
- 推荐入口与真实脚本行为不完全一致。

### 改动内容

1. 统一脚本路径；
2. 清理历史构建目标；
3. 以当前活跃服务为准重写 build 入口；
4. 补齐 Windows/PowerShell 对应说明或脚本；
5. 让 `AGENTS.md`、README、脚本三者一致。

### 落地方案

#### 方案说明

- `main.sh` 作为主入口；
- `build.sh` 简化成纯构建工具；
- 对 `etc/env/*` 做显式引用；
- 删除或标记 legacy 入口；
- 若短期不支持的命令，明确报错而不是静默漂移。

#### 建议改动文件

- `main.sh`
- `build.sh`
- 必要时新增 PowerShell 对应脚本
- `readme.md`
- `AGENTS.md`（若后续需要同步）

### 验收标准

- 当前活跃服务均可一键构建；
- 环境目录引用准确；
- 文档与脚本一致；
- 新人可按单一路径完成本地启动。

---

## P0-4 最小 CI 基线

### 当前业务形态

业务功能已有一定测试，但依赖人工执行，不具备平台化质量门禁。

### 当前系统形态

未观察到：

- CI workflow
- lint 规则
- 统一自动检查

### 改动内容

1. 增加最小 CI；
2. 加入格式检查、构建检查、单测、proto 检查；
3. 把当前已有测试资产接入流水线。

### 落地方案

#### 方案说明

第一版不追求复杂，只要实现：

- `go test ./...` 的合理分层执行；
- `go vet`；
- 核心服务 `go build`；
- proto 输出一致性检查。

#### 建议改动文件

- 新增 `.github/workflows/*.yml` 或等价 CI 文件
- 可能新增 `.golangci.yml`（如果引入）
- 必要时为测试加 `short`/跳过策略

### 验收标准

- PR 能自动发现基础质量问题；
- proto 漂移会被 CI 阻止；
- 核心服务构建失败能被自动发现。

---

## P0-5 transaction 关键硬编码参数配置化

### 当前业务形态

不同服务的事务压力模型其实不同：

- `mainsvr`：玩家主逻辑，强串行、强状态；
- `roomcentersvr`：房间中心，强节奏、强 tick；
- `infosvr`：轻查询；
- `mysqlsvr`：偏持久化；
- `connsvr`：少量内部事务 + 连接网关。

### 当前系统形态

仍存在：

- `waitRsp` 3 秒硬编码；
- dispatch 超时硬编码；
- 某些 queue 行为固定在代码里；
- 服务调优空间不足。

### 改动内容

1. 将 timeout / queue / shard 等关键参数纳入配置；
2. 区分服务级默认值；
3. 在 `gconf` 中增加 capacity/runtime 配置项；
4. 保持向后兼容。

### 落地方案

#### 方案说明

优先做“服务级配置化”，不要一上来做“按 cmd 配置化”，先保证：

- 每个服务可以独立配置 timeout；
- 每个服务可以独立配置 pending 上限；
- shard count 有合理默认。

#### 建议改动文件

- `lib/service/transaction/transaction_config.go`
- `lib/service/transaction/transaction_impl.go`
- `lib/service/transaction/transaction_mgr_impl.go`
- `common/gconf/config.go`
- 各服务 `app.go`
- 示例配置 yaml

### 验收标准

- 关键超时与容量参数可通过配置调整；
- 旧配置仍可运行；
- 各服务调优路径明确。

---

# 5. P1：基础平台层增强

## P1-1 bootstrap 强化与 admin 面增强

### 当前业务形态

业务服务启动流程已经比较统一，说明平台骨架已经形成。

### 当前系统形态

`bootstrap` 已有：

- 分阶段初始化
- admin server
- ready/alive

但还不够“可运维”。

### 改动内容

1. 增加组件状态暴露；
2. 增加 `/info`、`/components` 等端点；
3. 为后续 shutdown / reload / tracing 做统一挂点。

### 落地方案

#### 建议改动文件

- `lib/service/bootstrap/app.go`
- `lib/service/bootstrap/admin.go`

### 验收标准

- 能看到服务构建信息、依赖状态、组件状态；
- admin 面成为统一运维入口。

---

## P1-2 tracing 产品化

### 当前业务形态

当前业务链路已经跨多个服务，但排障仍主要靠日志与 metrics。

### 当前系统形态

`ssrpc` 已有 `TraceProvider` 接口，但没有真正实现。

### 改动内容

1. 接入最小 OpenTelemetry 实现；
2. 给 HTTP / gRPC / SSPacket / WS 加 span；
3. 将 trace id 注入日志；
4. 为后续 dashboard / alert 做观测闭环。

### 落地方案

#### 建议改动文件

- `lib/service/ssrpc/trace.go`
- `lib/service/ssrpc/default_mw.go`
- `lib/service/ssrpc/http.go`
- `lib/service/ssrpc/grpc.go`
- 可能新增 tracing 实现文件

### 验收标准

- 至少能看一条跨服务链路；
- trace id 能进入日志；
- 不破坏现有业务接口。

---

## P1-3 配置 reload 边界定义

### 当前业务形态

当前配置既承担业务参数，又承担运行时参数，但“哪些可以热生效”不明确。

### 当前系统形态

`OnReload()` 存在，但业务层热更新行为不完整。

### 改动内容

1. 将配置区分为冷/暖/热；
2. 文档化 reload 行为；
3. 为未来 watcher 真正接入铺路。

### 落地方案

- 优先做文档与结构层收口；
- 再做少量可安全热生效字段；
- 不要一开始就让所有配置支持热更。

### 验收标准

- 配置热更边界清晰；
- 至少一类暖配置可 reload 生效。

---

## P1-4 服务角色矩阵与 shard 策略明确化

### 当前业务形态

不同服务负载模式差异大，但当前事务策略缺少统一说明。

### 当前系统形态

- `mainsvr`、`roomcentersvr`：多 shard
- `connsvr`、`infosvr`、`mysqlsvr`：单 shard

### 改动内容

1. 明确每类服务的运行模型；
2. 为后续调优与扩容提供标准；
3. 防止团队误用事务模型。

### 落地方案

- 在文档与配置层同时定义；
- 必要时在 admin 输出当前 shard 策略。

### 验收标准

- 每个服务的运行模型可解释、可观测、可配置。

---

# 6. P2：中层平台能力建设

## P2-1 session / presence 正式建模

### 当前业务形态

目前在线状态、连接状态、业务会话状态还混在网关语义里。

### 改动内容

1. 区分 connection / session / presence；
2. 明确登录前后状态机；
3. 为好友、聊天、房间同步做基础能力。

### 落地方案

先建模，再逐步实现，不建议先写一堆散逻辑。

---

## P2-2 房间生命周期与匹配编排

### 当前业务形态

`roomcentersvr` 已有正确方向，但更像房间中心服务，而不是完整编排平台。

### 改动内容

1. 定义 room lifecycle；
2. 定义分配/回收流程；
3. 补齐房态、容量、房间指标。

### 落地方案

- 先统一 room state model；
- 再接 quick start / room list / tick 这些现有入口；
- 最终再扩展到 matchmaking。

---

## P2-3 `web_svr` 分层治理

### 当前业务形态

`web_svr` 当前是 HTTP/gRPC 对外组合层，但边界将来最容易膨胀。

### 改动内容

1. 拆分 public / admin / internal 路由；
2. 引入最小权限和审计接口；
3. 保持 `web_svr` 不承载核心状态机。

### 落地方案

先做路由与模块边界分层，再做 RBAC。

---

## P2-4 `infosvr` / `mysqlsvr` 边界治理

### 当前业务形态

这两个服务当前较轻，但长期最容易变成“杂项服务”和“万能数据服务”。

### 改动内容

1. 明确数据所有权；
2. 明确同步/异步访问模型；
3. 防止继续堆积杂逻辑。

### 落地方案

先在文档层定义边界，再在服务接口层收紧。

---

# 7. P3：运营与治理能力

## P3-1 RBAC 与审计

### 当前业务形态

后台与外部接口能力已经开始集中到 `web_svr`，但没有正式权限治理。

### 改动内容

1. 后台角色权限体系；
2. 敏感操作审计；
3. 配置修改留痕；
4. 高风险操作审批挂点。

### 落地方案

从最小后台能力开始，不必一次性做成大型中台。

---

## P3-2 feature flags / LiveOps 基础能力

### 当前业务形态

已有配置中心，但没有产品化的 flag / 活动治理能力。

### 改动内容

1. feature flag
2. 灰度开关
3. AB test 支撑
4. 配置回滚与版本记录

### 落地方案

建议与 `ssrpc` 上下文和 `web_svr` 管理面联动。

---

# 8. P4：平台高级能力

## P4-1 异步作业与事件平台

### 当前业务形态

当前有 bus，但更偏通信底层，不是产品化异步作业系统。

### 改动内容

1. durable jobs
2. delayed jobs
3. retries / dead letters
4. 业务工作流抽象

### 落地方案

将“业务异步平台”与“基础 MQ 抽象”分层，不混在 bus 内部直接实现。

---

## P4-2 analytics / risk / anti-cheat hooks

### 当前业务形态

当前业务已可承载角色与房间逻辑，但平台层没有分析/风控能力。

### 改动内容

1. 行为埋点
2. 经济流水
3. 风险事件
4. 反作弊接入点

### 落地方案

先做事件标准化，再做分析平台接入。

---

# 9. P5：长期平台产品化

## P5-1 弹性调度、托管与成本治理

### 当前业务形态

当前更像自托管服务集，而不是托管平台。

### 改动内容

1. 服务摘流能力；
2. 容量标签；
3. 房间实例弹性；
4. 与 K8s / Agones / Fleet 对接的准备；
5. 成本与容量治理。

### 落地方案

这部分不建议前置，必须等 P0-P4 稳住后再做。

---

## 10. 依赖关系图（简版）

### 第一层必须先做

- P0-1 主循环
- P0-2 shutdown
- P0-3 脚本一致性
- P0-4 CI
- P0-5 transaction 配置化

### 第二层建立平台基础

- P1-1 bootstrap/admin
- P1-2 tracing
- P1-3 配置 reload 边界
- P1-4 shard/service role matrix

### 第三层扩展业务平台能力

- P2-1 session/presence
- P2-2 房间生命周期
- P2-3 web 分层
- P2-4 infosvr/mysqlsvr 边界

### 第四层产品化治理能力

- P3 RBAC/审计/flags
- P4 async jobs / analytics / risk
- P5 托管与弹性

---

## 11. 建议实施顺序（直接可执行版）

### 第一批（立即开始）

1. `application.Run` 重构
2. shutdown contract 设计与落地
3. `main.sh` / `build.sh` 清理
4. transaction 参数配置化
5. 最小 CI

### 第二批

1. tracing
2. admin 扩展
3. reload 边界定义
4. 服务角色矩阵

### 第三批

1. session / presence
2. 房间生命周期 / 匹配编排
3. `web_svr` 分层

### 第四批

1. RBAC / audit
2. feature flags
3. async jobs
4. analytics / risk hooks

### 长期批次

1. autoscaling / fleet / 托管能力
2. 更完整的游戏平台产品化能力

---

## 12. 交付物要求

每个优先级任务落地时，建议至少输出以下交付物：

1. 代码改动
2. 配置改动
3. 文档改动
4. 测试改动
5. 验收方式
6. 回滚方式

这样可以避免“只有代码，没有交付闭环”。

---

## 13. 最终结论

这份执行计划的核心思想只有一句话：

> **先把 GoOne 做成稳定可靠的平台内核，再把它做成具备运营与治理能力的平台产品。**

对应到执行顺序就是：

- **P0**：保命线，先稳
- **P1**：平台基础层
- **P2**：业务平台中层
- **P3**：治理与运营
- **P4**：高级能力
- **P5**：长期产品化与弹性托管

如果按这个顺序做，GoOne 的每一步演进都会比较稳，不会陷入“平台能力还没补齐，就先堆大量业务复杂度”的被动局面。
