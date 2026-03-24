package ssrpc

import (
	"github.com/Iori372552686/GoOne/lib/api/cmd_handler"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/golang/protobuf/proto"
)

// WrapWS returns a CmdHandlerFunc for the WS (CSPacket) transport.
//
// Structurally identical to WrapUnary but stamps TransportWS on the Context
// so that middleware can distinguish WebSocket requests from SSPacket requests.
func WrapWS(desc MethodDesc, mws []Middleware, newReq func() any, invoke func(ctx *Context, req any) (any, error)) cmd_handler.CmdHandlerFunc {
	mws = prepareMW(mws, desc.UIDLock)
	return func(c cmd_handler.IContext, data []byte) g1_protocol.ErrorCode {
		if c == nil {
			return g1_protocol.ErrorCode_ERR_INTERNAL
		}
		ctx := WrapIContext(c, desc.Cmd)
		ctx.Transport = TransportWS
		applyDesc(ctx, &desc)

		reqAny := newReq()
		req, ok := reqAny.(proto.Message)
		if !ok || req == nil {
			ctx.Warningf("ssrpc.ws invalid req type: %T", reqAny)
			return g1_protocol.ErrorCode_ERR_INTERNAL
		}
		if err := ctx.ParseMsg(data, req); err != nil {
			ctx.Warningf("ssrpc.ws parse failed err=%v", err)
			return g1_protocol.ErrorCode_ERR_MARSHAL
		}

		rsp, err := buildHandler(mws, invoke)(ctx, req)
		if err != nil {
			return ToErrorCode(err)
		}
		if desc.OneWay {
			return g1_protocol.ErrorCode_ERR_OK
		}
		if rsp != nil {
			cmdResp := desc.CmdResp
			if cmdResp == 0 {
				cmdResp = g1_protocol.CMD(uint32(desc.Cmd) + 1)
			}
			SendMsgBackWithCmd(ctx, cmdResp, rsp)
		}
		return g1_protocol.ErrorCode_ERR_OK
	}
}
