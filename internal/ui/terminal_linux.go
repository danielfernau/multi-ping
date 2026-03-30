//go:build linux

package ui

import (
	"os"
	"syscall"
	"unsafe"
)

type winsize struct {
	rows uint16
	cols uint16
	x    uint16
	y    uint16
}

func terminalSize() (width, height int) {
	ws := &winsize{}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		os.Stdout.Fd(),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)),
	)
	if errno == 0 && ws.cols > 0 && ws.rows > 0 {
		return int(ws.cols), int(ws.rows)
	}
	return 120, 40
}
