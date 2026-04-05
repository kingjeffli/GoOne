# 07. RPC 超时配置建议（proto / 服务组 / YAML 分层方案）

## 1. 结论先说

**可以做，而且你们仓库里其实已经有一半基础了。**

当前 `GoOne` 已经支持在 **单个 RPC 的 proto method option** 上声明：

- `timeout_ms`
- `auth`
- `sign`
- `trace_tags`
- `ws`
- `grpc`
- `cmd_name/cmd_enum`

也就是说，像下面这样加 `timeout_ms`，**语法层面已经成立**：

```proto
service MainC2SService {
  rpc Login(g1.protocol.LoginReq) returns (g1.protocol.LoginRsp) {
    option (goone.options.v1.ssrpc) = {
      cmd_name: "CMD_MAIN_LOGIN_REQ"
      ws: true
      timeout_ms: 800
      comment: "mainsvr login"
    };
  }
}
```

但是当前系统状态是：

1. **proto -> 代码生成 -> `MethodDesc.Timeout`** 这条链已经打通；
2. **运行时只是把 deadline 挂到了 `ssrpc.Context` 上**；
3. **下游同步调用的 3 秒等待仍然硬编码在 transaction 层**；
4. **还没有 service 级 / service-group 级默认超时声明**；
5. **还没有形成“method > service > YAML”的统一覆盖优先级**。

所以答案不是“能不能做”，而是：

> **完全可以做，而且最合理的做法不是只选 proto 或只选 YAML，而是做成“契约层 + 运行时层”的分层方案。**

---

## 2. 当前项目的真实状态

### 2.1 已经存在的能力

#### `api/proto/goone/options/v1/options.proto`

当前 `SsRpc` 已经定义了：

```proto
message SsRpc {
  uint32 cmd = 1;
  uint32 cmd_resp = 2;
  bool one_way = 3;
  bool uid_lock = 4;
  string cmd_name = 5;
  goone.cmd.v1.CMD cmd_enum = 6;
  bool auth = 7;
  bool sign = 8;
  repeated string trace_tags = 9;
  uint32 timeout_ms = 10;
  string comment = 15;
  string http_path = 20;
  string http_method = 21;
  bool ws = 30;
  bool grpc = 40;
  string grpc_service = 41;
}

extend google.protobuf.MethodOptions {
  SsRpc ssrpc = 61001;
}
```

说明：

- **单 RPC method 的超时声明字段已经存在**；
- 不是概念设计，而是已经进了 proto source-of-truth；
- 不是未来才能做，而是现在就能继续补完。

#### `tools/protoc-gen-goone/generate.go`

生成器已经会读取 `timeout_ms`，并把它写入：

```go
ssrpc.MethodDesc{
    Timeout: 1500 * time.Millisecond,
}
```

说明：

- **proto option 已经能进入生成代码**；
- 这部分不是空设计，测试也覆盖了。

#### `lib/service/ssrpc/context.go`

`ctx.ApplyTimeout(desc.Timeout)` 已经会把 handler context 包一层 `context.WithTimeout(...)`。

说明：

- **入口 handler 级 deadline 已存在**；
- HTTP / gRPC / WS / SSPacket wrapper 都会吃到这个 timeout。

---

### 2.2 当前还不完整的地方

#### 问题 1：timeout 只到了 handler context，没有打通“完整语义”

当前 `WrapUnary` / `WrapHTTPGin` / `WrapGRPCUnary` / `WrapWS` 都会做：

```go
ctx.ApplyTimeout(desc.Timeout)
```

但这不等于“整个调用链真的会按这个超时失败”。

因为：

- 业务 handler 是否真的检查 `ctx.Done()` / `ctx.Err()`，目前并不统一；
- 某些下游同步调用并不自动继承这个 deadline；
- timeout error 到 `ErrorCode` 的映射还不够统一；
- transaction wait response 仍是固定 3 秒等待。

#### 问题 2：下游同步调用仍被 transaction 的 3 秒硬编码主导

在：

- `lib/service/transaction/transaction_impl.go`

当前仍有：

```go
return t.waitRsp(..., time.Second*3, ...)
```

这说明：

- 即便入口 proto method 配了 `timeout_ms: 800`；
- 如果 handler 里同步调用别的服务，底层等待仍可能默认走 3 秒；
- **入口 SLA 和内部调用等待策略目前不是一个体系**。

#### 问题 3：没有 service-group 级默认值

当前只有 `MethodOptions` 扩展，没有：

- `ServiceOptions` 扩展；
- `FileOptions` 扩展；
- 生成器层 service 默认值继承逻辑。

所以现在只能：

- 每个 method 单独写；
- 或者完全靠 YAML；
- 中间缺了一层“这组 RPC 默认 800ms，仅个别 override”的表达能力。

---

## 3. 我对这个需求的判断

## 3.1 你的方向是对的

你提到的目标，本质上是：

1. **单个 RPC 可声明超时**；
2. **同一服务组可声明默认超时**；
3. **尽量把契约写在 proto，而不是散落在代码里**；
4. **又要保留 YAML 作为运行时 fallback/override**。

这套思路比“只放 YAML”更强，原因是：

- proto 是接口契约，天然适合声明 **接口默认 SLA / 预期行为**；
- generator 可以把约束编译进 wrapper，减少人工漏配；
- 文档、代码、生成物保持同源；
- 与 go-zero 的“按 rpc / service 配 timeout”理念一致，但更适合 GoOne 现有模型。

---

## 3.2 但不能把所有 timeout 都塞进 proto

这是最关键的边界。

**不是所有 timeout 都适合放在 proto 里。**

建议把 timeout 分成三类：

### A. 接口契约级 timeout

这是“这个 RPC 正常应该在多久内完成”的默认 SLA。

适合放在：

- `MethodOptions`
- `ServiceOptions`

例如：

- 登录 800ms
- 心跳 200ms
- 房间列表 1500ms

这类 timeout **应该进 proto**。

### B. 进程运行时级 timeout

这是某个服务实例在某个环境里的运行时调优参数。

例如：

- transaction `wait_rsp_timeout_ms`
- dispatch timeout
- queue/pending timeout
- graceful shutdown drain timeout
- admin probe timeout

这类参数受环境、部署规模、链路延迟影响很大，**不建议只放 proto**。

这类更适合：

- `common/gconf/config.go`
- 各服务 YAML

### C. 下游调用级 timeout / 重试策略

这是“handler 里调用下游服务时等多久、重试几次”的策略。

这类最容易和业务耦合：

- `Login -> mysqlsvr` 也许 500ms
- `QuickStart -> roomcentersvr` 也许 1200ms
- 某些链路允许降级，某些不允许

这类可以有默认值，但**最终最好允许 call-site 覆盖**。

所以这里不能只靠 proto method 自己的一层 timeout。

---

## 4. 最推荐的方案：三层优先级模型

我建议最终做成下面这个优先级：

```text
MethodOptions(timeout_ms)
  > ServiceOptions(default_timeout_ms)
  > YAML(service runtime default)
  > framework built-in default
```

也就是：

1. **单 RPC method 显式声明优先级最高**；
2. 没写 method，就继承 **proto service 默认值**；
3. service 也没写，就落到 **服务进程 YAML 默认值**；
4. YAML 也没写，再用框架默认值。

这个模型的优点是：

- 契约层与运行时层职责清晰；
- 绝大多数接口不需要重复写 timeout；
- 少数慢接口可以 override；
- 运维仍有最后兜底；
- 兼容你现在的 P0-5“关键参数配置化”方向。

---

## 5. 建议的 proto 设计

## 5.1 第一层：保留现有 method 级 `timeout_ms`

这个已经有了，不建议推倒重来。

继续保留：

```proto
rpc Login(LoginReq) returns (LoginRsp) {
  option (goone.options.v1.ssrpc) = {
    cmd_name: "CMD_MAIN_LOGIN_REQ"
    timeout_ms: 800
    ws: true
    comment: "mainsvr login"
  };
}
```

这层适合：

- 少数关键接口单独调优；
- 对外 SLA 清晰的接口；
- 明显比同组其他接口快/慢很多的接口。

---

## 5.2 第二层：新增 service 级 option，作为“srv组默认值”

你说的“srv组”在 GoOne 里，**最自然的表达单位就是 proto `service`**。

因为：

- generator 本来就是按 proto service 生成 wrapper；
- 比 package 粒度更直观；
- 比 YAML 手写 service 名匹配更稳；
- 更接近 go-zero 里按 service/组配置默认值的体验。

建议新增：

```proto
message SsRpcService {
  uint32 timeout_ms = 1;
}

extend google.protobuf.ServiceOptions {
  SsRpcService ssrpc_service = 61002;
}
```

使用方式：

```proto
service MainC2SService {
  option (goone.options.v1.ssrpc_service) = {
    timeout_ms: 800
  };

  rpc Login(g1.protocol.LoginReq) returns (g1.protocol.LoginRsp) {
    option (goone.options.v1.ssrpc) = {
      cmd_name: "CMD_MAIN_LOGIN_REQ"
      ws: true
      comment: "mainsvr login"
    };
  }

  rpc GetRoomList(g1.protocol.RoomListReq) returns (g1.protocol.RoomListRsp) {
    option (goone.options.v1.ssrpc) = {
      cmd_name: "CMD_MAIN_GAME_ROOM_LIST_REQ"
      timeout_ms: 1500
      comment: "room list may be slower"
    };
  }
}
```

含义：

- `MainC2SService` 默认 800ms；
- `Login` 没单独配置，就吃 800ms；
- `GetRoomList` 单独 override 为 1500ms。

这就是最接近你要的“每个 rpc、srv组可配置”的方式。

---

## 5.3 第三层：不要把 transaction/bus 运行时参数也塞进 method option

例如这些字段，我**不建议**放到 `SsRpc` method option 里：

- queue length
- trans shard count
- trans manager max pending
- global dispatch wait timeout
- shutdown timeout
- service discovery timeout

原因：

- 它们不是接口契约，而是进程运行参数；
- 同一个 proto 在 dev/staging/prod 可能差异很大；
- 放进 proto 会把部署策略和接口契约绑死；
- generator 不应该开始承担太重的 infra/runtime 配置职责。

这些更应该放到：

- `common/gconf/config.go`
- `server_conf.yaml`
- 各服务 `runtime` / `capacity` 段

---

## 6. 比“只做 proto timeout_ms”更好的增强方案

如果你想比 go-zero 再走得更稳一点，我建议不是单纯一个 `timeout_ms`，而是把“入口超时”和“下游调用超时”分开。

### 6.1 为什么要分开

一个 RPC handler 的总预算，和它内部某次下游调用等待，不一定相同。

例如：

- `Login` 总预算 800ms；
- 其中访问 `infosvr` 最多 150ms；
- 访问 `mysqlsvr` 最多 300ms；
- 剩下时间给本地逻辑和编码。

如果只有一个 `timeout_ms`：

- 容易误以为所有内部调用都自动继承并正确切分；
- 实际上当前 GoOne 不是这样；
- 最后会出现“入口 800ms，但内部 waitRsp 还是 3 秒”的错觉。

---

### 6.2 更优雅的分层命名建议

#### proto 契约层

```proto
message SsRpc {
  uint32 timeout_ms = 10; // 整个 handler 的默认 deadline
}

message SsRpcService {
  uint32 timeout_ms = 1; // 该 proto service 下所有 method 的默认 deadline
}
```

#### YAML 运行时层

```yaml
mainsvr:
  runtime:
    ssrpc:
      default_timeout_ms: 1000
    transaction:
      wait_rsp_timeout_ms: 3000
      dispatch_timeout_ms: 3000
  capacity:
    trans_shard_count: 32
```

#### 可选：调用侧增强（后续阶段）

在 `ssrpc` client helper 或 transaction call helper 增加：

- `CallOption{ Timeout: ... }`
- `CallWithOptions(...)`

这样 handler 内部真的需要覆盖下游等待时，就能显式写出来。

---

## 7. 推荐的落地顺序

## P0：先承认现状，别一次做太大

### 第一步：正式启用 method `timeout_ms`

这一步实际上几乎已经具备，只差：

1. 在实际 proto 里开始使用；
2. 补 runtime timeout 行为验证；
3. 统一 timeout error 映射；
4. 让共享层测试覆盖真实 handler 超时表现。

这一步风险最小。

### 第二步：补 service 级默认 option

新增：

- `ServiceOptions` extension
- generator 继承逻辑
- 测试

这是你要的“srv组默认 timeout”的核心落地点。

### 第三步：把 transaction 的 3 秒硬编码改为 YAML 可配置

这一步不该拖，因为否则 proto timeout 很容易变成“看上去能配，实际上链路里不一致”。

建议新增到：

- `common/gconf/config.go`
- 各服务 `runtime/capacity`
- `transaction` config struct

例如：

- `wait_rsp_timeout_ms`
- `dispatch_timeout_ms`
- `max_pending_per_key`
- `shard_count`

### 第四步：再考虑 call-site override

如果未来要做更细粒度的跨服务治理，再补：

- `CallWithTimeout`
- `CallOption`
- retry / hedging / fallback（若需要）

这一步不适合现在一口气做完。

---

## 8. 我对三种方案的取舍建议

## 8.1 方案 A：只用 proto method option

### 优点

- 最直观；
- 文档和代码契约统一；
- generator 天然支持；
- 改动面小。

### 缺点

- 重复配置多；
- 缺少 service/group 默认值；
- runtime 层参数仍然无处承接；
- 无法表达环境差异。

### 结论

**不够。能用，但不够优雅。**

---

## 8.2 方案 B：method + service(proto) + YAML fallback

### 优点

- 兼顾契约与运行时；
- 支持单 RPC override；
- 支持 service-group 默认；
- 保留环境兜底；
- 最适合当前 GoOne 架构。

### 缺点

- 需要扩展 generator；
- 需要定义优先级；
- 需要补测试和文档。

### 结论

**这是我最推荐的方案。**

---

## 8.3 方案 C：只用 YAML

### 优点

- 改动简单；
- 运维友好；
- 环境覆盖方便。

### 缺点

- 契约层信息丢失；
- proto 文档看不出接口 SLA；
- 容易配漏；
- 跨项目生成时不可见；
- 不符合你想要的“像 go-zero 一样声明式”体验。

### 结论

**只能作为最后兜底，不建议作为主方案。**

---

## 9. 具体建议：你现在最应该怎么做

我建议你们直接定这个规则：

### 规则 1：proto method 可声明 `timeout_ms`

保留现有字段，不改语义。

### 规则 2：新增 proto service 级 `ssrpc_service.timeout_ms`

把它作为“srv组默认值”。

### 规则 3：YAML 只承担运行时 fallback 和 infra 参数

例如：

- 服务默认 timeout fallback
- transaction wait_rsp timeout
- dispatch timeout
- queue/pending/shard

### 规则 4：明确优先级

```text
method timeout_ms
> service default timeout_ms
> yaml default_timeout_ms
> framework default
```

### 规则 5：不要混淆“入口 handler deadline”和“内部同步调用等待”

这两个必须分开治理。

否则会出现：

- proto 上配了 800ms；
- handler 内部还在等 3s；
- 业务方以为 timeout 已生效；
- 实际上只是半生效。

---

## 10. 建议改动的代码位置

如果要正式落地，建议关注这些位置：

### proto / 生成器

- `api/proto/goone/options/v1/options.proto`
- `tools/protoc-gen-goone/generate.go`
- `tools/protoc-gen-goone/generate_test.go`

### runtime timeout 行为

- `lib/service/ssrpc/context.go`
- `lib/service/ssrpc/errors.go`
- `lib/service/ssrpc/http.go`
- `lib/service/ssrpc/grpc.go`
- `lib/service/ssrpc/ws.go`
- `lib/service/ssrpc/server.go`

### transaction / 下游等待超时

- `lib/service/transaction/transaction_impl.go`
- `lib/service/transaction/transaction_config.go`
- `lib/service/transaction/transaction_mgr_impl.go`

### 配置层

- `common/gconf/config.go`
- `common/gconf/server_conf.yaml`
- `etc/config/server_conf_ide.yaml`
- 各服务 `src/<service>/app.go`

---

## 11. 最终推荐方案（简版）

如果只给一句建议：

> **做成“proto method 覆盖 + proto service 默认 + YAML 运行时兜底”的三层模型，这是最适合 GoOne 当前架构的方案。**

如果再具体一点：

1. **现在就可以开始在 method option 上使用 `timeout_ms`**；
2. **尽快补 `ServiceOptions` 扩展，支持 srv 组默认 timeout**；
3. **transaction 的 3 秒硬编码必须同步配置化**；
4. **把 proto timeout 定义为“入口 handler 默认 deadline”，不要偷换成所有内部调用策略**；
5. **运行时参数继续放 YAML，不要全部塞进 proto**。

---

## 12. 我的明确结论

### 是否能做到“每个 RPC proto 声明时配置 timeout”？

**能，而且你们现在已经做了 50% 以上。**

### 是否适合继续往“srv组可配置”推进？

**非常适合，推荐新增 `ServiceOptions` 扩展。**

### 是否建议完全只用 YAML？

**不建议。YAML 只能作为最后 fallback。**

### 比 go-zero 风格更适合 GoOne 的方案是什么？

**不是单层超时配置，而是：**

- proto 管契约默认值；
- YAML 管运行时调优；
- transaction/下游调用单独治理；
- 最终形成统一优先级和继承模型。

---

## 13. 下一步建议

如果你同意，我下一步建议直接推进两件事：

1. **先补一版设计落地**：给 `options.proto` 加 `ServiceOptions` 扩展，并完善生成器继承逻辑；
2. **再补运行时一致性**：把 `transaction` 的 `waitRsp 3s` 改成配置化，避免 proto timeout 变成“半生效”。

这样就能把你现在提的需求，真正落成一个完整闭环，而不是只做表面配置入口。

