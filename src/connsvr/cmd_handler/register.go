package cmd_handler

import (
	"github.com/Iori372552686/GoOne/lib/api/logger"
)

// 所有的命令字对应的go需要在这里先注册
func RegCmd() {
	logger.Infof("register transaction commands")
	// ConnService is registered via generated ssrpc wrappers in src/connsvr/app.go.
}
