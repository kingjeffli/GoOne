package controller

import (
	"github.com/gin-gonic/gin"
)

// load web router
func LoadWebRoutes(router *gin.Engine) {
	//globals.PromMgr.SetGinMidAndRouter(router) // add mid and router

	// Phase 2 (new routes): IDL-driven HTTP handlers (gin) -> ssrpc runtime -> service implementation.
	d, srv := BuildWebDispatcher()
	d.MountGin(router)

	// Phase 1 (legacy routes): direct gin handlers -> legacy service functions.
	RegisterLegacyWebRoutes(router, srv)
}
