//go:build darwin || freebsd || linux || netbsd || openbsd || dragonfly || solaris
// +build darwin freebsd linux netbsd openbsd dragonfly solaris

package main

import (
	"os"
	"syscall"
)

var ReloadSignals = []os.Signal{
	syscall.SIGUSR1,
}
