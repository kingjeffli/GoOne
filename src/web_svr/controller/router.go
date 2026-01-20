package controller

import (
	websvrv1 "github.com/Iori372552686/GoOne/api/gen/web/websvr/v1"
	"github.com/Iori372552686/GoOne/lib/service/ssrpc"
	"github.com/Iori372552686/GoOne/src/web_svr/service"
	"github.com/gin-gonic/gin"
)

// load web router
func LoadWebRoutes(router *gin.Engine) {
	//globals.PromMgr.SetGinMidAndRouter(router) // add mid and router

	new(WebApiController).Init(router)

	// Phase 2 (new routes): IDL-driven HTTP handlers (gin) -> ssrpc runtime -> service implementation.
	srv := websvrv1.NewWebApiServiceSServer(&service.WebApiServiceImpl{}, ssrpc.DefaultMWOptions{})
	d := ssrpc.NewDispatcher()
	websvrv1.RegisterWebApiServiceToDispatcher(d, srv)
	d.MountGin(router)
}
