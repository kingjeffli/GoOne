# 03. 与通用游戏后端平台能力对比

本篇不拿 GoOne 去和单纯的 Go 微服务脚手架比较，而是对照更接近“**产品化游戏平台**”的能力模型来分析。

参考能力画像包括：

- Nakama：用户、会话、实时、社交、排行榜、比赛、RPC、运维等集成度高；
- Photon/Realtime/Quantum 类平台：房间、会话、匹配、实时同步能力成熟；
- PlayFab / AccelByte / Heroic Labs / Beamable 一类平台：更强调账号、LiveOps、分析、运营工具、经济系统与服务化能力；
- GameLift / Agones / Fleet 管理体系：更强调房间/进程托管、伸缩、部署与调度。

---

## 1. 总体判断

GoOne 当前最强的是：

- 自研 runtime 能力；
- 微服务拆分；
- 串行事务业务模型；
- 多 transport 统一；
- bus + router + registry 的基础服务治理。

GoOne 当前最弱的是：

- 平台产品能力；
- 运维和 LiveOps 一体化能力；
- 会话与 presence 产品化能力；
- 持续交付与 SRE 体系；
- 安全、审计、反作弊、运营后台、分析链路。

也就是说：

> **GoOne 更像“游戏后端框架内核”，而不是“即开即用的游戏后端平台”。**

---

## 2. 能力矩阵

| 能力域 | GoOne 当前状态 | 主流平台期望 | 差距等级 | 备注 |
|---|---|---|---|---|
| 账号与认证 | 有基础 HTTP 签名、部分登录接入 | 完整账号、设备、游客、第三方登录、Token 生命周期 | 高 | 当前更多是业务自接 |
| 会话治理 | `connsvr` 负责连接与简单会话映射 | 正式 session 模型、恢复、过期、踢线、多端策略 | 高 | 网关仍偏转发型 |
| 实时 RPC/消息 | `ssrpc` 很强 | 平台通常内建更强的统一实时语义 | 中 | 这是 GoOne 强项之一 |
| 房间/匹配 | 有 `roomcentersvr` 与房间中心方向 | 匹配、房态、席位、分配、生命周期编排完整 | 高 | 当前只有房间中心雏形 |
| Presence/在线状态 | 零散存在 | 统一在线状态、订阅、好友在线、房间 presence | 高 | 平台常见标配 |
| 社交能力 | 常量里有 friend/chat/guild 等概念 | 朋友、公会、聊天、屏蔽、推荐、通知 | 高 | 当前活跃服务集中在核心链路 |
| 排行榜/赛事 | 常量中有 rank | 平台通常具备排行榜、赛季、奖池、重置策略 | 高 | 需要产品化抽象 |
| 配置与 LiveOps | 有配置中心基础、web 管理入口 | feature flag、AB、灰度、运营活动编排 | 高 | 目前偏配置加载，不是 LiveOps |
| 分析与审计 | 基本缺失 | 用户行为、经济日志、GM 操作审计、合规留痕 | 高 | 非常明显缺口 |
| 可观测性 | metrics/health/pprof 有基础 | logs + metrics + traces + SLO + alert + runbook | 中高 | tracing 缺失是关键短板 |
| 部署与弹性 | 有脚本、Ansible、环境配置 | 自动伸缩、房间编排、容量管理、滚动发布 | 高 | 更像工程脚本，不像平台控制面 |
| 后台与权限 | `web_svr` 可做后台接口 | RBAC、审计、租户/角色体系、操作留痕 | 高 | 平台化必须补 |
| 反作弊/风控 | 基本未见 | 风控、黑名单、设备信誉、频控、异常行为检测 | 高 | 平台常见必要能力 |
| 异步作业/事件流 | bus 存在，但更偏 RPC 路由基础设施 | durable jobs、event sourcing、异步工作流 | 中高 | 目前缺“产品层工作流” |
| CI/CD 质量体系 | 无明显 CI 工作流 | 自动测试、lint、产物、部署门禁 | 高 | 研发平台能力缺口明显 |

---

## 3. 按能力域展开分析

## 3.1 认证与账号体系

### GoOne 当前情况

- `connsvr` 有登录接入和外部校验逻辑；
- `web_svr` 有签名校验场景；
- 但没有看到统一的账号域模型。

### 平台通常具备

- 游客登录
- 第三方平台登录
- 设备绑定
- Token 刷新与吊销
- 多端并发策略
- 黑名单与风险控制

### GoOne 缺口

- 缺少统一 session/token 生命周期；
- 缺少账号中心边界；
- 登录流程更像业务流程而不是平台能力。

### 建议落点

- 增加独立 `account/auth` 能力域；
- 把 `connsvr` 从“接入 + 登录业务”中解耦出来；
- 建立标准 token / session / device policy。

---

## 3.2 房间、匹配与编排能力

### GoOne 当前情况

- `roomcentersvr` 已经在正确方向上；
- 能管理 room list、tick、部分类似 quick start 的请求。

### 平台通常具备

- 匹配池
- 匹配策略
- 房间分配
- 房间回收
- 房间容量管理
- 房态可视化
- 区域/版本/玩法隔离

### GoOne 缺口

- 当前更像“房间中心服务”，不是“匹配与房间平台”；
- 缺少标准房间生命周期；
- 缺少房间进程编排视角；
- 缺少容量与伸缩联动。

### 建议落点

- 将 `roomcentersvr` 继续平台化：
  - room model
  - seat model
  - match pipeline
  - allocation pipeline
  - room metrics
- 长期可与 K8s/Agones/GameServer 调度模型衔接。

---

## 3.3 Presence / 在线状态 / 社交图谱

### GoOne 当前情况

- `connsvr` 维护连接；
- `infosvr` 可承载部分轻量信息；
- 但没有统一 presence 产品层。

### 平台通常具备

- 用户在线状态
- 房间 presence
- 好友在线订阅
- 最近在线时间
- 多终端策略
- 推送/通知联动

### GoOne 缺口

- 在线状态与连接状态还没正式分层；
- `connsvr` 的连接记录还不是 presence 产品；
- 缺少统一 presence 订阅/广播语义。

### 建议落点

- 引入 presence service 或 presence domain；
- 区分：
  - transport connection
  - logical session
  - online presence
  - room presence

---

## 3.4 LiveOps / feature flags / 运营能力

### GoOne 当前情况

- 有配置中心和 `web_svr`；
- 但更多是“配置加载/后台接口”，不是完整运营平台。

### 平台通常具备

- feature flag
- AB test
- 配置发布审批
- 活动开关
- 灰度发布
- 操作留痕
- 风险回滚

### GoOne 缺口

- 缺少 flag 系统；
- 缺少配置版本治理；
- 缺少变更审计；
- 缺少 GM/运营后台权限体系。

### 建议落点

- `web_svr` 不应只做 API，要逐步承接管理面；
- 增加 RBAC、审计、配置版本化；
- 把 feature flags 和运行时上下文接入 `ssrpc` middleware。

---

## 3.5 观测与 SRE 能力

### GoOne 当前情况

已有：

- `/healthz`
- `/readyz`
- `/metrics`
- `/debug/pprof/*`
- `router` metrics
- `transaction` metrics
- `ssrpc` metrics

### 缺失

- OpenTelemetry tracing
- 跨服务 trace propagation
- SLO 定义
- 业务级 dashboard 规范
- 告警策略
- runbook

### 平台通常具备

- metrics 是底线；
- traces 是微服务系统排障标配；
- 告警与运行手册是线上组织能力的一部分。

### 建议落点

- 先补 tracing；
- 再补 dashboard + alerts + runbooks；
- 最终按服务建立 SLI/SLO。

---

## 3.6 异步任务与事件平台

### GoOne 当前情况

- `bus` 很强，但主要用于服务通信；
- 没看到产品化的 job/event/workflow 层。

### 平台通常具备

- durable jobs
- delayed jobs
- scheduled tasks
- event bus
- replay/audit
- dead letter queue

### GoOne 缺口

- 当前 bus 偏底层基础设施；
- 缺少面向业务开发的异步作业抽象；
- 缺少可视化、失败重试、补偿机制。

### 建议落点

- 引入统一 async job framework；
- 将“消息总线能力”和“业务异步工作流能力”区分开。

---

## 3.7 安全、审计、风控、反作弊

### GoOne 当前情况

- 存在 HTTP 签名；
- 有敏感词处理；
- 但这还不是完整的安全域。

### 平台通常具备

- 操作审计
- 黑名单
- 限流
- 作弊检测
- 异常登录检测
- 经济系统风险监控
- 敏感操作审批/留痕

### GoOne 缺口

- 风控与安全能力基本缺位；
- 审计体系不足；
- 后台操作没有平台级留痕。

### 建议落点

- 最少先做：
  - audit log
  - admin RBAC
  - connection / uid rate limit
  - 风险事件埋点

---

## 4. 哪些差距不算问题，哪些必须尽快补

## 不必焦虑的差距

这些能力不一定要马上做成平台：

- 完整社交系统
- 完整经济系统平台
- 商业化赛事系统
- 全功能运营中台

原因是：

- 它们和业务类型相关；
- 不是所有游戏都需要；
- 可以在 runtime 稳定后逐步沉淀。

## 必须尽快补的差距

1. 生命周期与关闭治理
2. tracing
3. CI/CD 与测试分层
4. session/presence 边界
5. room/match 的平台化抽象方向
6. admin/RBAC/audit 的最小闭环

---

## 5. Build / Buy / Internalize 建议

## 适合内部继续建设的

- `ssrpc`
- `transaction`
- `router`
- `roomcentersvr` 的房间编排内核
- `bootstrap` 生命周期体系

因为这些能力与当前架构深度绑定，是 GoOne 的独特价值所在。

## 适合优先借鉴成熟标准实现的

- tracing：OpenTelemetry
- dashboard/metrics：Prometheus + Grafana
- CI/CD：GitHub Actions / GitLab CI / Jenkins pipeline
- room/fleet 托管：可借鉴 Agones/GameLift 模型
- RBAC / 审计：尽量采用通用后台能力模型

## 不建议短期闭门重造的

- 全自研可视化运维平台
- 全自研风控平台
- 全自研分析平台
- 全自研全链路 tracing 协议

---

## 6. 最终结论

从“平台能力”角度看，GoOne 当前并不弱，只是能力分布不均：

- **底层 runtime 很强**
- **平台产品层偏弱**

它最适合走的路线不是照抄 Nakama 或 PlayFab，而是：

> **保留 GoOne 在“串行事务 + 路由 + 多 transport runtime”上的特色，把缺失的平台能力一层层补齐。**

如果做到这一点，GoOne 最终会更像：

- 一个为中重度游戏后端量身打造的自研平台；
- 而不是一个只会转发消息的服务框架。
