//go:build windows
// +build windows

package main

import (
	"os"
	"syscall"
)

var ReloadSignals = []os.Signal{
	syscall.SIGHUP,
}
