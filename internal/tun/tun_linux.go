package tun

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// Open creates the TUN interface name (or attaches to it if it exists) and returns it as a file:
// each Read returns one whole IP packet the kernel routed into the interface, each Write injects one.
// It needs root (or CAP_NET_ADMIN): run with sudo.
//
// Theory: _networking-lessons/6-tun-tap.md#tun-interface
// Theory: _networking-lessons/6-tun-tap.md#permissions
func Open(name string) (*os.File, error) {
	fd, err := unix.Open("/dev/net/tun", unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open /dev/net/tun: %w", err)
	}

	ifr, err := unix.NewIfreq(name)
	if err != nil {
		unix.Close(fd)
		return nil, err
	}
	// IFF_TUN: layer 3, plain IP packets (TAP would be layer 2, Ethernet frames).
	// IFF_NO_PI: no extra 4-byte "packet information" prefix, so byte 0 is the IP header.
	// Theory: _networking-lessons/6-tun-tap.md#tun-vs-tap
	ifr.SetUint16(unix.IFF_TUN | unix.IFF_NO_PI)
	if err := unix.IoctlIfreq(fd, unix.TUNSETIFF, ifr); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("create TUN interface %s: %w (run with sudo)", name, err)
	}
	return os.NewFile(uintptr(fd), "/dev/net/tun"), nil
}
