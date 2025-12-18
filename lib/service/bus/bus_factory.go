package bus

import (
	"fmt"

	"github.com/Iori372552686/GoOne/lib/api/logger"
)

// BusCtor creates a bus implementation.
// Return error instead of panicking, to keep factory robust.
type BusCtor func(selfBusId uint32, onRecvMsg MsgHandler, args ...any) (IBus, error)

var busCtors = map[string]BusCtor{}

// RegisterBus registers a bus implementation by type string.
// Suggested types: rabbitmq, nsq, nats, kafka, rocketmq.
func RegisterBus(implType string, ctor BusCtor) {
	if implType == "" || ctor == nil {
		return
	}
	busCtors[implType] = ctor
}

// CreateBus keeps backward compatibility (returns nil on error/panic).
// Prefer CreateBusE in new code.
func CreateBus(selfBusId uint32, onRecvMsg MsgHandler, addr string) (IBus, error) {
	parsedType, cfg, err := ParseAddr(addr)
	if err != nil {
		logger.Errorf("CreateBus ParseAddr failed | implType:%v, selfBusId:0x%x  err| %v", parsedType, selfBusId, err)
		return nil, err
	}

	b, err := createBusByTypeE(parsedType, selfBusId, onRecvMsg, cfg)
	if err != nil {
		logger.Errorf("CreateBus failed {implType:%v, parsedType:%v, selfBusId:0x%x} | %v", parsedType, parsedType, selfBusId, err)
		return nil, err
	}

	logger.Infof("CreateBus success | implType:%v  addr:%v  selfBusId:%s", parsedType, addr, IpIntToString(selfBusId))
	return b, err
}

// createBusByTypeE is the internal helper for the legacy CreateBus(implType,...,args...) signature.
func createBusByTypeE(implType string, selfBusId uint32, onRecvMsg MsgHandler, conf any) (IBus, error) {
	if implType == "" {
		implType = "rabbitmq"
	}
	ctor, ok := busCtors[implType]
	if !ok {
		return nil, fmt.Errorf("unknown bus implType=%q", implType)
	}
	return ctor(selfBusId, onRecvMsg, conf)
}

func init() {
	// RabbitMQ
	RegisterBus("rabbitmq", func(selfBusId uint32, onRecvMsg MsgHandler, args ...any) (IBus, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("rabbitmq requires addr string or RabbitMQConfig")
		}
		if s, ok := args[0].(string); ok {
			if s == "" {
				return nil, fmt.Errorf("rabbitmq addr is empty")
			}
			return NewBusImplRabbitMQ(selfBusId, onRecvMsg, s), nil
		}
		conf, ok := args[0].(RabbitMQConfig)
		if !ok {
			return nil, fmt.Errorf("rabbitmq arg must be string or RabbitMQConfig")
		}
		if conf.Addr == "" {
			return nil, fmt.Errorf("rabbitmq addr is empty")
		}
		return NewBusImplRabbitMQ(selfBusId, onRecvMsg, conf.Addr), nil
	})

	// NSQ
	RegisterBus("nsq", func(selfBusId uint32, onRecvMsg MsgHandler, args ...any) (IBus, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("nsq requires NSQConfig")
		}
		conf, ok := args[0].(NSQConfig)
		if !ok {
			return nil, fmt.Errorf("nsq arg must be NSQConfig")
		}
		return NewBusImplNsqMQ(selfBusId, onRecvMsg, conf), nil
	})

	// NATS
	RegisterBus("nats", func(selfBusId uint32, onRecvMsg MsgHandler, args ...any) (IBus, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("nats requires NatsConfig")
		}
		conf, ok := args[0].(NatsConfig)
		if !ok {
			return nil, fmt.Errorf("nats arg must be NatsConfig")
		}
		return NewBusImplNatsMQ(selfBusId, onRecvMsg, conf), nil
	})

	// Kafka
	RegisterBus("kafka", func(selfBusId uint32, onRecvMsg MsgHandler, args ...any) (IBus, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("kafka requires KafkaConfig")
		}
		conf, ok := args[0].(KafkaConfig)
		if !ok {
			return nil, fmt.Errorf("kafka arg must be KafkaConfig")
		}
		return NewBusImplKafkaMQ(selfBusId, onRecvMsg, conf), nil
	})

	// RocketMQ
	RegisterBus("rocketmq", func(selfBusId uint32, onRecvMsg MsgHandler, args ...any) (IBus, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("rocketmq requires RocketMQConfig")
		}
		conf, ok := args[0].(RocketMQConfig)
		if !ok {
			return nil, fmt.Errorf("rocketmq arg must be RocketMQConfig")
		}
		return NewBusImplRocketMQ(selfBusId, onRecvMsg, conf), nil
	})
}
