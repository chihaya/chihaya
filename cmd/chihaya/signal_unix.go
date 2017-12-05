// +build darwin freebsd linux netbsd openbsd dragonfly solaris

package main

import (
	"os"
	"os/signal"
	"syscall"
)

func makeReloadChan() <-chan os.Signal {
	reload := make(chan os.Signal)
	signal.Notify(reload, syscall.SIGUSR1)
	return reload
}
