package tun

import (
	"log"
)

// Name is netforge's TUN interface.
const Name = "nf0"

// Run opens the TUN interface and logs every IP packet the kernel routes into it.
// Nothing is answered yet: the packets stop here.
//
// Theory: _networking-lessons/6-tun-tap.md#virtual-network-interfaces
func Run() error {
	dev, err := Open(Name)
	if err != nil {
		return err
	}
	defer dev.Close()
	log.Printf("TUN interface %s open. In another terminal: sudo ip addr add 10.10.0.1/24 dev %s && sudo ip link set %s up mtu 1468", Name, Name, Name)

	buf := make([]byte, 65535)
	for {
		// One Read = one whole IP packet, boundaries kept, like a UDP datagram.
		// The kernel only sends packets here when a route points at the interface.
		// Theory: _networking-lessons/6-tun-tap.md#routing-controls-tun-traffic
		n, err := dev.Read(buf)
		if err != nil {
			return err
		}
		// The first 4 bits of every IP packet are its version: 4 or 6.
		if buf[0]>>4 != 4 {
			log.Printf("IPv%d packet, %d bytes (not decoded)", buf[0]>>4, n)
			continue
		}
		h, err := parseIPv4(buf[:n])
		if err != nil {
			log.Printf("drop %d bytes: %v", n, err)
			continue
		}
		log.Printf("IPv4 %s", h)
	}
}
