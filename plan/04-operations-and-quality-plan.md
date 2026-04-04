# 04. 运维、质量与工程治理计划

这一篇关注的不是业务架构本身，而是：

- 如何把当前项目变得更容易构建、测试、上线、排障；
- 如何把“能运行”提升到“能稳定交付”。

---

## 1. 当前工程状态判断

## 已经有的基础

- 有统一入口 `main.sh`；
- 有 `deploy/` 与 `etc/env/`；
- 有 admin server 与 metrics；
- 有一定单元测试；
- 有 proto 生成与检查工具；
- 有 registry/config/bus 抽象。

## 当前明显短板

- `main.sh` 与真实目录结构存在漂移；
- `build.sh` 仍保留历史包袱，且未覆盖全部活跃服务；
- 缺少 CI workflow；
- 缺少 lint 规则与统一格式化门禁；
- 缺少集成测试基础设施；
- 缺少 tracing、SLO、告警、runbook；
- 缺少统一构建产物规范。

结论：

> 当前工程体系已经有“工具雏形”，但还没形成“可靠交付链”。

---

## 2. 构建与脚本治理

## 2.1 当前问题

### 问题 1：`main.sh` 与目录结构漂移

观察到：

- `main.sh` 某些逻辑仍假设 `env/*`；
- 实际仓库环境目录是 `etc/env/*`；
- 这会让入口脚本的可信度下降。

### 问题 2：`build.sh` 没跟上活跃服务集

例如：

- `roomcentersvr` 不在默认活跃构建目标中；
- 仍存在旧服务构建函数；
- 脚本与 `AGENTS.md` 里的“当前活跃服务”不完全一致。

### 问题 3：入口过多但规范不够统一

当前存在：

- `main.sh`
- `build.sh`
- `scripts/*.ps1`
- `scripts/*.sh`

如果不收敛，会逐步形成多套真相。

## 2.2 建议

### 建议 1：确立唯一主入口

建议统一：

- 对开发者：`main.sh` / PowerShell 对应入口
- 对 proto：`tools/cmd/genproto`
- 对构建：统一由一个入口调度

### 建议 2：重构 `build.sh`

建议目标：

- 只保留当前活跃服务；
- 明确 `build all` 和 `build <service>`；
- 输出统一到 `build/`；
- 自动创建目录；
- 支持 Windows/PowerShell 等价命令。

### 建议 3：给脚本做“契约检查”

例如：

- 服务目录存在性检查
- 配置文件存在性检查
- 环境目录存在性检查
- 依赖命令存在性检查

---

## 3. 本地开发体验（DX）计划

## 当前问题

- 新人需要自己理解多套入口；
- 不同服务运行方式靠经验；
- Windows 使用路径虽有说明，但体验不完整；
- 缺少一键“本地起核心链路”的命令。

## 建议

### 建议 1：提供一套明确的本地开发模式

例如：

- `dev core`：起 `connsvr + mainsvr + infosvr + mysqlsvr + roomcentersvr`
- `dev web`：起 `web_svr`
- `dev deps`：起中间件依赖

### 建议 2：标准化配置 profile

建议至少维护：

- local
- ide
- ci
- staging-like

### 建议 3：为 Windows 补齐等价工具链

仓库已经有 PowerShell proto 脚本，这是好方向。建议进一步做到：

- PowerShell build 入口
- PowerShell doctor 入口
- 明确哪些命令必须在 bash/WSL 下执行

---

## 4. CI/CD 基线建设

## 4.1 当前缺口

未见：

- `.github/workflows/*`
- `golangci-lint` 配置
- 标准化 pipeline

## 4.2 最小可行 CI

建议第一版 CI 至少包含：

1. `go fmt` / `gofmt -w` 检查
2. `go vet`
3. 单元测试
4. `go build` 核心服务
5. `go run ./tools/cmd/genproto` 或 `check-genproto`
6. 配置文件基础校验

### 推荐流水线阶段

- `lint`
- `unit-test`
- `build`
- `proto-check`
- `package`

## 4.3 发布链路建议

长期建议补齐：

- 版本号/commit hash 注入
- artifact 归档
- staging 冒烟测试
- canary/灰度
- 回滚策略

---

## 5. 测试体系分层

## 当前情况

优点：

- `lib/service/ssrpc`
- `lib/service/transaction`
- `lib/service/router`
- `lib/service/bootstrap`
- `common/gconf`

这些区域已有一定测试覆盖。

缺点：

- 集成测试基础薄弱；
- 缺少 testcontainers / miniredis / sqlmock 一类设施；
- 很多测试仍然依赖真实中间件或静态环境。

## 建议测试分层

### L1：纯单元测试

目标：

- 无外部依赖
- 跑得快
- PR 必过

### L2：组件集成测试

建议引入：

- `miniredis`
- `sqlmock`
- `httptest`
- `bufconn`（gRPC 已有）
- 必要时 `testcontainers-go`

### L3：服务链路测试

例如验证：

- `connsvr -> router -> mainsvr`
- `web_svr` HTTP/gRPC 接口行为
- room center 核心流程

### L4：环境冒烟测试

用于部署前检查：

- 配置加载
- admin 端口可达
- 核心依赖可用
- 基础 RPC/HTTP 通路可通

---

## 6. 观测性建设计划

## 6.1 当前已有

- `/metrics`
- `/healthz`
- `/readyz`
- `/debug/pprof`
- router/transaction/ssrpc metrics

## 6.2 明显缺失

### 缺 tracing

`lib/service/ssrpc/trace.go` 只是接口层，没有真正接 OpenTelemetry。

### 缺统一结构化日志上下文

当前日志有一定上下文，但还缺：

- trace_id
- span_id
- request_id
- session_id
- shard_id

### 缺运行手册

有 metrics 还不够，还需要：

- 看到某个指标高了怎么办
- 某个服务不 ready 怎么排
- 队列堆积时先看哪里

## 6.3 建议实施顺序

### 第一阶段

- 接入 OpenTelemetry
- 统一 logger fields
- 补 Dashboard

### 第二阶段

- 为核心服务定义 SLI/SLO
- 加告警
- 写 runbook

### 第三阶段

- 把业务指标也规范化
- 例如在线人数、房间数、活跃事务数、慢角色数、掉包数等

---

## 7. 生命周期与优雅停机规范

## 当前问题

- `web_svr` 对 HTTP/gRPC 有 shutdown；
- 但 router/transaction/bus/watcher 等没有统一 stop contract；
- 服务退出更多依赖“进程结束”。

## 建议规范

每个长期驻留组件都应实现：

- `Start(ctx)`
- `Shutdown(ctx)`
- `Health()`
- `Ready()`

并在 `bootstrap.ServiceApp` 中统一收口。

建议关闭顺序：

1. 标记 not ready
2. 停止接新流量
3. 停止接收新消息
4. drain 在途事务
5. 关闭 listener / watcher / bus / db
6. flush logs / final metrics

---

## 8. 运维文档与值班资产

平台成熟度不只看代码，也看文档资产。

建议补齐：

- 服务清单与职责
- 端口清单
- 配置说明
- 依赖图
- 常见故障 SOP
- 发布 SOP
- 回滚 SOP
- 扩容/缩容 SOP

---

## 9. 最小治理落地包

如果只允许近期做一轮工程治理，建议最少完成以下事项：

### 包 1：构建入口收敛

- 修正 `main.sh`
- 清理 `build.sh`
- 补 PowerShell 等价入口

### 包 2：CI 基线

- fmt
- vet
- unit test
- build
- proto check

### 包 3：观测性升级

- tracing
- logger context
- dashboard

### 包 4：优雅停机

- router close
- bus close
- transaction shutdown
- bootstrap 收口

---

## 10. 最终结论

从工程治理角度看，GoOne 当前不是“没有工具”，而是“工具还没被收敛成生产级规范”。

所以本阶段最值得做的事情不是继续加很多业务模块，而是：

> **把构建、测试、观测、停机、发布这几条链先打通。**

一旦这些基础链路稳住，后面无论做房间平台、LiveOps、RBAC 还是匹配系统，成本都会显著降低。
