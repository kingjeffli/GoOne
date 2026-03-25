package main

import (
	"strconv"

	mainsvrv1 "github.com/Iori372552686/GoOne/api/gen/game/mainsvr/v1"
	"github.com/Iori372552686/GoOne/lib/api/datetime"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/service/router"
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	"github.com/Iori372552686/GoOne/src/connsvr/globals"
	"github.com/Iori372552686/GoOne/src/connsvr/login"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/golang/protobuf/proto"
)

// ClientLoginGatewayImpl handles connsvr-side login pre-auth before forwarding
// the original LoginReq to mainsvr. The request arrives over the generated
// WS/CSPacket wrapper but the actual LoginRsp still comes back from mainsvr.
type ClientLoginGatewayImpl struct {
	mainsvrv1.UnimplementedMainC2SServiceSS
}

var _ mainsvrv1.MainC2SServiceSS = (*ClientLoginGatewayImpl)(nil)

func (s *ClientLoginGatewayImpl) Login(ctx *ssrpc.Context, req *g1_protocol.LoginReq) (*g1_protocol.LoginRsp, error) {
	if ctx == nil || ctx.IContext == nil {
		return nil, ssrpc.E(g1_protocol.ErrorCode_ERR_INTERNAL, "missing client packet context")
	}
	clientCtx, ok := any(ctx.IContext).(*clientPacketIContext)
	if !ok {
		logger.Errorf("conn login gateway: unexpected IContext type %T", ctx.IContext)
		return nil, ssrpc.E(g1_protocol.ErrorCode_ERR_INTERNAL, "invalid client packet context")
	}
	startTime := datetime.NowMs()
	if req == nil {
		return nil, ssrpc.E(g1_protocol.ErrorCode_ERR_MARSHAL, "missing login req")
	}

	if req.GetAccount() == "" || req.GetChannelId() == 0 {
		logger.Errorf("conn login gateway: account or ChannelId error")
		return nil, ssrpc.E(g1_protocol.ErrorCode_ERR_ARGV, "account or channel_id error")
	}

	ret, accUid := login.OnCheckAuthByAccSvr(req.Account, req.Token, req.ChannelId, req.LoginType)
	duration := datetime.NowMs() - startTime
	logger.Infof("conn login gateway: CheckAuthByAccSvr spent ms: %s | ret=%v",
		strconv.FormatInt(duration, 10), ret)

	if !ret {
		return nil, ssrpc.E(g1_protocol.ErrorCode_ERR_FAIL, "check auth by accsvr failed")
	}

	clientCtx.SetUid(accUid)
	zone := uint32(1) // TODO: derive from server config
	clientCtx.SetZone(zone)

	// Register/update the client connection mapping on the current transport (WS or TCP).
	client := clientCtx.ConnMgr().UpdateClientByUid(clientCtx.Conn(), accUid, zone)
	if client == nil {
		return nil, ssrpc.E(g1_protocol.ErrorCode_ERR_INTERNAL, "failed to register client connection")
	}

	body, err := proto.Marshal(req)
	if err != nil {
		return nil, ssrpc.Wrap(g1_protocol.ErrorCode_ERR_MARSHAL, "marshal login req", err)
	}

	// Forward the proto payload to mainsvr. mainsvr will send LoginRsp later.
	if err := router.SendMsgByConn(accUid, accUid, zone, clientCtx.Header().Cmd, 0, body, client.Ip, client.Port); err != nil {
		return nil, ssrpc.Wrap(g1_protocol.ErrorCode_ERR_INTERNAL, "forward login req to mainsvr", err)
	}

	return nil, nil
}

// regClientPacketHandlers registers connsvr-side CSPacket pre-auth handlers via the
// generated ws=true bindings.
func regClientPacketHandlers() {
	logger.Infof("register client packet dispatch handlers")
	mainsvrv1.RegisterMainC2SServiceToWS(globals.ClientPacketDispatcher, mainsvrv1.MainC2SServiceSServer{
		Impl: &ClientLoginGatewayImpl{},
		MW: []ssrpc.Middleware{
			ssrpc.Recover(),
			ssrpc.Logging(),
		},
	})
}
