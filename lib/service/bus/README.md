# Bus 中间件抽象层

`bus` 是服务器之间消息收发的中间件抽象层，统一提供：

- `Send(dstBusId, data1, data2)`：发送到目标 busId（内部会封装 `busPacketHeader`）
- `SetReceiver(onRecv)`：设置接收回调（约定 **单协程回调**）

## 工厂模式（两种风格）

### 1) 传统方式（兼容旧代码）

```go
// 兼容旧调用：CreateBus(type, selfBusId, onRecv, addr)
bus.CreateBus("rabbitmq", selfBusId, onRecv, "amqp://guest:guest@127.0.0.1:5672/")
```

### 2) register + addr 工厂（推荐）

`bus` 包内部使用 `RegisterBus` 注册各实现，并提供 `CreateBusE`（返回 error 更健壮）。

推荐：只传 addr，让 CreateBusE 自动识别类型并解析子配置：

```go
b, err := bus.CreateBusE(selfBusId, onRecv, "nats://127.0.0.1:4222?subject_prefix=bus&queue_group=g1")
_ = b
_ = err
```

## 协议一致性

所有实现都遵循 `busPacketHeader`：

- header(12 bytes) + data1 + data2
- 接收侧解析 header，校验 passCode，再回调 `onRecv(srcBusId, payload)`

特别说明：`nsq` 已统一为与 `rabbitmq/rocketmq` 相同的 header/payload 风格。


