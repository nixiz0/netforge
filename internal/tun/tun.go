package tun

import (
	"log"
)

// Name is netforge's TUN interface.
const Name = "nf0"

// Run opens the TUN interface, logs every IP packet the kernel routes into it, and answers pings
// (ICMP echo requests) itself. Everything else stops here.
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

		// Write path: a packet written into the TUN interface is received by the kernel as if it
		// had arrived on a network card, then delivered to the app (here: ping).
		// Theory: _networking-lessons/6-tun-tap.md#vpn-receive-path
		if reply := echoReply(buf[:n], h); reply != nil {
			if _, err := dev.Write(reply); err != nil {
				log.Println("write reply:", err)
				continue
			}
			log.Printf("answered ping: wrote %d-byte echo reply %s -> %s", len(reply), h.dst, h.src)
		}
	}
}
