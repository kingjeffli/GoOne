//go:build windows
// +build windows

package application

import (
	"os"
	"os/signal"
	"syscall"
)

func SignalNotify() {
	signal.Notify(sig, syscall.SIGABRT, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
}

func isReloadSignal(_ os.Signal) bool {
	return false
}
