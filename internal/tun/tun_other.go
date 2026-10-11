//go:build !linux

package tun

import (
	"errors"
	"os"
)

// Open is only implemented on Linux, where TUN is built into the kernel (/dev/net/tun).
// This file keeps the rest of netforge building on Windows and macOS.
func Open(name string) (*os.File, error) {
	return nil, errors.New("TUN interfaces are only supported on Linux: run netforge tun inside WSL2")
}
