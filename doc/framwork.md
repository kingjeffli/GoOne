# GoOne 公共模块与框架规范说明

## 1. 文档目的

本文面向两类读者：

- 项目开发者：快速建立对公共模块、服务结构、工具链和协作约定的整体认知。
- 自动化 agent：在改代码、补文档、排查问题前，先理解什么是框架层、什么是业务层、什么是生成产物、什么是历史遗留。

本文以当前仓库代码树为准，`readme.md` 和历史文档只作为补充背景。若文档描述与代码不一致，请优先相信代码，再回头修正文档。

`lib/` 的文件说明以“项目自有源码”为主。以下内容不逐个展开：

- `lib/contrib/protoc/`：内置的 `protoc 30.1` 官方分发包。
- `lib/util/deps/protoc/`：旧版 `protoc 3.11.4` 依赖包。
- `*_test.go`、局部 `README.md`：测试或补充文档，不是运行时主路径。

## 2. 一页结论

GoOne 是一套基于 Go 的微服务游戏服务端框架。当前仓库可以粗分为四层：

- `lib/`：框架内核，负责生命周期、事务分发、服务间路由、Bus、网络层、配置/注册中心适配、数据库适配、通用工具。
- `common/` + `module/`：项目级公共模块。前者偏配置/配置数据/生成数据仓库，后者偏业务侧的公共常量和工具函数。
- `src/`：具体服务实现，每个服务都以 `application.Init(...) + application.Run()` 驱动。
- `tools/` + `main.sh` + `deploy/` + `env/`：生成工具、测试工具、构建入口、环境管理、部署脚本。

对开发者和 agent 最重要的事实有四条：

1. 当前仓库没有顶层 `api/` 目录，README 中提到的“API 层”在当前代码中实际位于 `lib/api/`。
2. README 中提到的 `tools/protoc-gen-goone`、`tools/cmd/scaffold`、`tools/cmd/genproto`，在当前 workspace 中并不存在。
3. 协议与命令字主要来自外部模块 `github.com/Iori372552686/game_protocol`，不是本仓库现生成。
4. 运行主入口推荐使用 `main.sh`；`build.sh` 仍保留一些历史目标，不能把它当作完全等价的“真实现状”。

## 3. 核心名词

| 名词 | 含义 |
| --- | --- |
| `CSPacket` | 客户端到网关、网关到客户端的二进制包结构。 |
| `SSPacket` | 服务与服务之间通过 Bus 传递的二进制包结构。 |
| `BusId` | 服务实例在总线中的标识，路由、注册发现、消息投递都围绕它展开。 |
| `TransMgr` | 事务管理器。负责按命令字分发消息、创建事务协程、等待 RPC 响应。 |
| `IContext` | 业务 handler 拿到的统一上下文，封装 uid、zone、rid、Call/Send 能力。 |
| `RID` / `RouterID` | 自定义路由键。很多服务按它而不是单纯按 uid 做哈希路由。 |
| `RouteRule` | 某一类服务该如何选实例：随机、按 uid 哈希、按 zone 哈希、按 routerId 哈希等。 |
| `gamedata` | Excel 配置经过生成后的 `.conf` 文本数据和对应 Go 仓库代码。 |

## 4. 顶层目录速览

| 路径 | 作用 |
| --- | --- |
| `readme.md` | 项目总览、架构背景、推荐工作流入口。 |
| `main.sh` | 统一控制台入口，包装 doctor / go / docker / build / deploy / host init。 |
| `build.sh` | 原始构建脚本，从 repo root 构建服务二进制到 `build/`。 |
| `common/` | 公共配置、公共常量、游戏配置加载器、生成数据仓库。 |
| `module/` | 项目侧公共模块，包含 server type、路由规则、通用业务函数、掉落算法等。 |
| `lib/` | 框架核心库。 |
| `src/` | 各业务服务实现。 |
| `tools/` | Excel 配置生成工具、联调/集成测试工具。 |
| `env/` | 本地或远端依赖环境脚本、docker compose、IDE 配置样例。 |
| `deploy/` | Ansible 部署、机器初始化、服务启停脚本。 |
| `doc/` | 项目文档。 |

## 5. 推荐入门顺序

建议第一次接触项目时按这个顺序阅读：

1. `readme.md`
2. `common/gconf/config.go`
3. `module/misc/constant.go`
4. `lib/service/application/app.go`
5. `src/mainsvr/main.go` 与 `src/mainsvr/app.go`
6. `src/connsvr/app.go`
7. `lib/service/router/router.go`
8. `lib/service/transaction/transaction_mgr_impl.go`
9. `common/gamedata/gamedata.go`
10. 任意一个 `common/gamedata/repository/*/*Data.gen.go`
11. `deploy/README.md`

## 6. 服务地图

### 6.1 `src/connsvr`

网关服务，负责：

- 接 TCP 和 WebSocket 连接。
- 持有 HTTP Sign 与 RestApi 管理器。
- 把客户端消息转发到内部服务。
- 注册少量网关侧事务命令。

特征：

- `CreateTcpServer("", listen_port+1, ...)`
- `CreateWebSocketServer("gin", "debug", listen_port, ...)`
- `TransMgr.InitAndRun(..., false, 0)`，默认不启用 uid 串行锁。

### 6.2 `src/mainsvr`

主逻辑服务，负责大部分玩家业务与房间入口逻辑：

- 加载 `gamedata`
- 初始化 Redis
- 注册大量 `CMD_MAIN_*` 命令
- 用 `RoleMgr` 管理角色对象
- 按分钟 tick 做角色级周期处理

特征：

- 使用 `globals.RoleMgr`
- 使用 `NewRoleAdapter(...)` 包装需要先加载角色的 handler
- `TransMgr.InitAndRun(..., true, 100)`，启用串行化事务处理

### 6.3 `src/infosvr`

玩家简要信息服务：

- 维护 brief info 缓存
- Redis 落存 / 读回
- 提供其他服务查询玩家简档的能力

特征：

- 结构简单，基本是 `router + TransMgr + Redis + InfoMgr`
- 不启用 uid lock

### 6.4 `src/mysqlsvr`

持久化服务：

- 初始化 `OrmMgr`
- 注册 mysql 相关事务命令
- 用 `src/mysqlsvr/manager` 注册业务表与异步处理器

特征：

- 表结构来自 `manager.GetTables()`
- 处理结束时会 `manager.Close()` 与 `globals.MysqlMgr.Destroy()`

### 6.5 `src/roomcentersvr`

房间中心服务：

- 管理房间列表与房间实例
- 启动 AI 房间初始化逻辑
- 加载本地或 Nacos `gamedata`
- 处理德州房间相关命令

特征：

- `TransMgr.InitAndRun(..., true, 200)`
- `globals.RoomListMgr.Tick(nowMs)` 在 `OnTick` 中推进
- `NewZoneAdapter(...)` 先按 `RID` 找到房间管理器，再执行业务

### 6.6 `src/web_svr`

HTTP 管理/后台服务：

- 启动 Gin Web 服务
- 初始化 Redis、SignMgr、RestApiMgr、敏感词
- 注册 Web API controller

特征：

- 和 `connsvr`/`mainsvr` 不同，`web_svr` 不是典型的 Bus 驱动服务
- 入口重点在 `web_gin.RunGin(...)` 和 `controller.LoadWebRoutes`

### 6.7 关于 `build.sh` 的历史目标

`build.sh` 里还保留了 `dbsvr`、`gmconnsvr`、`rcmdsvr`、`gamesvr`、`friendsvr`、`guildsvr` 等分支，但当前 `src/` 目录并没有这些服务实现。说明这个脚本是“兼容历史项目形态”的脚本，而不是纯粹描述当前仓库真实结构的文件。

## 7. 公共模块说明

### 7.1 `common/`

#### `common/gconf/`

- `common/gconf/config.go`：整个项目最重要的共享配置定义。声明 `-svr_conf` flag，定义 `BaseCfg` 与各服务配置结构，统一承载注册中心、Bus、Redis、ORM、Nacos、HTTP 签名、RestApi、pprof 等配置。
- `common/gconf/server_conf.yaml`：通用运行配置样例。
- `common/gconf/server_conf_ide.yaml`：本地 IDE / 开发配置样例。

#### `common/gamedata/`

- `common/gamedata/gamedata.go`：统一的配置数据加载器。支持本地目录加载和 Nacos 远程加载/监听热更新。
- `common/gamedata/data/*.conf`：由配置工具生成的 protobuf text 数据文件。
- `common/gamedata/data/gamedata.tar`：打包后的数据文件。
- `common/gamedata/repository/*/*Data.gen.go`：生成出来的访问仓库代码。每个包会在 `init()` 中调用 `gamedata.Register(...)`，把某张表的解析函数注册到全局。

当前已看到的生成仓库包括：

- `common/gamedata/repository/global_config/GlobalData.gen.go`
- `common/gamedata/repository/item_config/ItemData.gen.go`
- `common/gamedata/repository/machine_config/MachineData.gen.go`
- `common/gamedata/repository/mall_config/MallData.gen.go`
- `common/gamedata/repository/recharge_config/RechargeData.gen.go`
- `common/gamedata/repository/task_config/TaskData.gen.go`
- `common/gamedata/repository/drop_item_confing/DropItemConfingData.gen.go`
- `common/gamedata/repository/texas_config/TexasData.gen.go`
- `common/gamedata/repository/texas_test_config/TexasTestData.gen.go`

这些文件的共同特征：

- 不建议手改。
- 底层依赖 `game_protocol` 的 protobuf 类型。
- 内部一般使用 `atomic.Value` 持有当前版本数据，支持热更替换。

#### 其他 `common` 文件

- `common/define/const.go`：少量项目级常量，目前可见的是房间 ID 长度。
- `common/sensitive/sensitive.txt`：敏感词库文件。

### 7.2 `module/`

#### `module/misc/`

- `module/misc/constant.go`：项目级核心常量。定义 server type、心跳间隔、事务上限、ORM LRU 限制，以及每类服务的 `ServerRouteRules`。
- `module/misc/func.go`：命令字位段解析、是否客户端命令/内部命令/GM 命令、机器人判断、道具数组转 protobuf、角色 icon 数据拼装等。

#### `module/gfunc/`

- `module/gfunc/pub_func.go`：公共杂项函数。包括 panic recover、栈跟踪、结构体赋值、IP 获取、空值判断、泛型三元表达式等。
- `module/gfunc/game_func.go`：游戏业务共用函数。当前重点是房间 ID 生成和德州房间列表索引计算。
- `module/gfunc/notify.go`：游戏事件通知封装。把 protobuf 消息包成 `GameUserEventNotify`，再通过 `router` 发给连接服。

#### 其他 `module` 文件

- `module/drop/mymath.go`：带权随机、随机抽 N 个等掉落/概率工具。

### 7.3 `src/web_svr/common/`

- `src/web_svr/common/define.go`：Web 服务侧常量，例如 `RestApi_SafeMsg_Dir`。
- `src/web_svr/common/stu_protocol.go`：Web 接口请求结构体，目前看到的是消息安全检查请求体。

## 8. `lib/` 文件级索引

### 8.1 总览

`lib/` 是项目框架核。建议把它理解成七个子系统：

- `lib/api/`：框架对外暴露的上下文、协议结构、日志、REST、HTTP 签名、错误封装等 API。
- `lib/db/`：MySQL/XORM/Redis/SSDB 适配层。
- `lib/net/`：TCP/KCP/WS/UDP 网络层与统一工厂。
- `lib/service/`：应用生命周期、Bus、事务、路由、注册发现管理、缓存、异步等核心服务能力。
- `lib/web/`：Gin 启动与 Web REST 辅助层。
- `lib/util/`：通用工具箱。
- `lib/contrib/`：配置中心与注册中心的抽象层与具体实现。

### 8.2 `lib/api/`

#### `lib/api/cmd_handler/`

- `lib/api/cmd_handler/cmd_handler_i.go`：定义 `CmdHandlerFunc`、`WebHandlerFunc`、`IContext`、`RegCmdInterface`，是业务 handler 和框架交互的核心接口层。
- `lib/api/cmd_handler/handler_mgr.go`：管理命令处理器的注册与 HTTP/WS handler 映射。

#### `lib/api/datetime/`

- `lib/api/datetime/const.go`：时间相关常量。
- `lib/api/datetime/datetime.go`：时间戳、tick、当前时间等工具。
- `lib/api/datetime/zonetime.go`：时区相关时间工具。

#### `lib/api/error_capture/`

- `lib/api/error_capture/sentry.go`：Sentry 错误上报适配。
- `lib/api/error_capture/bugsnag.go`：Bugsnag 错误上报适配。

#### `lib/api/http_sign/`

- `lib/api/http_sign/Sign.go`：HTTP 请求签名核心逻辑。
- `lib/api/http_sign/Sign_Mgr.go`：签名实现管理器，按名称维护签名实例。
- `lib/api/http_sign/Transform.go`：签名前后的参数转换、辅助处理。
- `lib/api/http_sign/config.go`：签名配置定义。

#### `lib/api/logger/`

- `lib/api/logger/logger.go`：统一日志入口，对外暴露 `InitLogger`、`Infof`、`Errorf`、`Flush` 等接口。
- `lib/api/logger/plug/cmd_blacklist.go`：按命令字做日志黑名单，减少心跳等高频消息刷屏。
- `lib/api/logger/plug/notify.go`：日志通知钩子，通常用于告警扩展。
- `lib/api/logger/zap/logger.go`：Zap logger 初始化。
- `lib/api/logger/zap/logging.go`：Zap 的扩展日志方法与适配逻辑。

#### `lib/api/net_conf/`

- `lib/api/net_conf/nacos_config.go`：Nacos 网络配置辅助，主要给服务初始化配置中心客户端时使用。

#### `lib/api/rest_api/`

- `lib/api/rest_api/config.go`：RestApi 配置结构。
- `lib/api/rest_api/rest_api.go`：对外 HTTP 调用封装，支持多 URL、按 uid hash、签名 GET/POST 与普通 GET/POST。
- `lib/api/rest_api/rest_api_mgr.go`：RestApi 实例管理器。

#### `lib/api/sharedstruct/`

- `lib/api/sharedstruct/ss_packet.go`：服务间 `SSPacket` 与 `SSPacketHeader` 定义及编解码。
- `lib/api/sharedstruct/cs_packet.go`：客户端侧 `CSPacket` 与 `CSPacketHeader` 定义及编解码。

#### 其他 `lib/api` 文件

- `lib/api/uerror/uerror.go`：带文件/行号/函数信息的统一错误包装。

### 8.3 `lib/db/`

#### `lib/db/mysql/`

- `lib/db/mysql/mysql_mgr.go`：MySQL 管理器。
- `lib/db/mysql/mysql_facade.go`：更薄的一层数据库 facade，对 `database/sql` 做简单初始化封装。

#### `lib/db/redis/`

- `lib/db/redis/config.go`：Redis 配置定义。
- `lib/db/redis/redis_mgr.go`：Redis 管理器，供服务初始化和访问 Redis 实例。

#### `lib/db/ssdb/`

- `lib/db/ssdb/config.go`：SSDB 配置。
- `lib/db/ssdb/ssdb_engin.go`：SSDB 引擎封装。
- `lib/db/ssdb/ssdb_mgr.go`：SSDB 管理器。

#### `lib/db/xorm/`

- `lib/db/xorm/config.go`：XORM 配置结构。
- `lib/db/xorm/orm_engin.go`：XORM engine / engine group 初始化。
- `lib/db/xorm/orm_mgr.go`：多实例 `OrmMgr`，按名字管理 orm 引擎。
- `lib/db/xorm/orm_session.go`：会话与事务辅助。

### 8.4 `lib/net/`

#### `lib/net/gnet_server/`

- `lib/net/gnet_server/gnet_udp.go`：基于 gnet 的 UDP 服务实现。

#### `lib/net/kcp_server/`

- `lib/net/kcp_server/kcp_i.go`：KCP 服务接口。
- `lib/net/kcp_server/kcp.go`：KCP 服务实现。

#### `lib/net/net_mgr/`

- `lib/net/net_mgr/net_i.go`：统一的连接服务对象定义，维护 uid 与连接的映射。
- `lib/net/net_mgr/net_factory.go`：统一工厂，负责创建 TCP / UDP / KCP / WS 服务。
- `lib/net/net_mgr/tcp_impl.go`：TCP 服务封装实现。
- `lib/net/net_mgr/kcp_impl.go`：KCP 服务封装实现。
- `lib/net/net_mgr/ws_impl.go`：WebSocket 服务封装实现。

#### `lib/net/tcp_server/`

- `lib/net/tcp_server/tcp_i.go`：TCP 服务事件接口定义。
- `lib/net/tcp_server/tcp_server.go`：基础 TCP 服务实现。
- `lib/net/tcp_server/tcp_packet.go`：带包边界处理的 TCP packet server。

#### `lib/net/ws_server/`

- `lib/net/ws_server/ws_i.go`：WebSocket 抽象接口。
- `lib/net/ws_server/ws.go`：WebSocket 核心实现。
- `lib/net/ws_server/gin_ws.go`：Gin 版本 WebSocket 实现。
- `lib/net/ws_server/beego_ws.go`：Beego 版本 WebSocket 实现。

### 8.5 `lib/service/`

#### `lib/service/algorithm/`

- `lib/service/algorithm/lru_cache.go`：LRU 缓存实现。

#### `lib/service/application/`

- `lib/service/application/app.go`：应用生命周期骨架。定义 `AppInterface`，驱动 `OnInit/OnReload/OnProc/OnTick/OnExit`。
- `lib/service/application/sig.go`：非 Windows 平台信号处理，`SIGUSR1` 触发 reload，其余退出。
- `lib/service/application/sig_windows.go`：Windows 平台信号兼容实现。

#### `lib/service/async/`

- `lib/service/async/async.go`：异步执行器/协程池实现，常用于串行任务处理。
- `lib/service/async/queue.go`：异步任务队列。

#### `lib/service/bus/`

- `lib/service/bus/bus_i.go`：`IBus` 接口定义。
- `lib/service/bus/bus_factory.go`：Bus 工厂，按地址 scheme 创建具体实现。
- `lib/service/bus/bus_config.go`：解析 `rabbitmq://`、`nats://`、`kafka://`、`rocketmq://`、`nsq://` 等地址。
- `lib/service/bus/bus_ip.go`：BusId 与 IP/字符串之间的辅助转换。
- `lib/service/bus/bus_impl_rmq.go`：RabbitMQ 实现。
- `lib/service/bus/bus_impl_nsq.go`：NSQ 实现。
- `lib/service/bus/bus_impl_nats.go`：NATS 实现。
- `lib/service/bus/bus_impl_kafka.go`：Kafka 实现。
- `lib/service/bus/bus_impl_rocketmq.go`：RocketMQ 实现。

#### `lib/service/bus/nsq/`

- `lib/service/bus/nsq/producer.go`：NSQ producer 辅助封装。
- `lib/service/bus/nsq/consumer.go`：NSQ consumer 辅助封装。

#### `lib/service/cache/`

- `lib/service/cache/cache.go`：通用缓存抽象。
- `lib/service/cache/sharded.go`：分片缓存实现。

#### `lib/service/config/`

- `lib/service/config/nacos_config.go`：服务层的 Nacos 配置辅助。

#### `lib/service/router/`

- `lib/service/router/router.go`：服务间消息发送核心入口。通过 `svrinstmgr + bus` 做实例选择、SS 包封装、广播、发回响应等。

#### `lib/service/sensitive_words/`

- `lib/service/sensitive_words/sensitive.go`：敏感词过滤工具。

#### `lib/service/svrinstmgr/`

- `lib/service/svrinstmgr/svr_inst_mgr.go`：服务实例管理器。负责把自己注册到注册中心，并监听在线实例变化，再按 `ServerRouteRules` 选目标实例。

#### `lib/service/transaction/`

- `lib/service/transaction/transaction_i.go`：事务管理器接口定义。
- `lib/service/transaction/transaction_factory.go`：事务管理器构造入口。
- `lib/service/transaction/transaction_impl.go`：`Transaction` 的具体实现，也是 `IContext` 的主要承载体。
- `lib/service/transaction/transaction_mgr_impl.go`：事务管理器主逻辑。负责命令字注册、事务协程创建、UID 串行锁、等待队列、响应回传。

### 8.6 `lib/web/`

#### `lib/web/http/`

- `lib/web/http/http_api.go`：对外 HTTP 请求客户端，用于被 `lib/api/rest_api` 调用。

#### `lib/web/rest/`

- `lib/web/rest/Auth.go`：Web 鉴权辅助。
- `lib/web/rest/Captcha.go`：验证码相关逻辑。
- `lib/web/rest/Controller.go`：基础控制器与默认路由处理。
- `lib/web/rest/Crypto.go`：Web 层密码学辅助。
- `lib/web/rest/Func.go`：模板函数注册。
- `lib/web/rest/JsonTime.go`：适合 JSON 输出的时间类型。
- `lib/web/rest/RegUtil.go`：Web 路由注册辅助。
- `lib/web/rest/Result.go`：统一 HTTP 返回结构辅助。
- `lib/web/rest/SessionUtil.go`：会话辅助。

#### `lib/web/web_gin/`

- `lib/web/web_gin/config.go`：Gin 服务配置结构。
- `lib/web/web_gin/http.go`：启动 Gin 服务，装配日志/恢复/CORS 中间件。
- `lib/web/web_gin/handler_process.go`：HTTP handler 处理过程辅助。

### 8.7 `lib/util/`

#### 基础工具

- `lib/util/version/version.go`：版本信息辅助。
- `lib/util/slices/slices.go`：切片工具。
- `lib/util/file/file.go`：文件读写辅助。

#### 压缩与定时

- `lib/util/zip/zip.go`：ZIP 压缩/解压辅助。
- `lib/util/zip/gzip.go`：Gzip 压缩/解压辅助。
- `lib/util/timer/timewheel.go`：时间轮实现。

#### OS / 运行环境

- `lib/util/tos/dir.go`：目录辅助。
- `lib/util/tos/port.go`：端口检测与占用检查。
- `lib/util/tos/proc.go`：进程辅助。
- `lib/util/tos/sysinfo.go`：系统信息获取。

#### 并发与锁

- `lib/util/mutex/recursive.go`：递归锁实现。
- `lib/util/mutex/try_lock.go`：可尝试获取的锁实现。
- `lib/util/safego/safe_goroutine.go`：安全 goroutine 封装，避免 panic 直接炸穿主逻辑。
- `lib/util/safego/singleflight.go`：相同任务去重执行。

#### 标识与加载

- `lib/util/idgen/idgen.go`：ID 生成器。
- `lib/util/marshal/load.go`：配置文件加载与反序列化，是各服务 `OnReload()` 的共用入口。

#### 随机、转换、泛型

- `lib/util/random/init.go`：随机数种子初始化。
- `lib/util/random/rand.go`：随机数工具。
- `lib/util/random/bytes.go`：随机字节序列生成。
- `lib/util/random/string.go`：随机字符串生成。
- `lib/util/convert/common.go`：常见类型转换。
- `lib/util/generic/types.go`：轻量泛型类型辅助。

#### 编解码

- `lib/util/encoding/encoding.go`：统一编解码入口。
- `lib/util/encoding/json/json.go`：JSON 编解码。
- `lib/util/encoding/yaml/yaml.go`：YAML 编解码。
- `lib/util/encoding/xml/xml.go`：XML 编解码。
- `lib/util/encoding/msgp/msg_pack.go`：MessagePack 编解码。
- `lib/util/encoding/proto/proto.go`：Protobuf 编解码辅助。

#### 加密

- `lib/util/crypto/md5.go`：MD5 相关辅助。
- `lib/util/crypto/base64.go`：Base64 编解码。
- `lib/util/crypto/aes/base.go`：AES 公共底座。
- `lib/util/crypto/aes/cbc.go`：AES-CBC。
- `lib/util/crypto/aes/cfb.go`：AES-CFB。
- `lib/util/crypto/aes/ecb.go`：AES-ECB。
- `lib/util/crypto/xxtea/xxtea.go`：XXTEA 实现。

#### Excel 旧工具

- `lib/util/xlstrans/main.go`：旧版 Excel 转换 CLI 入口。
- `lib/util/xlstrans/parse_struct.go`：旧版 Excel 结构解析。
- `lib/util/xlstrans/xls_to_const.go`：Excel 转常量文件。
- `lib/util/xlstrans/xls_to_data.go`：Excel 转数据文件。
- `lib/util/xlstrans/xls_to_go.go`：Excel 转 Go 访问代码。
- `lib/util/xlstrans/xls_to_pb.go`：Excel 转 protobuf。
- `lib/util/xlstrans/xls_to_system_unlock.go`：功能开放表专用生成逻辑。

说明：`lib/util/xlstrans` 是历史工具链，当前主推荐链路已经转到 `tools/cfgtool`。

### 8.8 `lib/contrib/`

#### 配置中心抽象

- `lib/contrib/config/config.go`：配置中心的核心接口定义，如 `Source`、`Watcher`、`Client`、`KeyValue`。
- `lib/contrib/config/codec.go`：配置编解码辅助。

#### 配置中心工厂

- `lib/contrib/config/factory/factory.go`：解析 `consul://`、`etcd://`、`k8s://`、`apollo://`、`nacos://` 地址并创建 client。
- `lib/contrib/config/factory/backend_etcd_enabled.go`：启用 etcd 时的实现。
- `lib/contrib/config/factory/backend_etcd_disabled.go`：未启用 etcd 时的占位实现。

#### 配置中心具体实现

- `lib/contrib/config/consul/config.go`：Consul 配置源实现。
- `lib/contrib/config/consul/watcher.go`：Consul 配置监听。
- `lib/contrib/config/etcd/config.go`：etcd 配置源实现。
- `lib/contrib/config/etcd/watcher.go`：etcd 配置监听。
- `lib/contrib/config/kubernetes/config.go`：Kubernetes ConfigMap/Secret 配置源。
- `lib/contrib/config/kubernetes/watcher.go`：Kubernetes 配置监听。
- `lib/contrib/config/apollo/apollo.go`：Apollo 配置源实现。
- `lib/contrib/config/apollo/watcher.go`：Apollo 配置监听。
- `lib/contrib/config/apollo/json_parser.go`：Apollo JSON 数据解析辅助。
- `lib/contrib/config/nacos/nacos.go`：Nacos 配置源实现。
- `lib/contrib/config/nacos/watcher.go`：Nacos 配置监听。

#### 注册中心抽象

- `lib/contrib/registry/registry.go`：注册中心接口定义，如 `Registrar`、`Discovery`、`Watcher`、`ServiceInstance`。

#### 注册中心工厂

- `lib/contrib/registry/factory/factory.go`：解析注册中心地址，统一创建 zk / etcd / consul / nacos / k8s client。
- `lib/contrib/registry/factory/backend_etcd_enabled.go`：启用 etcd 时的实现。
- `lib/contrib/registry/factory/backend_etcd_disabled.go`：未启用 etcd 时的占位实现。

#### 注册中心具体实现

- `lib/contrib/registry/consul/client.go`：Consul 客户端包装。
- `lib/contrib/registry/consul/registry.go`：Consul 注册/发现实现。
- `lib/contrib/registry/consul/service.go`：Consul service 数据结构与辅助处理。
- `lib/contrib/registry/consul/watcher.go`：Consul watcher。
- `lib/contrib/registry/etcd/registry.go`：etcd 注册/发现实现。
- `lib/contrib/registry/etcd/service.go`：etcd service 注册辅助。
- `lib/contrib/registry/etcd/watcher.go`：etcd watcher。
- `lib/contrib/registry/nacos/registry.go`：Nacos 注册/发现实现。
- `lib/contrib/registry/nacos/watcher.go`：Nacos watcher。
- `lib/contrib/registry/zookeeper/register.go`：ZooKeeper 注册逻辑。
- `lib/contrib/registry/zookeeper/service.go`：ZooKeeper service 节点辅助。
- `lib/contrib/registry/zookeeper/watcher.go`：ZooKeeper watcher。
- `lib/contrib/registry/kubernetes/registry.go`：Kubernetes 注册/发现实现。

## 9. 工具链与脚本说明

### 9.1 主入口与基础脚本

- `main.sh`：项目推荐入口。提供 `help`、`doctor`、`install ansible`、`go`、`docker`、`build`、`env list`、`role list`、`deploy`、`host init` 等命令。
- `build.sh`：原始构建脚本，从当前目录推断项目根目录，所以必须在 repo root 执行。

推荐原则：

- 优先用 `./main.sh ...`
- 非必要不要直接调用底层脚本
- 如果在 Windows 上工作，优先用 WSL2 或 Git-Bash 执行这些 shell 脚本

### 9.2 `env/`

- `env/go-manager.sh`：Go 版本安装/切换脚本。
- `env/docker.sh`：用 Ansible 在目标环境管理 Docker 依赖服务。
- `env/docker-playbook.yml`：`docker.sh` 使用的 playbook。
- `env/env_docker.yaml`：依赖服务 docker compose 定义。
- `env/server_conf_ide.yaml`：本地调试配置样例。
- `env/README.md`：环境管理说明。

### 9.3 `deploy/`

- `deploy/common.sh`：公共 shell 函数、日志样式、dotenv 加载、Ansible 默认配置。
- `deploy/deploy.sh`：主部署脚本，支持 `env list`、`role list`、`run --env ... --action ...`。
- `deploy/init-host.sh`：机器初始化脚本。
- `deploy/install.sh`：控制机安装 Ansible 的脚本。
- `deploy/scripts/server.sh`：目标机上的服务启停脚本，支持 `start/stop/restart/check/reload`。
- `deploy/hosts/*.txt`：Ansible inventory。
- `deploy/playbook_dev/*.yml`：不同环境的 playbook。
- `deploy/roles/*`：各服务的部署 role。
- `deploy/README.md`：部署说明。

重要事实：

- `deploy/scripts/server.sh reload` 实际发送 `SIGUSR1`，会触发 `application` 的 `OnReload()`。
- `SERVER_CONF` 环境变量可覆盖目标机默认配置路径。
- `NO_COLOR=1` 可关闭彩色输出。

### 9.4 `tools/cfgtool`

这是当前仓库里最重要的生成工具，职责是把 Excel 配置转成 proto、数据文件和 Go 访问代码。

#### 入口与基础定义

- `tools/cfgtool/main.go`：命令行入口。解析 `-xlsx`、`-text`、`-proto`、`-json`、`-bytes`、`-code`、`-module`、`-pb` 等参数，然后串起完整生成流程。
- `tools/cfgtool/domain/define.go`：全局参数与类型枚举定义。

#### 基础层

- `tools/cfgtool/internal/base/typespec.go`：配置、结构、字段、索引、枚举等元数据类型定义。
- `tools/cfgtool/internal/base/func.go`：元数据对象的公共方法，例如字段追加、索引参数拼接、类型字符串生成。
- `tools/cfgtool/internal/base/util.go`：文件保存、Go 格式化、目录遍历等基础工具。

#### 解析层

- `tools/cfgtool/internal/parser/parser.go`：读取 Excel 中的“生成表”，识别 `@config`、`@struct`、`@enum` 规则，构建元数据。
- `tools/cfgtool/internal/parser/reference.go`：分析跨文件引用关系，为生成 proto import 做准备。

#### 管理层

- `tools/cfgtool/internal/manager/table_mgr.go`：原始表格数据管理。
- `tools/cfgtool/internal/manager/type_mgr.go`：`Config` / `Struct` / `Enum` 元数据管理。
- `tools/cfgtool/internal/manager/convert_mgr.go`：基础类型与枚举转换函数注册。
- `tools/cfgtool/internal/manager/proto_mgr.go`：保存生成出来的 proto 文本，并用动态 proto 解析器反射生成 message。

#### 模板层

- `tools/cfgtool/internal/templ/proto_tpl.go`：proto 文件模板。
- `tools/cfgtool/internal/templ/code_tpl.go`：Go 仓库代码模板。
- `tools/cfgtool/internal/templ/index_tpl.go`：多字段索引辅助代码模板。

#### 生成层

- `tools/cfgtool/service/proto_gen.go`：生成 proto 文本。
- `tools/cfgtool/service/data_gen.go`：生成 `.conf` / `.json` / `.bytes` 数据。
- `tools/cfgtool/service/code_gen.go`：生成 `*Data.gen.go` 仓库代码。
- `tools/cfgtool/service/index_gen.go`：生成索引支持代码。

#### 其他

- `tools/cfgtool/test/tool_test.go`：cfgtool 测试。

生成链路可以简化理解为：

1. 解析 Excel
2. 抽取元数据
3. 生成 proto
4. 解析 proto 描述
5. 生成数据
6. 生成 Go 仓库代码

### 9.5 `tools/tester`

`tools/tester` 更像联调脚本集合，而不是单元测试框架。

#### 测试文件

- `tools/tester/conn_test.go`：连接相关测试。
- `tools/tester/login_test.go`：登录链路测试。
- `tools/tester/sync_data_test.go`：同步/数据类测试。
- `tools/tester/rummy_test.go`：牌局或房间联调测试。
- `tools/tester/t_test.go`：其他测试入口。

#### 辅助层

- `tools/tester/tester_util/tester_util.go`：底层发包/收包工具，直接使用 `CSPacketHeader` 与 protobuf。
- `tools/tester/tester_util/session.go`：测试 session 抽象，负责 TCP 连接、登录、登出、发包收包。
- `tools/tester/tester_util/role.go`：角色侧测试发包构造，如登录、心跳。
- `tools/tester/tester_util/room.go`：房间相关测试发包构造，如 QuickStart、JoinRoom、CreateRoom。
- `tools/tester/tester_util/ws_util.go`：WebSocket 联调客户端，支持心跳、自动重连、消息回调。

注意事项：

- 这些测试工具里存在硬编码地址、uid、token 样例，不能直接当成通用测试框架使用。
- 更适合“服务已经跑起来后”的联调与冒烟验证。

### 9.6 旧工具：`lib/util/xlstrans`

虽然当前主链路是 `tools/cfgtool`，但仓库里仍然保留旧版 Excel 转换工具：

- 位置在 `lib/util/xlstrans/`
- 这些文件是 `package main`
- 更像历史版本保留，不建议继续扩展

如果需要配置生成能力，优先读和改 `tools/cfgtool`。

## 10. 框架共识与规范

### 10.1 服务启动规范

所有标准服务都遵循同一骨架：

1. `main.go` 中 `flag.Parse()`
2. `defer logger.Flush()`
3. `application.Init(&XxxImpl{})`
4. `application.Run()`

服务实现需要满足 `AppInterface`：

- `OnInit()`
- `OnReload()`
- `OnProc()`
- `OnTick(lastMs, nowMs)`
- `OnExit()`

通常的初始化顺序是：

1. `runtime.GOMAXPROCS(...)`
2. `OnReload()` 读取配置
3. `logger.InitLogger(...)`
4. 初始化 router / TransMgr / Redis / ORM / SignMgr / RestApiMgr / gamedata 等资源
5. 启动网络服务或 HTTP 服务

### 10.2 配置规范

- 统一使用 `-svr_conf` 指定配置文件。
- 各服务配置结构都定义在 `common/gconf/config.go`。
- 公共配置统一放在 `BaseCfg`。
- 新增跨服务配置时，优先加在 `BaseCfg`；只被某个服务使用的配置，才放到对应服务结构里。

### 10.3 路由与通信规范

- 服务间通信统一走 `router`。
- `router` 底层统一依赖 `IBus`。
- 服务实例选择统一走 `svrinstmgr`。
- 路由规则统一放在 `module/misc/constant.go` 的 `ServerRouteRules`。

也就是说，业务代码原则上不应该自己重新发明“选哪台服务实例、怎么组包、怎么发总线消息”这一套逻辑。

### 10.4 事务与 handler 规范

- 所有命令字都要先 `RegisterCmd(...)` 再启动 `TransMgr`。
- `RegisterCmd` 必须发生在 `TransMgr.InitAndRun(...)` 之前。
- `handler` 收到的是 `IContext + data []byte`。
- 有状态服务通常会用 adapter 先把业务对象取出来再执行业务：
  - `mainsvr` 用 `NewRoleAdapter(...)`
  - `roomcentersvr` 用 `NewZoneAdapter(...)`

### 10.5 UID 串行化规范

`TransMgr` 支持 `useUidLock` 模式：

- 有内存态角色/房间状态的服务应优先启用
- 同一 uid 或同一路由键在同一时刻只允许一个事务协程处理
- 后续消息进入等待队列

当前可见实践：

- `mainsvr`：启用
- `roomcentersvr`：启用
- `connsvr` / `infosvr` / `mysqlsvr`：未启用

### 10.6 消息所有权规范

`onRecvSSPacket(packet)` 之后，代码里通常会立即把包交给 `globals.TransMgr.ProcessSSPacket(packet)`，并写注释说明“所有权转交给 transmgr，后面不能再使用 packet（包括 data）”。

这是项目里的明确约定：

- 一旦把 `SSPacket` 交给 `TransMgr`，后续代码不要再读写它
- 不要把它缓存到外部变量里继续使用

### 10.7 生成代码规范

以下内容默认视为生成产物：

- `common/gamedata/repository/*/*Data.gen.go`
- cfgtool 产出的 `.conf` / `.json` / `.bytes`
- cfgtool 产出的 proto / index / repository 代码

规则：

- 不手改生成文件
- 改 Excel 源或生成模板，再重新生成
- 代码评审时要区分“模板逻辑变更”和“生成结果变化”

### 10.8 日志与错误处理规范

- 统一走 `lib/api/logger`
- 高风险 goroutine 优先使用 `safego.Go` 或 `safego.SafeFunc`
- 高频命令可以注册到 cmd blacklist，避免日志噪音
- 致命错误最终要落到统一 logger，而不是随意 `fmt.Println`

### 10.9 Web 服务规范

- Web 服务统一通过 `web_gin.RunGin(...)` 启动
- 路由集中在 `controller.LoadWebRoutes(...)`
- 如果需要访问其他 HTTP 服务，优先走 `RestApiMgr`
- 需要签名的 HTTP 请求，优先走 `SignMgr + RestApi`

### 10.10 注册中心与配置中心规范

当前框架已经统一成“地址字符串 + 工厂”的模式：

- 注册中心：`zk://`、`etcd://`、`consul://`、`nacos://`、`k8s://`
- 配置中心：`consul://`、`etcd://`、`apollo://`、`nacos://`、`k8s://`
- Bus：`rabbitmq://`、`nats://`、`kafka://`、`rocketmq://`、`nsq://`

规则：

- 新后端优先接到工厂层，而不是在服务代码里写 `if backend == ...`
- 地址解析逻辑要继续保持字符串配置化，而不是在业务层泄漏具体 SDK 初始化细节

## 11. 规则清单

这部分适合开发者和 agent 直接当“做事前检查表”。

### 11.1 新增或修改命令字

- 先确认命令字与 protobuf 消息定义在外部 `game_protocol` 中已经存在。
- 再到对应服务的 `cmd_handler/register.go` 注册。
- 如果命令依赖角色、房间或其他内存态对象，优先套已有 adapter。

### 11.2 新增服务配置

- 先判断它是否属于 `BaseCfg` 共享配置。
- 配置加载必须通过 `marshal.LoadConfFile(...)` 进入统一结构体。
- 不要在服务里散落读取多个独立配置文件。

### 11.3 新增共享模块

- 框架通用能力放 `lib/`
- 项目侧公共能力放 `common/` 或 `module/`
- 只属于某个服务的逻辑留在 `src/<svr>/`

### 11.4 新增路由规则或服务类型

- 服务类型加在 `module/misc/constant.go`
- 路由规则配置加在 `ServerRouteRules`
- 如果路由维度变化，优先扩展 `svrinstmgr` 和 `router`，不要在业务 handler 手写实例选择

### 11.5 新增配置表 / gamedata

- 从 Excel 源出发
- 通过 `tools/cfgtool` 重新生成
- 确认 `repository/*/*.gen.go` 会自动 `Register(...)`
- 本地目录和 Nacos 两种加载路径都要考虑

### 11.6 修改部署脚本

- 优先修改 `main.sh` 包装层的行为描述，再改底层脚本
- 保持 `deploy.sh` / `init-host.sh` / `server.sh` 的参数风格一致
- 注意生产环境 inventory 和凭据文件可能包含敏感信息

### 11.7 修改 `build.sh`

- 先确认它服务于“当前仓库实际存在的服务”还是“兼容历史目标”
- 不要误以为 `build.sh` 里的所有 target 当前都真实存在

### 11.8 改文档时的优先级

- 代码
- 当前脚本
- 当前配置
- README
- 历史 setup 文档

## 12. 给 agent 的额外提醒

如果你是自动化 agent，进入这个仓库时最好先记住这几件事：

1. 先确认当前代码树，不要直接相信 README 中提到但仓库里不存在的工具或目录。
2. `lib/contrib/protoc/` 和 `lib/util/deps/protoc/` 里大部分是第三方随包文件，不是项目业务逻辑。
3. 真正的协议命令字来自 `github.com/Iori372552686/game_protocol`。
4. 生成文件很多，尤其是 `common/gamedata/repository/*/*.gen.go`，默认不要手改。
5. 如果你要理解一个服务，优先读：
   - `main.go`
   - `app.go`
   - `globals/globals.go`
   - `cmd_handler/register.go`
6. 如果你要改服务间通信，先读：
   - `lib/service/router/router.go`
   - `lib/service/transaction/transaction_mgr_impl.go`
   - `lib/api/sharedstruct/ss_packet.go`
7. 如果你要改配置或注册中心适配，先读：
   - `common/gconf/config.go`
   - `lib/contrib/config/factory/factory.go`
   - `lib/contrib/registry/factory/factory.go`
8. 如果你要改配置表生成，先读 `tools/cfgtool/`，不要先碰 `lib/util/xlstrans/`。

## 13. 最后总结

理解这个项目时，最有效的心智模型是：

- `lib/` 是框架层
- `common/` 和 `module/` 是项目公共层
- `src/` 是服务层
- `tools/` 是生成与联调工具层
- `env/` 与 `deploy/` 是工程化运维层

只要先按这个分层理解，再去看某个具体服务，就不容易把“业务代码”“框架代码”“生成代码”“历史工具代码”混在一起。
