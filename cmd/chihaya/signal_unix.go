//go:build darwin || freebsd || linux || netbsd || openbsd || dragonfly || solaris
// +build darwin freebsd linux netbsd openbsd dragonfly solaris

package main

import (
	"os"
	"syscall"
)

// ReloadSignals are the signals that the current OS will send to the process
// when a configuration reload is requested.
var ReloadSignals = []os.Signal{
	syscall.SIGUSR1,
}
