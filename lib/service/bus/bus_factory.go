package bus

import (
	"fmt"

	"github.com/Iori372552686/GoOne/lib/api/logger"
)

// BusCtor creates a bus implementation.
// Return error instead of panicking, to keep factory robust.
type BusCtor func(selfBusId uint32, onRecvMsg MsgHandler, conf any) (IBus, error)

var busCtors = map[string]BusCtor{}

// RegisterBus registers a bus implementation by type string.
// Suggested types: rabbitmq, nsq, nats, kafka, rocketmq.
func RegisterBus(implType string, ctor BusCtor) {
	if implType == "" || ctor == nil {
		return
	}
	busCtors[implType] = ctor
}

// CreateBus creates a bus instance based on addr string.
// It parses addr to detect backend type and config, then uses registered impl ctors.
func CreateBus(selfBusId uint32, onRecvMsg MsgHandler, addr string) (IBus, error) {
	implType, cfg, err := ParseAddr(addr)
	if err != nil {
		logger.Errorf("CreateBus ParseAddr failed | implType:%v, selfBusId:0x%x  err| %v", implType, selfBusId, err)
		return nil, err
	}

	b, err := createBusByTypeE(implType, selfBusId, onRecvMsg, cfg)
	if err != nil {
		logger.Errorf("CreateBus failed {implType:%v, selfBusId:0x%x} | %v", implType, selfBusId, err)
		return nil, err
	}

	logger.Infof("CreateBus success | implType:%v  addr:%v  selfBusId:%s", implType, addr, IpIntToString(selfBusId))
	return b, nil
}

// createBusByTypeE is the internal helper that uses implType + typed config.
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
