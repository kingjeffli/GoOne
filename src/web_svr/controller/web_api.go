package controller

import (
	websvrv1 "github.com/Iori372552686/GoOne/api/gen/web/websvr/v1"
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	define "github.com/Iori372552686/GoOne/src/web_svr/common"
	"github.com/gin-gonic/gin"
)

// RegisterLegacyWebRoutes keeps the old safe_msg entrypoint alive while routing
// it through the same ssrpc HTTP runtime and middleware chain as the new IDL route.
func RegisterLegacyWebRoutes(router *gin.Engine, srv websvrv1.WebApiServiceSServer) {
	if router == nil || srv.Impl == nil {
		return
	}

	msgSecCheck := ssrpc.WrapHTTPGin(
		ssrpc.MethodDesc{
			Cmd:  0,
			Sign: true,
			Name: "msg security check",
		},
		srv.MW,
		func() any { return new(websvrv1.MsgSecCheckReq) },
		func(ctx *ssrpc.Context, in any) (any, error) {
			return srv.Impl.MsgSecCheck(ctx, in.(*websvrv1.MsgSecCheckReq))
		},
	)

	path := define.RestApi_SafeMsg_Dir + "msgSecCheck"
	router.Handle("POST", path, msgSecCheck)
	router.Handle("GET", path, msgSecCheck)
}
