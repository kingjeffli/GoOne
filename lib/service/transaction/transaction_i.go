package transaction

import (
	"github.com/Iori372552686/GoOne/lib/api/cmd_handler"
	"github.com/Iori372552686/GoOne/lib/api/sharedstruct"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
)

type ITransactionMgr interface {
	InitAndRun(maxTrans int32, useUidLock bool, maxUidPendingPacket int)
	InitAndRunWithConfig(cfg TransactionMgrConfig)

	RegisterCmd(cmd g1_protocol.CMD, cmdHandler cmd_handler.CmdHandlerFunc)
	ProcessSSPacket(packet *sharedstruct.SSPacket)
	StatsSnapshot() TransactionMgrStats
}
