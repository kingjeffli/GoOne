package transaction

import "runtime"

type TransactionMgrConfig struct {
	MaxTrans         int32
	ShardCount       int
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
	if cfg.MaxPendingPerKey < 0 {
		cfg.MaxPendingPerKey = 0
	}
	return cfg
}
