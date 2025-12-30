package web_gin

import (
	"errors"
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/api/logger/zap"
	"github.com/Iori372552686/GoOne/lib/web/rest"
	"time"

	"github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"strconv"
)

/**
* @Description: Run gin start the server
* @param: http_port
* @param: mode
* @param: session_name
* @param: load_routers
* @return: error
* @Author: Iori
* @Date: 2022-02-28 11:27:27
**/
func RunGin(conf Config, load_routers func(router *gin.Engine)) error {
	if conf.Port <= 0 {
		return errors.New("port args err!")
	}

	//mode
	switch conf.Mode {
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(rest.Cors())
	router.Use(
		ginzap.Ginzap(zap.ZapLogger, time.RFC3339, true), // 使用 Zap 替换默认日志中间件
		ginzap.RecoveryWithZap(zap.ZapLogger, true),      // 替换 gin.Recovery()
	)

	//loadRoutes
	load_routers(router)
	go router.Run(conf.IP + ":" + strconv.Itoa(conf.Port))
	logger.Infof("------ Http Gin Server Running by %v:%v ------", conf.IP, conf.Port)
	return nil
}
