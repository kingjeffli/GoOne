//go:build !windows
// +build !windows

package application

import (
	"os"
	"os/signal"
	"syscall"
)

func SignalNotify() {
	signal.Notify(sig, syscall.SIGABRT, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
}

func isReloadSignal(s os.Signal) bool {
	return s == syscall.SIGUSR1
}
