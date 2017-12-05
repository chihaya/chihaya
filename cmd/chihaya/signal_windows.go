// +build windows

package main

import (
	"os"
	"os/signal"
	"syscall"
)

func makeReloadChan() <-chan os.Signal {
	reload := make(chan os.Signal)
	signal.Notify(reload, syscall.SIGHUP)
	return reload
}
