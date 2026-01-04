package ssrpc

import (
	"github.com/Iori372552686/GoOne/lib/api/cmd_handler"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/golang/protobuf/proto"
)

// SendMsgBackWithCmd tries to send response with a specified cmd.
//
// - If underlying context supports SendMsgBackWithCmd, it will be used.
// - Otherwise it falls back to SendMsgBack (cmd+1 convention).
func SendMsgBackWithCmd(ctx cmd_handler.IContext, cmd g1_protocol.CMD, pbMsg proto.Message) {
	if ctx == nil || pbMsg == nil {
		return
	}
	if v, ok := any(ctx).(interface {
		SendMsgBackWithCmd(cmd g1_protocol.CMD, pbMsg proto.Message)
	}); ok {
		v.SendMsgBackWithCmd(cmd, pbMsg)
		return
	}
	ctx.SendMsgBack(pbMsg)
}


