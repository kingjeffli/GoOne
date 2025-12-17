package dispatch

import (
	"github.com/Iori372552686/GoOne/lib/api/http_sign"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/web/rest"
	"github.com/Iori372552686/GoOne/src/web_svr/cmd_handler"
	define "github.com/Iori372552686/GoOne/src/web_svr/common"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

/**
* @Description: dispach handler
* @param: ctx
* @Author: Iori
* @Date: 2022-04-26 18:01:39
**/
func CmdHandlerProcess(ctx *gin.Context) {
	var bodyBytes []byte
	body, ok := ctx.Get("body")
	uri := ctx.Request.RequestURI
	headerParams := make(map[string]interface{})

	//pre body
	if ok && body.([]byte) != nil {
		bodyBytes = body.([]byte)
		logger.Debugf("Request.Body:  %s", body)
	}

	//copy head
	//logger.Debugf("Request.Heade [%v]", ctx.Request.Header)
	for k, v := range ctx.Request.Header {
		if len(v) == 0 {
			continue
		}
		headerParams[k] = v[0]
	}

	//set ctx params
	ctx.Set("uri_map", http_sign.UriParam2Map(ctx.Request.URL.RawQuery))
	ctx.Set("header", headerParams)

	//dispatch request
	if index := strings.Index(uri, "?"); index > 0 {
		uri = uri[:index]
	}
	cmd := strings.ReplaceAll(uri, define.RestApi_SafeMsg_Dir, "")
	handler, ok := cmd_handler.CmdHandler.ChMgr.GetHttpHandlers(cmd)
	if !ok {
		logger.Errorf("web no reg handler!  url | %v", uri)
		rest.Result(ctx, int32(g1_protocol.ErrorCode_ERR_FUNCTION_NOT_OPEN), nil, g1_protocol.ErrorCode_ERR_FUNCTION_NOT_OPEN.String())
		return
	}

	ctx.JSON(http.StatusOK, handler(ctx, bodyBytes))
}
