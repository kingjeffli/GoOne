## GoOne SSPacket RPC (Phase A) — IDL 驱动方案

本方案的目标是：把 **CloudWeGo 的 IDL 驱动**（proto service + 代码生成）与 **Kratos 的 middleware/transport 分层** 嫁接到 GoOne 现有的 **TransactionMgr + SSPacket** 执行模型上。

### 1. 目录约定

- **IDL 源码**：`api/proto/**`
- **生成代码**：`api/gen/**`
- **运行时**：`lib/service/ssrpc/**`
- **生成器**：`tools/protoc-gen-goone/**`

### 2. options.proto（cmd 绑定）

`api/proto/goone/options/v1/options.proto` 提供 method option：`(goone.options.v1.ssrpc)`，用于把 proto method 绑定到 `SSPacketHeader.Cmd`。

示例：

- `cmd`：请求 cmd（uint32）
- `cmd_resp`：响应 cmd（默认 `cmd+1`）
- `one_way`：只发不回
- `uid_lock`：预留（Phase A+）

### 3. 定义 service（示例）

示例文件：`api/proto/game/main/v1/main.proto`

### 4. 生成命令（建议）

建议把 `api/proto` 作为 include 根目录（这样 import 写成 `goone/options/v1/options.proto`）。

> 注意：你需要安装 `protoc`、`protoc-gen-go`、`protoc-gen-goone`（本仓库提供了 `tools/protoc-gen-goone` 源码，可 `go install`）。

示例（思路）：

```bash
# in repo root
protoc -I=api/proto \
  --go_out=. --go_opt=module=github.com/Iori372552686/GoOne \
  --goone_out=. --goone_opt=module=github.com/Iori372552686/GoOne \
  api/proto/game/common/v1/common.proto \
  api/proto/game/main/v1/main.proto
```

### 4.2 如何拿到 g1_protocol.CMD 的数值（用于 option cmd）

因为 Phase A 的 `(goone.options.v1.ssrpc).cmd` 是 `uint32`，你需要把 `g1_protocol.CMD_XXX` 的 **数值**填进 proto。

仓库提供了一个小工具：

- `./scripts/dumpg1cmd.sh --exact CMD_ROOM_CENTER_INNER_ROOM_LIST_REQ`
- `./scripts/dumpg1cmd.sh CMD_ROOM_CENTER_INNER_`

### 4.1 关于 go_package（强烈建议）

Phase A+（跨 package message type）要求每个 proto 文件都配置正确的 `option go_package = "...;name"`，否则生成器无法解析跨文件类型的 Go import path。

### 5. 在 mainsvr 中初始化（接入方式）

Phase A 的目标是生成：

- `Register<SomeService>ToTransactionMgr(mgr *transaction.TransactionMgr, srv <SomeService>SSServer)`
- `Register<SomeService>ToTransactionMgr(mgr transaction.ITransactionMgr, srv <SomeService>SSServer)`

典型接入点：`src/mainsvr/app.go` 的初始化阶段（替代手写 `globals.TransMgr.RegisterCmd(...)`）。

伪代码：

```go
mainv1.RegisterMainServiceToTransactionMgr(&globals.TransMgr, mainv1.MainServiceSSServer{
  Impl: &service.MainServiceImpl{},
  MW: []ssrpc.Middleware{
    ssrpc.Recover(),
    ssrpc.Logging(),
  },
})
```

生成器会为所有带 `(goone.options.v1.ssrpc)` option 的 method 自动生成：

- cmd 解码（`ctx.ParseMsg`）
- middleware 链执行
- 调用 `srv.Impl.<Method>()`
- `one_way=false` 且返回 rsp 非 nil 时自动 `ctx.SendMsgBack(rsp)`（默认响应 cmd=cmd+1）

### 6. 生成代码风格（优化：统一走 ssrpc.WrapUnary）

当前 `protoc-gen-goone` 生成的 `Register<Service>ToTransactionMgr` 会调用 runtime 的 `ssrpc.WrapUnary(...)`，而不是把解码/中间件/回包逻辑全部内联进生成文件。

好处：

- runtime 行为集中在 `lib/service/ssrpc/*`，更容易演进（metrics/trace/mcp/uid_lock）
- 生成文件更小、diff 更稳定

### 5.1 cmd_resp 的语义

- 如果 `cmd_resp=0`（未配置），生成器默认使用 `cmd+1`（保持 GoOne 现有约定）
- 如果显式配置了 `cmd_resp`，生成器会尝试用指定 cmd 回包：
  - 若底层 context 支持 `SendMsgBackWithCmd`（当前 `Transaction` 已支持），会按 `cmd_resp` 回包；
  - 否则退化为 `SendMsgBack`（仍按 cmd+1）。

> 备注：GoOne 的 `Transaction` 侧默认通过 `CmdSeq` 关联请求/响应。为兼容 `cmd_resp != cmd+1` 的情况，
> `Transaction.waitRsp` 已优先以 `CmdSeq` 匹配响应，并在 cmd 不符合 `cmd+1` 时仅输出告警但仍尝试解码。


