升级计划
已经有一个很“游戏服友好”的骨架：
- 内网 RPC：SSPacket（Cmd/Uid/RouterID/TransID/CmdSeq）+ router（bus 转发）+ TransactionMgr（按 DstTransID 分发、可做 uid/rid 串行）+ cmd_handler.IContext（ParseMsg/Call/Send/日志）
- HTTP/WS：CmdHandlerMgr 用 string->handler 的方式注册，dispatch.go 把 URL 转成 cmd 字符串再找 handler
最大问题是：多协议入口是多套 handler 注册/ctx/编解码，而“命令字/方法名”也分散在 register.go、URL path、cmd enum 里。
--------------------------------------------------------------------------------
业界框架“共同点”（Kratos / due / CloudWeGo）
- Kratos：核心是 transport 抽象 + middleware 链 + 业务 service 层；接口强、扩展方便，但“游戏服 uid 串行/事务”要自己做。
- due：模块化思路借鉴 Kratos，强调“只暴露必要接口”，更偏游戏业务落地（它 README 就明确借鉴 kratos 的模块设计思路，可参考：）。
- 字节系 CloudWeGo（Kitex/Hertz）：核心是 IDL 驱动（proto/thrift）生成 client/server 桩代码 + middleware + 高性能网络；工程化最强的一点是“定义一次 service，自动生成注册/路由/脚手架”。
你要的“聚合 ctx + cmd_handler + dispatch，并支持 HTTP/GRPC/WS/SSPack，且用 proto service 统一定义方法名与 cmd 绑定 + 代码生成”，本质上就是把 CloudWeGo 的 IDL 驱动 + Kratos 的 middleware/transport 分层，嫁接到你们现有 TransactionMgr/SSPacket 这套“游戏服执行模型”上。
--------------------------------------------------------------------------------
方案总览：IDL（proto service）为中心，一套业务实现，多种 transport 复用
1) 统一 IDL：用 proto service 定义“方法名”，用 option 绑定“cmd”
建议引入三类 proto 文件：
A. cmd.proto（统一命令字）
把你们现有的 g1_protocol.CMD_* 迁移/镜像到 proto（或由脚本从现有枚举生成）：
- enum Cmd { CMD_MAIN_LOGIN_REQ = 0x....; ... }
这样后续 service 的 method 能直接引用 Cmd 常量，不会散落在 Go 代码里。
B. options.proto（统一路由与协议绑定）
定义 method options（挂在 google.protobuf.MethodOptions）：
- goone.cmd：绑定 SSPacket/CSPacket 的 cmd（通常绑定 *_REQ）
- goone.http：绑定 HTTP path/method（也可以直接用 google.api.http）
- goone.ws：是否开放 WS / 或 WS 的 cmd（可复用 goone.cmd）
- goone.uid_lock：是否需要 uid/rid 串行（对有状态服很重要）
- goone.auth / goone.sign / rate_limit：后续中间件策略（可选）
C. service_xxx.proto（业务服务定义）
例如：
- service MainService { rpc Login(LoginReq) returns (LoginRsp) { option (goone.cmd)=CMD_MAIN_LOGIN_REQ; option (goone.uid_lock)=true; option (goone.http)="/v1/main/login"; } }
> 你们现在约定 rsp_cmd = req_cmd + 1，生成器可以在编译期校验这个约定（或者也允许显式配置 cmd_rsp）。
--------------------------------------------------------------------------------
2) 统一 Context：GoOne Context 做“多协议聚合 ctx”
建议新增一个“聚合 ctx”，它不是替换 context.Context，而是 包一层，把协议无关的信息统一起来：
- 身份：uid/zone/rid、auth claims、role/session
- 路由：cmd、service/method、src/dst bus id、trace_id
- 传输：transport=http/grpc/ws/sspack、peer ip、headers/metadata
- 能力：
- Unmarshal(req)：统一解析 proto（SSPack 是 proto binary；HTTP 可 JSON->proto）
- Reply(rsp)：统一回包（SSPack 回 SSPacket；HTTP 回 JSON；gRPC 直接 return；WS 回 CSPacket）
- Call/Send：复用你们现有的 Transaction.CallMsgBySvrType/SendMsgByServerType 能力（对内网 RPC 非常关键）
这样，业务 handler 永远只面对一种 ctx，不再区分 gin.Context / Transaction / grpc.Context。
--------------------------------------------------------------------------------
3) 统一 Dispatcher：一套路由表 + 一套 middleware
建议引入一个“统一 dispatcher（注册中心）”：
- 核心索引：
- cmd(uint32) -> handler（SSPacket/CSPacket/WS）
- http_path(string) -> handler（HTTP）
- grpc_method(fullname) -> handler（gRPC，通常由 grpc 桩代码直接调用业务实现，也可走 dispatcher）
- handler 形态（推荐）：
- 业务层统一为：func(ctx *Context, req proto.Message) (proto.Message, error)
- middleware 形态：
- func(next Handler) Handler
- 典型 middleware：
- recover、日志、metrics、tracing
- auth/sign/acl
- uid_lock / rid_lock（游戏服强需求）
- 超时、限流、熔断（可选）
> 你们现有 TransactionMgr 已经实现了“按 trans + uid/rid 串行”的执行模型；建议把它保留为 SSPacket 的执行引擎，同时让 HTTP/GRPC/WS 也可选走同一套“串行执行器”（这样全协议语义一致）。
--------------------------------------------------------------------------------
4) Transport 适配：HTTP / gRPC / WS / SSPacket 四种入口复用同一业务
建议分层：
- Service（业务实现）：只写一次
- Transport adapter：负责把协议转换成统一 ctx + req，然后调用 service
四种入口的典型流转：
A. SSPacket（你们现有内网 RPC）
- 收到 SSPacket：按 header.Cmd 找 handler
- proto.Unmarshal(body, req)
- 调用 service
- proto.Marshal(rsp) 并按你们规则回 cmd+1、DstTransID/SrcTransID/CmdSeq 等
B. HTTP（Gin 或 Hertz 都可）
- path -> method -> handler
- body JSON -> proto（protojson）
- service -> rsp proto -> JSON
C. WebSocket（客户端 CSPacket）
- 从 CSPacket header 取 cmd
- body -> proto -> service -> proto -> CSPacket rsp
D. gRPC
- gRPC server 收到 method
- 直接调用同一个 service 实现（推荐），或走 dispatcher（便于统一 middleware）
--------------------------------------------------------------------------------
代码生成（关键）：protoc-gen-goone 生成注册与脚手架
你要的“绑定 cmd 注册、初始化、手架生成”，建议做成一个专用生成器插件：
生成器输出（建议）
对每个 service Foo 生成：1) SSPacket 注册代码（对接 TransactionMgr.RegisterCmd）
- 自动生成 RegisterFooToTransMgr(transMgr, impl)：
- 遍历每个 method 的 goone.cmd
- 生成 wrapper：CmdHandlerFunc(IContext, raw)->ErrorCode，内部做 proto decode/encode，最后 ctx.SendMsgBack(...)2) HTTP 路由注册代码（对接 gin）
- 自动生成 RegisterFooHTTP(r *gin.Engine, impl) 或 RegisterFooHTTP(g gin.IRoutes, impl)
- 支持 option 指定 path/method/body3) WS 路由注册代码（对接 CSPacket）
- 自动生成 RegisterFooWS(dispatcher, impl) 或 RegisterFooCmd(dispatcher, impl)（本质按 cmd）4) Client stub（可选）
- 生成 FooClient，内部用你们现有 IContext.CallMsgBySvrType/SendMsgByServerType 或新的统一 Invoker5) 脚手架（可选）
- goonectl new service mainsvr：生成 proto + handler skeleton + 注册入口
- 生成文件要遵循“不覆盖手写文件”的策略（只生成 *.gen.go 或 *_goone.pb.go）
--------------------------------------------------------------------------------
迁移建议（不破现网）
第一阶段（最快见效）：先统一 SSPacket cmd 注册为“生成代码”
- 不动 HTTP/WS
- 把 src/*/cmd_handler/register.go 逐步替换为 generated.RegisterXXXToTransMgr(globals.TransMgr, impl)
- 业务逻辑逐步从“函数式 handler”迁移到 “service 方法”
第二阶段：HTTP/WS 变成 transport adapter（调用同一 service）
- dispatch.go 只负责解析 uri/body，然后交给 dispatcher（统一注册中心）
- 老的 CmdHandlerMgr 先保留为兼容层，逐步下线
第三阶段：引入 gRPC（可选）
- 用于后台/运营/内部服务调用（不一定替代 SSPacket）
- gRPC 也调用同一个 service 实现
--------------------------------------------------------------------------------
你现在立刻可以定的“统一规范”
- IDL 为主：新增功能必须在 proto service 里定义 method
- 命令字来源唯一：由 cmd.proto（生成到 Go enum / 或映射到现有 g1_protocol）统一管理
- handler 不再手写 RegisterCmd 列表：由 protoc 插件生成
- 一套 middleware：签名/鉴权/uid lock/日志/metrics 在 dispatcher 层做
--------------------------------------------------------------------------------
如果你想我继续“落地到 GoOne 级别”，我建议你先选一个切入点：
- A：先做 SSPacket（内网 RPC）生成注册（最贴合你们现有 TransactionMgr/uid lock）
- B：先做 HTTP（web_svr）把 URL/path 映射到 proto service method
你希望先从 A 还是 B 开始？我可以在你选定后，把 options.proto + cmd.proto + protoc-gen-goone 的输出格式（文件名、生成函数签名、如何在 src/mainsvr/app.go 初始化）写成一份可直接开工的“技术设计 + 目录结构草案”。
