package transaction

import "runtime"

type SerialKeyMode uint8

const (
	SerialKeyModeNone SerialKeyMode = iota
	SerialKeyModeUID
	SerialKeyModeRouterID
)

func (m SerialKeyMode) String() string {
	switch m {
	case SerialKeyModeUID:
		return "uid"
	case SerialKeyModeRouterID:
		return "router_id"
	default:
		return "none"
	}
}

type TransactionMgrConfig struct {
	MaxTrans         int32
	ShardCount       int
	SerialKeyMode    SerialKeyMode
	MaxPendingPerKey int
}

type TransactionMgrStats struct {
	ShardCount         int
	ActiveTransactions int64
	PendingPackets     int64
	DroppedPackets     int64
}

func DefaultShardCount() int {
	shardCount := runtime.GOMAXPROCS(0)
	if shardCount <= 0 {
		return 1
	}
	if shardCount > 32 {
		return 32
	}
	return shardCount
}

func normalizeConfig(cfg TransactionMgrConfig) TransactionMgrConfig {
	if cfg.MaxTrans <= 0 {
		cfg.MaxTrans = 1
	}
	if cfg.ShardCount <= 0 {
		cfg.ShardCount = DefaultShardCount()
	}
	if cfg.ShardCount <= 0 {
		cfg.ShardCount = 1
	}
	switch cfg.SerialKeyMode {
	case SerialKeyModeNone, SerialKeyModeUID, SerialKeyModeRouterID:
	default:
		cfg.SerialKeyMode = SerialKeyModeNone
	}
	if cfg.MaxPendingPerKey < 0 {
		cfg.MaxPendingPerKey = 0
	}
	return cfg
}
