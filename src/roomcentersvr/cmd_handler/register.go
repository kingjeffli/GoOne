package cmd_handler

import (
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/src/roomcentersvr/globals"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
)

// 所有的命令字对应的go需要在这里先注册
func RegCmd() {
	logger.Infof("register transaction commands")
	// Phase A migration:
	// These cmds are now registered via IDL-driven ssrpc wrappers in api/gen/... and
	// bound in src/roomcentersvr/app.go via RegisterRoomCenterInnerServiceToTransactionMgr.
	//
	// Keep this function to avoid breaking existing init flow while migration continues.
	_ = globals.TransMgr
	_ = g1_protocol.CMD(0)
}
