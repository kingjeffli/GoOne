package controller

import (
	"errors"
	"fmt"
	websvrv1 "github.com/Iori372552686/GoOne/api/gen/web/websvr/v1"
	"github.com/Iori372552686/GoOne/common/gconf"
	"github.com/Iori372552686/GoOne/lib/api/http_sign"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	"github.com/Iori372552686/GoOne/lib/web/rest"
	define "github.com/Iori372552686/GoOne/src/web_svr/common"
	"github.com/Iori372552686/GoOne/src/web_svr/globals"
	"github.com/Iori372552686/GoOne/src/web_svr/service"
	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
	"net/http"
)

/*
*  WebApiController
*  @Description:
 */
type WebApiController struct {
	rest.Controller
}

/**
* @Description:  AOP pre
* @receiver: self
* @return: gin.HandlerFunc
* @Author: Iori
* @Date: 2022-01-28 11:40:41
**/
func (self *WebApiController) before() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bodyBytes, _ := io.ReadAll(ctx.Request.Body)
		ctx.Set("body", bodyBytes)

		if !self.Auth {
			ctx.Next()
			return
		}

		ok, err, _ := globals.SignMgr.GetSignIns().CheckSign(http_sign.UriParam2Map(ctx.Request.URL.RawQuery), bodyBytes, "")
		if ok {
			ctx.Next()
			return
		}

		logger.Errorf("CheckSign fail, url_args | %v | err: %v", ctx.Request.RequestURI, err.Error())
		rest.ResultFail(ctx, fmt.Sprintf("Invalid signature ! err | %v", err.Error()))
		ctx.Abort()
	}
}

/**
* @Description: init
* @Author: Iori
* @Date: 2022-07-26 17:50:15
**/
func (self *WebApiController) Init(router *gin.Engine) error {
	if router == nil {
		return errors.New("gin router is nil!")
	}

	self.Auth = gconf.WebSvrCfg.HttpServer.AuthEnable
	self.router(router)
	return nil
}

/**
* @Description: add Router
* @receiver: self
* @param: router
* @Author: Iori
* @Date: 2022-01-28 11:40:28
**/
func (self *WebApiController) router(router *gin.Engine) {
	rg := router.Group(define.RestApi_SafeMsg_Dir)
	rg.Use(self.before())

	// Keep a single explicit compatibility route while the old safe_msg endpoint remains in use.
	rg.POST("/msgSecCheck", self.legacyMsgSecCheck)
	rg.GET("/msgSecCheck", self.legacyMsgSecCheck)
}

func (self *WebApiController) legacyMsgSecCheck(ctx *gin.Context) {
	var bodyBytes []byte
	if body, ok := ctx.Get("body"); ok {
		if data, ok := body.([]byte); ok {
			bodyBytes = data
		}
	}

	pbReq := &websvrv1.MsgSecCheckReq{}
	if len(bodyBytes) != 0 {
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(bodyBytes, pbReq); err != nil {
			ctx.JSON(http.StatusOK, gin.H{
				"code": g1_protocol.ErrorCode_ERR_MARSHAL,
				"data": nil,
				"msg":  g1_protocol.ErrorCode_ERR_MARSHAL.String(),
			})
			return
		}
	}

	impl := &service.WebApiServiceImpl{}
	pbRsp, err := impl.MsgSecCheck(nil, pbReq)
	if err != nil {
		code := ssrpc.ToErrorCode(err)
		ctx.JSON(http.StatusOK, gin.H{
			"code": code,
			"data": nil,
			"msg":  err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": g1_protocol.ErrorCode_ERR_OK,
		"data": pbRsp,
		"msg":  g1_protocol.ErrorCode_ERR_OK.String(),
	})
}
