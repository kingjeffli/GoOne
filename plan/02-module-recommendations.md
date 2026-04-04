# 02. 模块逐项优化建议

本文按“**当前职责 / 现状优点 / 关键问题 / 建议动作 / 优先级**”来分析。

---

# 1. `lib/service/application`

## 当前职责

负责服务主循环与系统信号处理，是所有服务统一入口的最外层运行框架。

关键文件：

- `lib/service/application/app.go`
- `lib/service/application/sig.go`
- `lib/service/application/sig_windows.go`

## 现状优点

- 入口统一，所有服务一致；
- 暴露 `OnInit / OnReload / OnProc / OnTick / OnExit` 扩展点；
- 非 Windows 支持 `SIGUSR1` reload。

## 关键问题

1. `Run()` 空转严重；
2. `OnProc` 语义模糊，很多服务直接 `return true`；
3. reload 只对配置重读有意义，未形成服务级热更体系；
4. Windows 上 reload 能力不对齐；
5. 缺少 context 驱动的优雅退出。

## 建议动作

### 建议 1：主循环改成“定时器 + 事件”模型

建议重构为：

- `signalCh` 处理退出/重载；
- `ticker` 处理 Tick；
- 可选 `procCh` 或阻塞式 worker 处理内部事件；
- 避免 busy-spin。

### 建议 2：收紧 `OnProc` 语义

建议定义三种服务：

- **event-driven**：不需要 `OnProc`
- **poll-driven**：必须有阻塞等待策略
- **hybrid**：tick + 事件混合

### 建议 3：把退出改成可测试、可观测的 shutdown 过程

不要在底层直接 `os.Exit(0)` 结束全部逻辑；应尽量：

- 先触发 `OnExit`
- 等待组件收口
- 最后由主函数退出

## 优先级

- **P0**

---

# 2. `lib/service/bootstrap`

## 当前职责

统一服务初始化、logger 初始化、admin server 启动、分阶段执行服务依赖装配。

关键文件：

- `lib/service/bootstrap/app.go`
- `lib/service/bootstrap/admin.go`

## 现状优点

- 结构清晰；
- 已具备 `alive/ready`；
- 有标准管理面；
- 初始化失败有清理。

## 关键问题

1. 仍偏“初始化 orchestrator”，不是完整“生命周期容器”；
2. 对底层组件没有统一关闭协议；
3. `OnReload()` 只负责重读配置，没协调依赖重载；
4. 缺少组件注册表，不便输出组件状态。

## 建议动作

### 建议 1：引入组件生命周期接口

例如：

- `Start(ctx)`
- `Ready() bool`
- `Shutdown(ctx) error`
- `Reload(ctx) error`

然后 `ServiceApp` 统一管理。

### 建议 2：扩展 admin server

当前已有：

- `/healthz`
- `/readyz`
- `/metrics`
- `/debug/pprof/*`

建议再加：

- `/info`：版本、构建信息、启动时间
- `/components`：依赖组件状态
- `/configz`：脱敏后当前运行配置快照
- `/debug/transmgr`：事务统计

### 建议 3：服务初始化增加“阶段耗时日志”

线上排查启动慢时非常有用。

## 优先级

- **P1**

---

# 3. `common/gconf`

## 当前职责

提供统一配置结构、兼容旧字段、服务级 validate。

关键文件：

- `common/gconf/config.go`

## 现状优点

- grouped config 方向正确；
- 兼容 legacy flat fields；
- validate 明确；
- `websvr`、`mainsvr`、`roomcentersvr` 等都已开始接入新结构。

## 关键问题

1. 热更新契约不明确；
2. 配置中心能力和业务加载路径未统一；
3. 某些默认值分散在代码中，不在配置里；
4. 配置结构增长后可能逐渐过重。

## 建议动作

### 建议 1：把所有重要运行参数配置化

当前仍有不少硬编码，典型包括：

- transaction timeout
- queue limits
- 某些 goroutine/tick 行为

### 建议 2：按能力拆分配置域

例如：

- runtime
- transport
- dependency
- capacity
- observability
- security
- feature_flags

### 建议 3：定义 reload 支持矩阵

文档和代码要一致说明：

- 哪些配置热生效
- 哪些配置 reload 生效
- 哪些配置必须重启

## 优先级

- **P1**

---

# 4. `lib/service/router`

## 当前职责

负责服务间消息发送、接收、本地短路、自身 bus 通信与 metrics。

关键文件：

- `lib/service/router/router.go`
- `lib/service/router/metrics.go`

## 现状优点

- 路由链路简单直接；
- 支持 local short-circuit；
- 已有 Prometheus 指标；
- 与服务发现层解耦。

## 关键问题

1. 无统一 close/shutdown；
2. 错误处理仍偏字符串拼装；
3. 广播与批量发送缺少背压/失败策略；
4. 与 bus 生命周期绑定不够清晰；
5. 缺少“当前路由命中行为”的调试面板。

## 建议动作

### 建议 1：补生命周期管理

增加：

- `InitAndRun(...)` 对应 `Close()`
- bus 关闭时停止收包
- 释放 discovery watcher

### 建议 2：增强调试与治理能力

建议增加：

- 最近路由错误统计
- 各 svrType 实例分布
- consistent hash ring 快照
- 本地短路比例

### 建议 3：引入按 cmd 维度的可治理策略

例如：

- 允许广播的 cmd 白名单
- 慢消息/大消息统计
- 按 cmd 的 QoS 等级

## 优先级

- **P1**

---

# 5. `lib/service/svrinstmgr`

## 当前职责

维护在线服务实例、路由规则、consistent hash ring。

关键文件：

- `lib/service/svrinstmgr/svr_inst_mgr.go`

## 现状优点

- 支持多种 registry backend；
- 已支持 consistent hash；
- 有 benchmark，说明热路径性能意识是有的。

## 关键问题

1. watcher 生命周期未强绑定服务退出；
2. 缺少实例状态分级（仅在线/不在线还不够）；
3. 路由策略仍以固定枚举为主，扩展性一般；
4. 对容量、负载、版本、区域、标签等维度支持不够。

## 建议动作

### 建议 1：实例元数据升级

未来建议注册更多 metadata：

- version
- region/zone
- role
- capacity tags
- feature tags
- drain state

### 建议 2：支持“摘流 / drain”语义

对优雅发布、灰度、缩容非常重要。

### 建议 3：把路由从“纯规则枚举”提升到“策略对象”

有助于未来支持：

- canary
- 区域亲和
- 版本路由
- 玩家群体分流

## 优先级

- **P2**

---

# 6. `lib/service/transaction`

## 当前职责

GoOne 最核心的事务执行层；负责按 key 串行处理请求、等待响应、管理分片与队列。

关键文件：

- `lib/service/transaction/transaction_mgr_impl.go`
- `lib/service/transaction/transaction_impl.go`
- `lib/service/transaction/transaction_config.go`
- `lib/service/transaction/metrics.go`

## 现状优点

- 已有 shard 模型；
- 已有 pending queue；
- 已有 dropped/active/pending metrics；
- 核心逻辑贴合游戏服“按玩家/房间串行”的经典模型。

## 关键问题

1. timeout 硬编码；
2. shutdown 缺失；
3. 某些服务仍是单 shard；
4. 缺少热点 key 识别；
5. 缺少按 cmd 的资源治理；
6. 对长事务、慢事务缺少独立监控面。

## 建议动作

### 建议 1：配置化 timeout 与 queue 策略

至少把以下能力配置化：

- `wait_rsp_timeout`
- `dispatch_timeout`
- `max_pending_per_key`
- `max_trans`
- `shard_count`

### 建议 2：加 `Shutdown(ctx)`

要做到：

- 停止接新包
- drain 队列或快速失败
- 等待在途事务结束
- 输出 summary metrics/log

### 建议 3：增强热点可观测性

建议暴露：

- busiest shard
- top pending keys
- top slow cmds
- drop reason 分布

### 建议 4：服务分层治理

建议明确：

- `mainsvr`、`roomcentersvr`：重点优化多 shard
- `infosvr`：视请求量考虑升级
- `mysqlsvr`：如果更多像数据 RPC，可能需要单独的异步/批处理模式
- `connsvr`：事务仅用于少量内部命令，不一定需要和逻辑服同模型

## 优先级

- **P0 / P1**

---

# 7. `lib/service/ssrpc`

## 当前职责

统一不同 transport 的处理模型，是未来平台层最关键的承载点之一。

关键文件：

- `lib/service/ssrpc/dispatcher.go`
- `lib/service/ssrpc/server.go`
- `lib/service/ssrpc/http.go`
- `lib/service/ssrpc/grpc.go`
- `lib/service/ssrpc/ws.go`
- `lib/service/ssrpc/metrics.go`
- `lib/service/ssrpc/trace.go`

## 现状优点

- 统一注册中心非常好；
- Middleware 抽象合理；
- transport 统一上下文良好；
- metrics 已有；
- `TraceProvider` 留了扩展点；
- `UIDLock` 设计简洁。

## 关键问题

1. tracing 只是接口，没有产品实现；
2. 缺少统一 request-id / trace-id 透传；
3. 中间件能力还缺限流、熔断、审计、AB 标记等；
4. auth/sign 已有接口，但跨 transport 策略不够统一；
5. 没有正式的 OpenAPI/HTTP 契约治理。

## 建议动作

### 建议 1：优先接入 OpenTelemetry

建议做到：

- HTTP 入站 span
- gRPC 入站 span
- SSPacket/WS 内部 span
- 跨服务 trace context 透传
- logger 注入 trace_id/span_id

### 建议 2：中间件体系平台化

建议新增标准中间件：

- request id
- rate limit
- audit log
- access log 标准化
- circuit breaker（外部依赖调用侧）
- feature flag injection

### 建议 3：定义 transport 能力基线

例如：

- 哪些功能 HTTP/gRPC/WS/SSPacket 都必须支持
- 哪些功能仅某 transport 支持
- 错误码到 HTTP/gRPC status 的映射策略

## 优先级

- **P1**

---

# 8. `lib/service/bus`

## 当前职责

抽象多种消息总线后端：RabbitMQ/NSQ/NATS/Kafka/RocketMQ。

## 现状优点

- 抽象层存在；
- 多 backend 支持较丰富；
- 比很多项目“MQ 写死”强很多。

## 关键问题

1. `IBus` 没有统一关闭接口；
2. 发送/接收语义和 QoS 能力不统一；
3. 重试逻辑分散在实现类；
4. 缺少能力矩阵与推荐 backend 策略。

## 建议动作

### 建议 1：给 `IBus` 增加生命周期与状态接口

例如：

- `Close() error`
- `Healthy() bool`
- `Stats() BusStats`

### 建议 2：输出 backend 能力矩阵

明确：

- 适合开发环境的 backend
- 适合生产环境的 backend
- 顺序性、吞吐、延迟、部署复杂度差异

### 建议 3：统一重试/退避策略抽象

不要把 backoff 只写在个别实现里。

## 优先级

- **P1**

---

# 9. `src/connsvr`

## 当前职责

网关层，负责 TCP/WS 接入、客户端消息收发、少量内部 RPC。

## 现状优点

- 网关职责相对清晰；
- 接入层和业务层已分离；
- 与后端通过 router 对接。

## 关键问题

1. 仍然偏“转发网关”，会话治理能力不够；
2. `onWebSocketPacket` / `onTcpPacket` 中的 uid 依赖比较原始；
3. 缺少更完整的连接态、踢人、重连、会话恢复策略；
4. 限流、防刷、恶意包治理不明显；
5. `OnProc` 无实际作用。

## 建议动作

- 引入 session manager 的正式模型；
- 增加连接级与 UID 级限流；
- 增加登录前/登录后双状态机；
- 增加断线重连、session resume、踢线策略统一化；
- 统一 TCP 与 WS 行为策略。

## 优先级

- **P1**

---

# 10. `src/mainsvr`

## 当前职责

玩家主逻辑服，是最核心的业务服务之一。

## 现状优点

- 已使用 sharded transaction；
- 角色逻辑集中度高；
- `gamedata`、`redis`、`idgen` 等依赖装配比较清晰。

## 关键问题

1. `OnTick` 每分钟 `safego.Go(globals.RoleMgr.Tick)`，但调度治理较弱；
2. 角色缓存、持久化节流、脏数据同步缺少更明确的策略面板；
3. 角色域逻辑大概率会继续膨胀；
4. 玩家状态同步和跨服务边界未来容易失控。

## 建议动作

- 把角色域拆成更清晰的 domain modules；
- 给 RoleMgr 增加运行时 stats/export；
- 把 role persistence / sync policy 完整配置化；
- 对慢角色、频繁脏写用户做专项观测。

## 优先级

- **P1**

---

# 11. `src/infosvr`

## 当前职责

轻量信息/缓存服务。

## 现状优点

- 职责相对单一；
- 结构清晰；
- 依赖少。

## 关键问题

1. 仍沿用单 shard 事务；
2. 长期容易沦为“杂项查询服务”；
3. 缺少清晰的数据所有权边界。

## 建议动作

- 明确 infosvr 只承载什么，不承载什么；
- 若请求量上涨，考虑多 shard；
- 给外部暴露的数据契约做稳定化治理。

## 优先级

- **P2**

---

# 12. `src/mysqlsvr`

## 当前职责

数据持久化服务，依赖 ORM。

## 现状优点

- 持久化入口集中；
- 配置校验已有保障；
- 服务职责清晰。

## 关键问题

1. 事务模型与数据访问模式之间还不够精细；
2. 单 shard 事务未必适合长期数据访问压力；
3. 缺少批处理、异步写、写入队列等模式；
4. 缺少 SQL 慢查询治理与连接池观测。

## 建议动作

- 输出 MySQLSvr 的访问模型：同步 RPC / 异步队列 / 批量写；
- 增加 ORM stats、慢 SQL metrics；
- 考虑将纯存储型能力从业务型事务模型中抽离。

## 优先级

- **P2**

---

# 13. `src/roomcentersvr`

## 当前职责

房间中心、房间列表、房间 tick、部分 AI 初始化。

## 现状优点

- 已使用 sharded transaction；
- 明显具备房间中心服务定位；
- 是最接近“玩法编排层”的服务。

## 关键问题

1. 房间与 match/seat/session/presence 的平台化能力还不完整；
2. Tick 与房间调度还偏手工；
3. 未来如果玩法增多，房间中心可能既做中心又做业务，边界容易模糊；
4. 缺少房间生命周期、房态、容量、摘流治理。

## 建议动作

- 抽象房间生命周期状态机；
- 抽象 match / allocate / assign / recycle 流程；
- 增加房间级 metrics：创建数、销毁数、空房率、满房率、平均存活时长；
- 明确 room center 与 game room 实例的边界。

## 优先级

- **P1 / P2**

---

# 14. `src/web_svr`

## 当前职责

HTTP 管理接口与可选 gRPC 对外服务。

## 现状优点

- 已具备 Gin + gRPC 双入口；
- 已有签名校验接入；
- 已支持 gRPC health/reflection；
- 是对外能力聚合的良好试验场。

## 关键问题

1. 很容易膨胀成万能服务；
2. 后台管理、外部开放 API、内部调试 API 可能混在一起；
3. 缺少 RBAC、审计、操作留痕；
4. 依赖 Redis / Sign / Rest 等较多，边界要更严格。

## 建议动作

- 把路由按角色拆分：admin / public / internal；
- 引入 RBAC、审计日志；
- 补 OpenAPI/接口治理；
- 保持 `web_svr` 作为组合层，不承载核心状态机。

## 优先级

- **P1**

---

# 15. `tools/cmd/scaffold` 与工具链

## 当前职责

用于新服务脚手架和 proto 生成。

## 现状优点

- `genproto` 已经比 shell 脚本更现代；
- scaffold 至少提供了初始骨架。

## 关键问题

1. scaffold 仍偏 legacy cmd_handler 骨架；
2. 没完全体现当前 IDL-first 主路径；
3. 对配置、admin、测试、CI 生成支持不足。

## 建议动作

- scaffold 默认按 `ssrpc + bootstrap + admin + config` 生成；
- 自动生成最小测试文件；
- 自动生成推荐 `app.go` 模板，而不是留太多 TODO 占位；
- 输出与 `AGENTS.md` 一致的服务接入方式。

## 优先级

- **P2**

---

# 16. 总结：最值得优先改的模块

## 第一梯队

1. `lib/service/application`
2. `lib/service/transaction`
3. `lib/service/router`
4. `main.sh` / `build.sh` / 目录入口一致性

## 第二梯队

1. `lib/service/ssrpc`
2. `common/gconf`
3. `src/web_svr`
4. `src/roomcentersvr`

## 第三梯队

1. `src/connsvr`
2. `src/mysqlsvr`
3. `tools/cmd/scaffold`
4. 更高级的平台服务建设

最终判断：

> GoOne 现在最需要的不是“加更多业务功能”，而是先把 **运行时稳定性、生命周期、观测与平台边界** 再做扎实一层。

