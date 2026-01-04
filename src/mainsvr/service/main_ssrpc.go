package service

import (
	commonv1 "github.com/Iori372552686/GoOne/api/gen/game/common/v1"
	mainv1 "github.com/Iori372552686/GoOne/api/gen/game/main/v1"
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
)

// MainServiceImpl is an example IDL-driven ssrpc implementation for mainsvr.
//
// NOTE: This is scaffold-level logic to prove the IDL -> ssrpc -> TransactionMgr chain.
// Replace cmd values and business behavior with your real mainsvr commands later.
type MainServiceImpl struct{}

func (s *MainServiceImpl) Login(ctx *ssrpc.Context, req *mainv1.LoginReq) (*mainv1.LoginRsp, error) {
	_ = ctx
	rsp := &mainv1.LoginRsp{
		Ret: &commonv1.Ret{
			Code: int32(g1_protocol.ErrorCode_ERR_OK),
			Msg:  "ok",
		},
		Uid:     req.GetUid(),
		Welcome: "welcome",
	}
	return rsp, nil
}


