# 01. GoOne 架构总评审

## 1. 结论摘要

`GoOne` 当前已经具备一个 **可运行、可扩展、偏工程化** 的分布式游戏后端骨架，尤其在以下方面明显优于很多“只写业务、不做平台层”的中小型项目：

- 有明确的服务切分与启动约定；
- 有统一的 `bootstrap` 装配层；
- 有 `router + transaction` 这条比较清晰的消息驱动主链路；
- 有 `ssrpc` 统一抽象 SSPacket / HTTP / WS / gRPC；
- 有基础的 `/healthz`、`/readyz`、`/metrics`、`pprof` 管理面；
- 有一定的单元测试覆盖，不完全依赖手工联调。

但如果从“**长期可维护的游戏后端平台**”而不是“可用框架”标准看，当前架构还停留在 **强框架、弱平台；强运行时、弱产品化；强主链路、弱边角治理** 的阶段。

一句话评价：

> **它已经像一个‘能带业务跑起来的游戏服务框架’，但还没完全长成一个‘可持续交付、可运营、可观测、可扩展的游戏平台内核’。**

---

## 2. 当前架构的主路径

### 2.1 启动模型

服务统一遵循：

- `src/<service>/main.go`：`flag.Parse()` -> `application.Init(newApp())` -> `application.Run()`
- `src/<service>/app.go`：通过 `bootstrap.NewServiceApp(...)` 填充：
  - `LoadConfig`
  - `InitDeps`
  - `RegisterHandlers`
  - `StartRuntime`
  - `OnProc`
  - `OnTick`
  - `OnExit`

这套模型的优点是：

- 启动入口比较一致；
- 初始化顺序被约束住；
- 配置加载、依赖初始化、处理器注册、运行时启动被显式分阶段；
- 比散落在 `main.go` 中的大段逻辑更可维护。

代表文件：

- `lib/service/application/app.go`
- `lib/service/bootstrap/app.go`
- `src/mainsvr/app.go`
- `src/connsvr/app.go`
- `src/web_svr/app.go`

### 2.2 主业务链路

常规链路大致是：

1. 客户端接入 `connsvr`
2. `connsvr` 收到 CSPacket/WS 消息后，转为服务间消息
3. 进入 `lib/service/router/router.go`
4. 根据 `module/misc.ServerRouteRules` 和服务实例发现结果路由
5. 投递到目标服务 `globals.TransMgr`
6. 目标服务使用 `transaction` 串行处理

这是一个典型的 **消息驱动 + 串行事务执行** 设计，适合：

- 玩家上下文较强的业务；
- 强顺序要求的角色状态修改；
- 通过 `uid/routerId` 做天然分片的玩法逻辑。

### 2.3 例外路径：`web_svr`

`web_svr` 明显是例外：

- 不走标准 bus 主事务链；
- 直接启动 Gin HTTP；
- 可选挂 gRPC listener；
- 更像“管理面/API 网关/后台聚合层”。

这本身不是问题，反而是合理的。但长期要注意：

- `web_svr` 不能无限膨胀成“万能入口”；
- HTTP 管理面、后台 API、外部集成 API、运营工具 API 最终应进一步分层；
- 否则 `web_svr` 会成为新的单体瓶颈。

---

## 3. 当前架构的优点

## 3.1 `bootstrap` 层是当前最值得保留的骨架

`lib/service/bootstrap/app.go` 的设计已经具备比较好的工程感：

- 初始化阶段被拆成明确步骤；
- `AdminServer` 与服务生命周期对接；
- `alive/ready` 状态有原子布尔值；
- 初始化失败时有清理逻辑；
- 管理端口统一暴露 `/healthz`、`/readyz`、`/metrics`、`pprof`。

这意味着项目已经有了“平台化入口”的雏形。

### 评价

这是 GoOne 当前最接近“现代服务治理”的部分，建议把它继续做成全项目统一标准，而不是只当启动辅助层。

---

## 3.2 `ssrpc` 是现代化程度很高的一层

`lib/service/ssrpc` 已经明显超出传统游戏服的“命令号 + switch-case”模式：

- 有 `Dispatcher` 统一注册中心；
- 有 `Middleware` 链；
- 同时覆盖：
  - SSPacket
  - HTTP/Gin
  - WS
  - gRPC
- 已带 Prometheus metrics；
- `web_svr` 已接 gRPC health/reflection。

这层的价值很大：

- 统一了 transport 差异；
- 降低服务接入成本；
- 给后续统一鉴权、签名、限流、追踪提供了挂点。

### 评价

如果 GoOne 未来要继续演进，`ssrpc` 应当是平台能力承载层之一。

---

## 3.3 `router + transaction` 的核心执行模型是成立的

`lib/service/router/router.go` + `lib/service/transaction/*` 构成了真正的业务主干。

它的优点：

- 玩家/房间等 key 维度天然适合串行执行；
- 减少锁竞争；
- 逻辑状态更容易推理；
- 与游戏服“按角色、按房间串行化”的经典模型一致。

尤其 `mainsvr` 和 `roomcentersvr` 已经使用 `InitAndRunWithConfig` + `ShardCount`，说明你们已经开始把单线程模型升级为“**按 key 分片串行**”。

### 评价

方向是对的，甚至是整个项目最有“游戏后端味道”的部分。

---

## 4. 当前架构的主要问题

## 4.1 `application.Run()` 存在空转问题，是最直接的性能瑕疵

在 `lib/service/application/app.go` 中，主循环是：

- 持续 `for {}`
- 每轮检查信号
- 每轮 `datetime.Tick()`
- 每轮执行 `app.loopOnce()`
- 只有当 `idleLoopCnt > 1000` 时才 `Sleep(5ms)`

而现在绝大多数服务 `OnProc()` 直接返回 `true`，例如：

- `src/mainsvr/app.go`
- `src/connsvr/app.go`
- `src/infosvr/app.go`
- `src/mysqlsvr/app.go`
- `src/roomcentersvr/app.go`
- `src/web_svr/app.go`

这意味着：

- 主循环并没有等待事件；
- `Run()` 本质上在高频轮询；
- 在服务空闲时会无意义消耗 CPU；
- Tick 触发判断也依赖忙轮询。

### 影响

- 空闲 CPU 偏高；
- 云资源成本放大；
- 在多实例环境下，空闲损耗会累积得很明显；
- 这类问题在线上最容易被忽视，因为“功能上没错”。

### 建议

把主循环改成 **事件驱动 + 定时器驱动** 的模式，至少应做到：

- 使用 `ticker` 驱动 Tick；
- 使用信号 channel 驱动退出；
- `OnProc` 如果不是必须轮询，应改为阻塞等待或干脆移除；
- 明确哪些服务真的需要 `OnProc` 持续执行。

这条建议优先级非常高。

---

## 4.2 生命周期“能启动”，但“不能很好地收口”

虽然 `bootstrap` 已经有 `OnExit()`，但整个系统的可关闭性并不完整：

- `router` 没有统一的 close/shutdown 合同；
- `transaction` 没有对外统一 shutdown；
- bus 接口 `IBus` 没有 `Close()`；
- 服务发现 watcher 由 `svrinstmgr.Close()` 管，但没有被普遍纳入服务退出路径；
- 事务分片 goroutine 缺乏标准停止机制；
- 一些后台 goroutine 只启动，不收口。

### 结果

项目当前更像“进程式服务”，不是“生命周期完整的服务组件”。

### 建议

建立统一的 shutdown contract：

- `router.Close()`
- `transaction.Shutdown(ctx)`
- `bus.Close()`
- `ServiceApp` 在 `OnExit` 前后按顺序收口
- 所有后台 goroutine 都必须有停止信号来源

这会显著提升：

- 灰度重启可靠性；
- 单测/集成测试可控性；
- 线上问题排查效率；
- 未来容器化/自动扩缩容的友好度。

---

## 4.3 配置层在进步，但热更新闭环还没形成

`common/gconf/config.go` 已经做了不少正确的事情：

- 支持 grouped config；
- 对 legacy flat fields 做兼容；
- 有 normalize + validate；
- 有不同服务配置约束。

这是好事。

但目前还有几个明显缺口：

1. **配置结构演进和运行时热更新没有完全闭环**
   - `application.OnReload()` 只是重新 `loadConfig()`；
   - 但绝大多数依赖初始化、运行时组件、logger、网络 listener 并没有真正跟着热更新。

2. **配置中心能力存在，但没有成为全局主路径**
   - 仓库内有 `lib/contrib/config/*`；
   - 但业务服务普遍还是“启动时读取一次 + 局部 remote gamedata”。

3. **配置边界不够清晰**
   - 哪些字段支持热更，哪些必须重启，文档和代码契约都不够明确。

### 建议

把配置分成三类：

- 冷配置：必须重启生效
- 暖配置：可 reload 生效
- 热配置：watch 自动生效

并在服务内显式标注支持范围。

---

## 4.4 事务模型方向正确，但还有几个“平台级缺口”

`transaction` 层已经具备：

- shard 化；
- pending queue；
- drop 统计；
- active/pending/queue metrics。

但仍有以下问题：

### 1）超时硬编码

在 `lib/service/transaction/transaction_impl.go`：

- `waitRsp(..., 3*time.Second, ...)`

在 `transaction_mgr_impl.go`：

- `packet.SendToChan(..., 3*time.Second)`

这会导致：

- 不同业务无法差异化 timeout；
- 排查线上慢链路时缺乏策略空间；
- 与配置层/方法描述层脱节。

### 2）分片策略不一致

- `mainsvr`、`roomcentersvr` 已经支持 shard；
- `connsvr`、`infosvr`、`mysqlsvr` 还是老式单 shard `InitAndRun(...)`。

不一定要全部改成多 shard，但应该明确：

- 哪些服务就是轻量单线程；
- 哪些服务需要按 key 并行；
- 哪些服务未来要支持动态 shard。

### 3）缺乏背压治理分层

目前主要是：

- 超限 drop
- 指标计数

但还缺：

- 按 cmd 的限流/隔离；
- 大户 UID 限流；
- 慢事务告警；
- 分片热点识别；
- 针对 queue 堆积的自动保护策略。

---

## 4.5 `ssrpc` 很先进，但 tracing 只做了接口，没有做产品

`lib/service/ssrpc/trace.go` 当前只有：

- `TraceProvider` interface
- `TraceWith(...)`
- 默认 no-op

这意味着：

- 设计上已经留出了 tracing 钩子；
- 但没有 concrete implementation；
- 没有和 OpenTelemetry、Jaeger、Tempo、Zipkin 之类打通；
- 没有 trace/span id 在日志、metrics、上下游请求中贯通。

### 结果

线上出问题时：

- 你能看 metrics；
- 你能看日志；
- 但看不到一条跨服务调用链到底在哪个 hop 变慢。

对于微服务游戏后端，这是非常关键的缺口。

---

## 4.6 工具链和仓库真实结构存在漂移

这是一个很现实的问题，而且会直接伤害“易用性”和“新人上手体验”。

### 已观察到的漂移

1. `main.sh` 中一些路径仍按 `env/*` 推断；
2. 仓库实际环境目录是 `etc/env/*`；
3. `build.sh` 仍保留很多旧服务目标；
4. `build.sh` 没覆盖当前活跃的 `roomcentersvr`；
5. `build.sh` 和 `AGENTS.md`/README 中的推荐入口并不完全一致。

### 影响

- 对外文档和实际仓库不一致；
- 脚手架和构建体验不稳定；
- “看起来有统一入口”，但实际使用时容易踩坑。

这类问题不会导致业务 bug，但会长期消耗团队效率。

---

## 5. 哪些设计应该保留，不要轻易推翻

在优化时，不建议推翻以下骨架：

## 5.1 保留 `bootstrap.Options` 这一层

它已经天然适合承接：

- 服务阶段化初始化
- 生命周期扩展点
- 管理面统一化
- 后续 shutdown contract
- 统一观测接入

## 5.2 保留 `router + transaction` 的业务主链

尤其是：

- 按 `uid/routerId` 串行执行；
- 事务内状态变更；
- 服务间用 bus 解耦。

这是游戏后端的合理主模型，不建议追求“把所有逻辑都改成完全无状态 HTTP/gRPC”。

## 5.3 保留 `ssrpc` 的多 transport 统一抽象

不要回退到：

- HTTP 一套逻辑
- WS 一套逻辑
- SSPacket 一套逻辑
- gRPC 再一套逻辑

`Dispatcher + Middleware + Context` 这套思路是正确的，应该继续扩。

---

## 6. 从“框架”走向“平台”的关键差异

当前 GoOne 已经像：

- 一个可复用的游戏后端开发框架；
- 一个能承载多服务和多玩法的内核。

但要成为更成熟的平台，还需要补齐：

- 生命周期治理
- 可观测链路
- 运维标准化
- CI/CD
- LiveOps
- 安全与审计
- 自动化测试分层
- 平台级能力目录

也就是说，当前更像“**runtime + infra framework**”，还不是“**game backend platform**”。

---

## 7. 总体建议优先级

## P0：立即处理

1. 修正 `application.Run()` 的忙轮询问题
2. 建立 router/transaction/bus 的 shutdown contract
3. 统一 `main.sh` / `build.sh` / `etc/env` 的真实路径与入口
4. 为事务层 timeout / queue / drop 补充配置化能力

## P1：中期重构

1. 补齐 tracing（至少 OpenTelemetry 接口实现）
2. 给所有服务定义明确的运行模型：单 shard / 多 shard / 无事务型
3. 规范 config reload 能力和热更新边界
4. 建立 CI：fmt、vet、test、proto check、build

## P2：平台化建设

1. 后台运营能力（RBAC、审计、配置变更留痕）
2. 事件流与异步任务体系
3. 平台级会话/在线状态/presence 能力
4. 更完整的匹配、房间编排与弹性伸缩能力

---

## 8. 最终判断

如果只问“这个项目值不值得继续演进”，答案是：**值得**。

因为它最核心的几块——

- 服务分层
- 启动规范
- 路由模型
- 事务模型
- 多传输统一 runtime

都不是随便拼凑出来的，而是已经形成了一个比较像样的骨架。

但如果只问“它现在是否已经足够成熟到接近 Nakama / Photon / PlayFab 风格的平台能力”，答案是：**还差一大段工程化与平台化建设**。

所以正确策略不是“重写”，而是：

> **保留正确的主干，优先修复生命周期与运行时缺陷，再逐步把它从框架升级成平台。**

