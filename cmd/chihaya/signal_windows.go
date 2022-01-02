//go:build windows
// +build windows

package main

import (
	"os"
	"os/signal"
	"syscall"
)

var ReloadSignals = []os.Signal{
	syscall.SIGHUP,
}
