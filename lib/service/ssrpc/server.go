package ssrpc

import (
	"github.com/Iori372552686/GoOne/lib/api/cmd_handler"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/golang/protobuf/proto"
)

// MethodDesc describes one SSPacket-exposed RPC method.
//
// This is primarily used by code generation to keep wrapper logic consistent.
type MethodDesc struct {
	Cmd     g1_protocol.CMD
	CmdResp g1_protocol.CMD // 0 means default Cmd+1
	OneWay  bool
	UIDLock bool // reserved for Phase A+
	Name    string
}

// WrapUnary returns a TransactionMgr-compatible command handler.
//
// It:
// - wraps cmd_handler.IContext into *ssrpc.Context
// - unmarshals request via ctx.ParseMsg (legacy-compatible)
// - executes middleware chain
// - maps error via ToErrorCode
// - auto-replies unless OneWay
func WrapUnary(desc MethodDesc, mws []Middleware, newReq func() proto.Message, invoke func(ctx *Context, req proto.Message) (proto.Message, error)) cmd_handler.CmdHandlerFunc {
	// IMPORTANT: never mutate caller-provided middleware slice in the returned closure.
	// Otherwise UIDLock injection may accumulate across requests if slice capacity allows.
	if desc.UIDLock {
		base := append([]Middleware(nil), mws...)
		mws = append(base, UIDLock())
	}
	return func(c cmd_handler.IContext, data []byte) g1_protocol.ErrorCode {
		if c == nil {
			return g1_protocol.ErrorCode_ERR_INTERNAL
		}
		ctx := WrapIContext(c, desc.Cmd)

		req := newReq()
		if err := ctx.ParseMsg(data, req); err != nil {
			ctx.Warningf("ssrpc parse failed err=%v", err)
			return g1_protocol.ErrorCode_ERR_MARSHAL
		}

		h := Handler(invoke)
		if len(mws) > 0 {
			h = Chain(mws...)(h)
		}

		rspAny, err := h(ctx, req)
		if err != nil {
			return ToErrorCode(err)
		}
		if desc.OneWay {
			_ = rspAny
			return g1_protocol.ErrorCode_ERR_OK
		}
		if rspAny != nil {
			cmdResp := desc.CmdResp
			if cmdResp == 0 {
				cmdResp = g1_protocol.CMD(uint32(desc.Cmd) + 1)
			}
			SendMsgBackWithCmd(ctx, cmdResp, rspAny)
		}
		return g1_protocol.ErrorCode_ERR_OK
	}
}


