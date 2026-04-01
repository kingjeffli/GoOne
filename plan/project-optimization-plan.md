# GoOne 项目优化计划

## 目标

本计划用于整理当前 `GoOne` 项目的性能优化、架构整理和工程体验提升项，优先处理会影响吞吐、延迟、稳定性和后续开发效率的问题。

目标分为 4 类：

1. 降低核心请求链路上的额外开销，提升网关到业务到房间链路的吞吐和响应稳定性。
2. 减少串行瓶颈、无效 goroutine、重复编解码和不必要的跨服务跳转。
3. 统一服务启动、配置、观测和错误处理方式，让新增功能更顺手。
4. 建立基线指标和验证方式，避免优化后只有“感觉变快”而没有数据。

## 当前项目判断

### 1. 核心链路基本清晰，但热点路径存在额外损耗

当前主链路是：

`connsvr -> router -> TransactionMgr -> mainsvr/roomcentersvr -> router -> connsvr`

问题不在“有没有框架”，而在以下组合开销：

1. `TransactionMgr` 以单 goroutine 做入口调度，状态服再叠加 `uid/rid` 串行约束，热点用户和热点房间会形成明显排队。
2. 多处同步 RPC 仍使用“发送消息 + 等响应 + 3 秒定时器”的事务模式，跨服务调用成本偏高。
3. 周期任务存在“tick 再发消息给自己”的做法，增加序列化、路由、日志和调度成本。
4. 网关和事务层日志较重，很多地方会主动调用 `pb.String()`，在高频路径上会产生明显分配和 CPU 消耗。

### 2. 框架层和业务层有可复用能力，但风格尚未完全收敛

项目已经在往 IDL 驱动的 `ssrpc` 迁移，方向是对的，但当前还存在：

1. 服务启动方式重复较多，多个 `app.go` 基本模板相似。
2. `ssrpc` 已有 `Logging/Trace/Metrics` 扩展点，但默认只接了日志，没有真正落地统一指标。
3. 部分模块保留老式事务调用接口，新增需求时开发者很容易继续沿用旧路径。
4. 配置和运行时能力分散在各服务中，缺少统一的 profile、指标、压测、容量检查入口。

### 3. 当前编译健康度一般

执行过一次仅编译不跑用例的检查：

`go test ./... -run TestNonExistent`

结果是主服务包基本可编译，但仓库里仍有一些测试或工具代码已漂移，包括：

1. 测试代码与现有接口不一致。
2. 一些旧测试依赖已不存在的字段或类型。
3. 少数包有格式化/调用方式问题。

这说明项目主路径还能跑，但“工程卫生”不足，后续重构时容易因为缺少可靠回归而放慢速度。

## 核心问题与优化方向

## P0：优先处理

### P0-1 事务入口串行瓶颈过重

状态：已完成（TransactionMgr 已改为分片调度，mainsvr 按 uid 分片，roomcentersvr 按 rid 分片）

相关位置：

- [lib/service/transaction/transaction_mgr_impl.go](D:/WorkCode/GoOne/lib/service/transaction/transaction_mgr_impl.go)
- [src/mainsvr/app.go](D:/WorkCode/GoOne/src/mainsvr/app.go)
- [src/roomcentersvr/app.go](D:/WorkCode/GoOne/src/roomcentersvr/app.go)

现状：

1. 所有包先进入 `TransactionMgr.run()` 单线程调度。
2. `mainsvr` 和 `roomcentersvr` 再依赖 `uid/rid` 锁保证串行处理。
3. `maxUidPendingPacket` 超限后直接丢包。

影响：

1. 单个热点用户或热点房间容易堆积。
2. 长事务会拉长同 uid/rid 的尾延迟。
3. 入口 goroutine 既负责路由，又负责 pending 队列管理，扩展性有限。

建议：

1. 将 `TransactionMgr` 升级为分片调度。
2. `mainsvr` 按 `uid` 分片，`roomcentersvr` 按 `rid` 分片。
3. 每个分片维护自己的事务表和 pending 队列，避免全局单入口调度。
4. `maxUidPendingPacket` 改为可观测指标，不只是错误日志。

预期收益：

1. 热点冲突从全局影响收敛为分片内影响。
2. 更容易做容量规划。
3. 降低尾延迟抖动。

### P0-2 房间 tick 采用“发消息给自己”的方式，路径过重
状态：已完成，参考了 SendPbMsgToMyself 的思路，但把优化下沉到了 router 层

相关位置：

- [src/roomcentersvr/room_mgr/base.go](D:/WorkCode/GoOne/src/roomcentersvr/room_mgr/base.go)
- [src/roomcentersvr/service/inner_ssrpc.go](D:/WorkCode/GoOne/src/roomcentersvr/service/inner_ssrpc.go)
- [src/roomcentersvr/app.go](D:/WorkCode/GoOne/src/roomcentersvr/app.go)

现状：

1. `RoomMgr.Tick()` 每 5 秒遍历 zone。
2. 每个 zone 又通过 `TickByRouterSimple` 走一次内部 ssrpc。
3. 实际是本进程内周期逻辑，却仍走一套消息化入口。

影响：

1. 产生不必要的编码、日志和事务调度。
2. tick 频率升高后会放大系统噪音。
3. 调用栈变长，排查慢 tick 成本高。

建议：

1. 本进程周期调度改为直接调用 `zone.Tick(nowMs)`。
2. 跨进程广播/路由才走 ssrpc。
3. 保留一个统一的调度接口，但区分 `local invoke` 和 `remote invoke`。

预期收益：

1. 减少无意义的消息转发。
2. 降低 tick 相关 CPU 与日志开销。
3. 调试和 profile 更直接。

### P0-3 高频路径日志过重
状态：已完成，`ssrpc.Logging()` 已改为快请求 `debug`、慢请求 `info`、异常 `warn`，并将高频路径中的完整 pb 日志收敛为摘要日志

相关位置：

- [lib/service/router/router.go](D:/WorkCode/GoOne/lib/service/router/router.go)
- [lib/service/transaction/transaction_impl.go](D:/WorkCode/GoOne/lib/service/transaction/transaction_impl.go)
- [lib/service/ssrpc/logging.go](D:/WorkCode/GoOne/lib/service/ssrpc/logging.go)
- [src/mainsvr/role/role.go](D:/WorkCode/GoOne/src/mainsvr/role/role.go)
- [src/connsvr/pack_proc.go](D:/WorkCode/GoOne/src/connsvr/pack_proc.go)

现状：

1. 请求入口和出口有大量 `Infof/Debugf`。
2. 多处主动调用 `req.String()` / `rsp.String()` / `data.String()`。
3. 默认中间件会对每次 ssrpc 请求记录开始和结束日志。

影响：

1. protobuf `String()` 在高并发下会带来额外分配。
2. 业务包越大，日志开销越高。
3. CPU profile 很容易被日志格式化吞掉。

建议：

1. 高频路径默认只打结构化摘要日志，不展开完整 pb。
2. `ssrpc.Logging()` 降为 debug 级或仅记录慢请求。
3. 增加慢调用阈值，例如 `cost > 50ms` 才打 info/warn。
4. 包体日志统一走采样开关，只在排障时打开。

预期收益：

1. 直接减少 CPU 与内存分配。
2. 显著降低日志 IO。
3. 减少线上噪音。

### P0-4 周期任务使用 goroutine 包裹，存在重入风险

相关位置：

- [src/mainsvr/app.go](D:/WorkCode/GoOne/src/mainsvr/app.go)
- [src/roomcentersvr/app.go](D:/WorkCode/GoOne/src/roomcentersvr/app.go)

现状：

1. `OnTick()` 内使用 `safego.Go(...)` 派发实际 tick。
2. 如果上一次 tick 未完成，下一次 tick 仍可能继续启动新 goroutine。

影响：

1. 周期任务重叠。
2. tick 不再具备稳定节奏。
3. 数据竞争和突发 CPU 峰值更难发现。

建议：

1. 周期任务改为显式调度器。
2. 单任务明确“可重入”还是“不可重入”。
3. 对 `RoleMgr.Tick()` 和 `RoomListMgr.Tick()` 至少增加互斥或 running 标记。

## P1：第二阶段处理

### P1-1 网关收包模型过于粗放

相关位置：

- [lib/net/net_mgr/tcp_impl.go](D:/WorkCode/GoOne/lib/net/net_mgr/tcp_impl.go)
- [lib/net/net_mgr/ws_impl.go](D:/WorkCode/GoOne/lib/net/net_mgr/ws_impl.go)
- [src/connsvr/pack_proc.go](D:/WorkCode/GoOne/src/connsvr/pack_proc.go)

现状：

1. 每个包到达后直接 `go t.handler(conn, data)`。
2. WS/TCP 都是“来包即开 goroutine”。

影响：

1. 峰值流量下 goroutine 数暴涨。
2. 登录、重连、心跳等高频命令下调度成本高。
3. 连接级顺序语义依赖上层间接保证，不够直观。

建议：

1. 网关按连接绑定轻量队列，改为单连接串行消费。
2. 登录前后分阶段处理，避免同一连接同时跑多条关键逻辑。
3. 为网关引入连接级背压和慢连接统计。

### P1-2 Role 加载路径存在重复读和并发窗口

相关位置：

- [src/mainsvr/role/role_mgr.go](D:/WorkCode/GoOne/src/mainsvr/role/role_mgr.go)

现状：

1. `obtainRole()` 先查内存，再读 Redis，再次查内存后再写回。
2. 没有单飞控制，同一个 uid 并发首次加载时会重复打 Redis。

影响：

1. 热用户登录/重连时会重复加载。
2. 增加缓存击穿时的后端压力。

建议：

1. 对 `uid` 维度加 singleflight。
2. 将“加载并放入缓存”收敛为单入口。
3. 明确 `Role` 的生命周期和淘汰策略。

### P1-3 玩家数据同步粒度还可以继续压缩

相关位置：

- [src/mainsvr/role/role.go](D:/WorkCode/GoOne/src/mainsvr/role/role.go)

现状：

1. 已有按 section flag 同步的思路。
2. 但部分路径仍会带出较大对象，且日志会打印 `ScSyncUserData.String()`。

建议：

1. 引入 dirty-set 或 patch 模式。
2. 将数据同步与持久化拆开建模。
3. 对 `RoleInfo` 大字段做子对象分组或增量消息。

### P1-4 MySQL 更新存在“先查后写”模式

相关位置：

- [src/mysqlsvr/service/mysql_ssrpc.go](D:/WorkCode/GoOne/src/mysqlsvr/service/mysql_ssrpc.go)

现状：

1. `UpdateRoleInfo()` 先查是否存在，再决定 insert/update。
2. 多数表写入逻辑没有充分利用 upsert。

影响：

1. 多一次往返。
2. 并发下更容易出现竞争窗口。

建议：

1. 改成 `INSERT ... ON DUPLICATE KEY UPDATE`。
2. 对 Texas 相关表统一整理 upsert 策略。
3. 将幂等、去重和时间戳检查下沉到 DAO 层。

## P2：工程与优雅性改造

### P2-1 统一服务启动模板

状态：已完成（已抽出统一 bootstrap，六个服务启动顺序统一为 load config -> init logger -> init deps -> register handlers -> start runtime，并内置 admin server 的 healthz/readyz/metrics/pprof 与 `web_svr` 的优雅关闭）

相关位置：

- [src/connsvr/app.go](D:/WorkCode/GoOne/src/connsvr/app.go)
- [src/mainsvr/app.go](D:/WorkCode/GoOne/src/mainsvr/app.go)
- [src/infosvr/app.go](D:/WorkCode/GoOne/src/infosvr/app.go)
- [src/mysqlsvr/app.go](D:/WorkCode/GoOne/src/mysqlsvr/app.go)
- [src/roomcentersvr/app.go](D:/WorkCode/GoOne/src/roomcentersvr/app.go)
- [src/web_svr/app.go](D:/WorkCode/GoOne/src/web_svr/app.go)

建议：

1. 抽一个统一的 service bootstrap。
2. 标准化 `load config -> init logger -> init deps -> register handlers -> start runtime`。
3. 把 `pprof`、metrics、graceful shutdown、health check 做成默认能力。

### P2-2 真正接入 metrics，而不是只保留扩展点

状态：已完成（已为 `ssrpc`、`router`、`TransactionMgr`、Redis、MySQL、网关连接接入 Prometheus 指标，并补充 Grafana dashboard 模板 `deploy/observability/grafana/goone-runtime-dashboard.json`）

相关位置：

- [lib/service/ssrpc/metrics.go](D:/WorkCode/GoOne/lib/service/ssrpc/metrics.go)
- [lib/service/ssrpc/default_mw.go](D:/WorkCode/GoOne/lib/service/ssrpc/default_mw.go)

建议：

1. 为 `ssrpc`、`router`、`TransactionMgr`、Redis、MySQL、网关连接数接 Prometheus 指标。
2. 指标至少覆盖 QPS、P50/P95/P99、错误率、超时数、pending 队列长度。
3. 关键指标接到 dashboard，而不是只存在代码里。

### P2-3 配置体验可以更统一

相关位置：

- [common/gconf/config.go](D:/WorkCode/GoOne/common/gconf/config.go)

建议：

1. 拆分公共配置与服务私有配置文档。
2. 增加配置校验器，启动前提前失败。
3. 统一 `yaml/json` tag 风格。
4. 将运行依赖、容量参数、调试开关显式区分。

### P2-4 路由策略需要从“可用”升级到“可控”

相关位置：

- [module/misc/constant.go](D:/WorkCode/GoOne/module/misc/constant.go)
- [lib/service/svrinstmgr/svr_inst_mgr.go](D:/WorkCode/GoOne/lib/service/svrinstmgr/svr_inst_mgr.go)

建议：

1. 为每类服务明确使用普通 hash 还是一致性 hash。
2. 给路由策略增加压测结果和使用说明。
3. 热点房间服优先使用一致性 hash，减少扩缩容数据抖动。

## 建议实施顺序

### 第一阶段：先拿到收益和数据

1. 下调高频日志级别，去掉大对象 `String()`。
2. 给 `TransactionMgr`、网关、房间 tick 增加基础指标。
3. 去掉 `roomcentersvr` 本地 tick 的自发消息链路。
4. 给 `RoleMgr` 首次加载加 singleflight。

### 第二阶段：改造并发模型

1. 网关改为连接级串行处理模型。
2. `TransactionMgr` 改为分片调度。
3. 周期任务引入防重入调度器。

### 第三阶段：提升优雅性和可维护性

1. 统一 bootstrap。
2. DAO 层整理 upsert 和批量接口。
3. 完善配置校验、指标面板和压测脚本。
4. 清理老测试和失效工具代码。

## 验证指标

建议优化时同步建立以下观测项：

1. `connsvr` 每秒收包数、活跃连接数、goroutine 数、单连接排队长度。
2. `TransactionMgr` 当前事务数、pending 队列长度、超时数、丢包数。
3. `mainsvr` 热门 uid 请求耗时分布。
4. `roomcentersvr` 每次 tick 的耗时、room/zone 数量、tick 重入次数。
5. Redis 和 MySQL 的 P95/P99 延迟。
6. 日志吞吐量和磁盘写入速率。

## 具体落地建议

### 短期一周内

1. 完成日志瘦身。
2. 接入最小指标集。
3. 改掉房间本地 tick 自发消息。
4. 增加 `RoleMgr` singleflight。

### 中期两到三周

1. 网关改为连接队列模型。
2. `TransactionMgr` 完成分片化设计和迁移。
3. MySQL 写入改为 upsert。

### 中长期

1. 统一服务框架。
2. 建立标准压测场景。
3. 完成旧事务接口收敛，新增功能只走 `ssrpc` 标准路径。

## 备注

这份计划以当前仓库代码为依据，重点偏向运行性能和开发体验，不是一次性大重构清单。建议先做能够快速观察收益的 P0/P1 项，再决定是否推进更深层的架构演进。
