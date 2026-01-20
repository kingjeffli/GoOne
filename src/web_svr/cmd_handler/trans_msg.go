package cmd_handler

import (
	"github.com/Iori372552686/GoOne/src/web_svr/service"
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/encoding/protojson"
	websvr "github.com/Iori372552686/GoOne/api/gen/web/websvr/v1"
)

func MsgSecCheck(ctx *gin.Context, data []byte) gin.H {
	// Compatibility layer: legacy /safe/msg/msgSecCheck now calls the IDL-driven implementation.
	pbReq := &websvr.MsgSecCheckReq{}
	if len(data) != 0 {
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, pbReq); err != nil {
			return gin.H{"code": g1_protocol.ErrorCode_ERR_MARSHAL, "data": nil, "msg": g1_protocol.ErrorCode_ERR_MARSHAL.String()}
		}
	}

	impl := &service.WebApiServiceImpl{}
	pbRsp, err := impl.MsgSecCheck(nil, pbReq)
	if err != nil {
		code := ssrpc.ToErrorCode(err)
		// Keep legacy shape: {code,data,msg}. data was historically nil for this endpoint.
		return gin.H{"code": code, "data": nil, "msg": err.Error()}
	}
	return gin.H{"code": g1_protocol.ErrorCode_ERR_OK, "data": pbRsp, "msg": g1_protocol.ErrorCode_ERR_OK.String()}
}
