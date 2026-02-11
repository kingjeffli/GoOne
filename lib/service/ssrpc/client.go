package ssrpc

import (
	"github.com/Iori372552686/GoOne/lib/api/cmd_handler"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/golang/protobuf/proto"
)

// SvrTypeFromCmd extracts the server type from a CMD value (bits [23:16]).
//
// This matches the convention in module/misc.ServerTypeInCmd but lives in
// the ssrpc package so that generated client stubs do not depend on module/misc.
func SvrTypeFromCmd(cmd g1_protocol.CMD) uint32 {
	return uint32(cmd) >> 16 & 0xff
}

// CallByCmd performs a synchronous RPC call, automatically deriving the target
// server type from the CMD value (bits [23:16]).
//
// This is the primary helper used by generated Client stubs for request/response
// methods (one_way=false).
func CallByCmd(ctx cmd_handler.IContext, cmd g1_protocol.CMD, req, rsp proto.Message) error {
	return ctx.CallMsgBySvrType(SvrTypeFromCmd(cmd), cmd, req, rsp)
}

// SendByCmd sends a fire-and-forget message, automatically deriving the target
// server type from the CMD value (bits [23:16]).
//
// This is the primary helper used by generated Client stubs for one-way methods
// (one_way=true).
func SendByCmd(ctx cmd_handler.IContext, cmd g1_protocol.CMD, req proto.Message) error {
	return ctx.SendMsgByServerType(SvrTypeFromCmd(cmd), cmd, req)
}

// CallByCmdWithRouter performs a synchronous RPC call with an explicit routerId,
// automatically deriving the target server type from the CMD value.
//
// Use this when the target server instance is determined by a routing key
// (e.g. room ID) rather than the default UID-based routing.
func CallByCmdWithRouter(ctx cmd_handler.IContext, routerId uint64, cmd g1_protocol.CMD, req, rsp proto.Message) error {
	return ctx.CallMsgByRouter(SvrTypeFromCmd(cmd), routerId, cmd, req, rsp)
}

// SendByCmdWithRouter sends a fire-and-forget message with an explicit routerId.
func SendByCmdWithRouter(ctx cmd_handler.IContext, routerId uint64, cmd g1_protocol.CMD, req proto.Message) error {
	return ctx.SendMsgByRouter(SvrTypeFromCmd(cmd), routerId, cmd, req)
}
