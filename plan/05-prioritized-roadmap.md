# 05. 分阶段优先级路线图

本路线图把前面几份分析收敛成可执行计划，目标不是一次性“大重构”，而是：

- 先稳主干；
- 再补平台能力；
- 最后再做更完整的产品化能力。

---

## 1. 路线图原则

排序原则按以下维度综合判断：

1. **风险降低**：是否直接影响稳定性
2. **影响范围**：是否覆盖所有服务
3. **投入产出比**：短期收益是否明显
4. **后续解锁能力**：是否为下一阶段铺路
5. **对团队效率的改善**：是否减少日常摩擦成本

---

## 2. 三阶段策略概览

| 阶段 | 主题 | 目标 |
|---|---|---|
| Phase 1 | 稳定内核 | 解决空转、生命周期、脚本漂移、CI 缺失等基础问题 |
| Phase 2 | 平台化中层 | 把 tracing、配置热更边界、会话/presence、房间编排能力补起来 |
| Phase 3 | 产品化上层 | LiveOps、RBAC、审计、异步作业、风控、分析等平台能力体系化 |

---

## 3. Phase 1：稳定内核（建议 2~6 周）

## 3.1 目标

让 GoOne 从“能跑”提升到“稳定、可构建、可观测、可关闭”。

## 3.2 必做事项

### 事项 A：重构 `application.Run()`

目标：

- 去掉 busy-spin
- 使用 ticker / channel 驱动
- 明确 `OnProc` 存在意义

收益：

- 立刻降低空闲 CPU
- 明显改善服务基础运行质量

优先级：**P0**

---

### 事项 B：建立统一 shutdown contract

目标：

- `router.Close()`
- `transaction.Shutdown(ctx)`
- `bus.Close()`
- watcher 正常退出
- `bootstrap` 统一收口

收益：

- 更可靠的发布/重启
- 更好的集成测试能力
- 更少 goroutine 泄漏与资源残留

优先级：**P0**

---

### 事项 C：修正 `main.sh` / `build.sh` / `etc/env` 漂移

目标：

- 统一实际目录结构
- 收敛构建入口
- 只保留活跃服务目标
- 明确 Windows/PowerShell 等价命令

收益：

- 新人上手成本降低
- 本地开发一致性提升
- 文档与仓库不再脱节

优先级：**P0**

---

### 事项 D：建立最小 CI

目标：

- fmt
- vet
- unit test
- build
- proto check

收益：

- 防止质量倒退
- 把“人肉守规范”变成“工具守规范”

优先级：**P0**

---

### 事项 E：事务层参数配置化

目标：

- timeout
- queue limit
- shard count
- max trans

收益：

- 线上治理手段更多
- 服务可按业务特征调优

优先级：**P1**

---

## 3.3 Phase 1 完成标准

满足以下条件才算结束：

- 服务空闲 CPU 显著下降；
- 服务可以优雅关闭；
- 构建入口统一；
- 核心包进入 CI；
- timeout / queue 等不再依赖硬编码。

---

## 4. Phase 2：平台化中层（建议 1~2 个季度）

## 4.1 目标

把当前的“运行时框架”升级成更像平台基础层的结构。

## 4.2 核心工作流

### 工作流 A：Tracing 与观测增强

目标：

- OpenTelemetry 接入 `ssrpc`
- trace id / span id 注入日志
- 建立 dashboard
- 关键告警与 runbook

依赖：Phase 1 CI 和生命周期基础完成。

优先级：**P1**

---

### 工作流 B：配置与 reload 体系化

目标：

- 定义冷/暖/热配置边界
- 明确哪些配置可 reload
- 引入 watcher 到真正运行时配置

优先级：**P1**

---

### 工作流 C：session / presence 正式建模

目标：

- transport connection
- logical session
- online presence
- room presence

收益：

- `connsvr` 角色更清晰
- 为好友/聊天/房间状态同步打基础

优先级：**P1**

---

### 工作流 D：房间/匹配编排能力升级

目标：

- 明确 room lifecycle
- seat / allocation 模型
- 匹配与分配流程
- 房间指标体系

收益：

- `roomcentersvr` 进入平台化轨道
- 更接近通用游戏后端平台能力

优先级：**P1 / P2**

---

### 工作流 E：服务角色再分层

目标：

- 明确哪些服务是单 shard
- 哪些服务是多 shard
- 哪些服务只做组合/网关
- 哪些服务是平台基础设施

收益：

- 避免后续边界继续模糊

优先级：**P2**

---

## 4.3 Phase 2 完成标准

- tracing 可用且能跨服务看链路；
- 会话与在线状态模型明确；
- 房间中心具备标准生命周期与指标；
- 配置 reload 行为被明确定义并被文档化。

---

## 5. Phase 3：产品化上层（建议长期持续）

## 5.1 目标

把 GoOne 从“框架 + 平台内核”继续提升到“可运营的平台产品”。

## 5.2 核心能力包

### 能力包 A：Admin / RBAC / 审计

目标：

- 后台角色权限
- GM 操作记录
- 配置修改留痕
- 高风险操作审批

优先级：**P1**

---

### 能力包 B：Feature flags / LiveOps

目标：

- feature flag
- AB test
- 活动配置
- 灰度发布
- 快速回滚

优先级：**P2**

---

### 能力包 C：异步作业平台

目标：

- durable jobs
- delayed jobs
- retries
- dead letters
- 业务工作流

优先级：**P2**

---

### 能力包 D：分析与风控

目标：

- 行为埋点
- 经济流水
- 安全事件
- 风险识别
- 反作弊接入点

优先级：**P2 / P3**

---

### 能力包 E：弹性与托管能力

目标：

- 服务摘流
- 容量标签
- 房间实例弹性
- 与 K8s/Agones/Fleet 模型衔接

优先级：**P3**

---

## 6. 快速收益项（Quick Wins）

这些事情建议最先做，因为投入不大但回报明显：

1. 修复 `application.Run()` 忙轮询
2. 统一 `main.sh` 与 `etc/env`
3. `build.sh` 清理历史服务目标
4. 给 transaction timeout 配置化
5. 增加 CI 基线
6. 为 admin 补 `/info` 和 `/components`
7. 将 tracing 接口接一个最小 OpenTelemetry 实现

---

## 7. 高风险但高价值的重构项

这些事情价值高，但建议在前置条件满足后再做：

1. transaction shutdown 改造
2. bus 生命周期统一抽象
3. session/presence 正式建模
4. 房间/匹配平台化重构
5. `web_svr` 的后台/开放接口分层

---

## 8. 推荐 ADR（架构决策记录）主题

以下主题建议单独写 ADR，而不是边做边改：

1. `application.Run` 新模型与 `OnProc` 语义
2. transaction shutdown 设计
3. session / presence 分层模型
4. `roomcentersvr` 与 gameplay server 的边界
5. tracing 方案（OTel）
6. config reload 边界
7. admin / RBAC / audit 的平台方案

---

## 9. 推荐实施顺序（最实用版本）

### Sprint 1

- 修主循环
- 清脚本
- 建最小 CI

### Sprint 2

- transaction 配置化
- router/bus/transaction 生命周期收口设计
- admin 扩展

### Sprint 3

- tracing 接入
- logger context
- dashboard + alerts

### Sprint 4~5

- session/presence 模型
- room lifecycle 模型
- `web_svr` 分层

### 后续长期

- RBAC / 审计
- feature flags
- 异步作业平台
- 分析/风控
- 托管与弹性能力

---

## 10. 最终建议

最合理的演进姿势不是“大改一遍”，而是：

> **先解决运行时和生命周期问题，再补 observability 和平台中层，最后才做产品化能力。**

换句话说：

- **Phase 1** 解决“能不能稳”
- **Phase 2** 解决“能不能像个平台”
- **Phase 3** 解决“能不能像个平台产品”

这条路线最现实，也最符合 GoOne 当前已有基础。
