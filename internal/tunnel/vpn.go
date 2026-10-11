package tunnel

import (
	"errors"
	"io"
	"log"
	"net"
	"net/netip"
)

// RunTUN carries IP packets between a TUN interface and a peer. Every packet the kernel routes into
// dev is encapsulated and sent to peerAddr over UDP; every tunnel packet arriving on listen is
// decapsulated and written into dev, where the kernel receives it as if from a network card.
// Both ends run the same code: a symmetric point-to-point tunnel with fixed endpoints.
//
// dev is any io.ReadWriter, so this package never depends on how the TUN interface was opened.
//
// Theory: _networking-lessons/6-tun-tap.md#vpn-send-path
// Theory: _networking-lessons/5-tunneling.md#full-tunnel-flow
func RunTUN(dev io.ReadWriter, listen, peerAddr string) error {
	conn, err := net.ListenPacket("udp", listen)
	if err != nil {
		return err
	}
	defer conn.Close()
	peer, err := net.ResolveUDPAddr("udp", peerAddr)
	if err != nil {
		return err
	}
	log.Printf("tunnel over TUN: outer %s <-> peer %s", conn.LocalAddr(), peer)

	// Outer -> TUN, in the background: unwrap each tunnel packet and inject the IP packet inside.
	// Theory: _networking-lessons/6-tun-tap.md#vpn-receive-path
	go func() {
		buf := make([]byte, 65535)
		for {
			n, from, err := conn.ReadFrom(buf)
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if err != nil {
				log.Println("outer read:", err)
				continue
			}
			pkt, err := decode(buf[:n])
			if err != nil {
				log.Printf("drop %d bytes from %s: %v", n, from, err)
				continue
			}
			if pkt.typ != typeData {
				continue
			}
			if _, err := dev.Write(pkt.payload); err != nil {
				log.Println("TUN write:", err)
				continue
			}
			log.Printf("tunnel %d bytes <- %s, decapsulated %d-byte IP packet %s -> TUN", n, from, len(pkt.payload), innerAddrs(pkt.payload))
		}
	}()

	// TUN -> outer, here: wrap every IP packet the kernel routes into dev and send it to the peer.
	// The inner packet travels untouched: only the outer UDP/IP headers cross the real network.
	// Theory: _networking-lessons/5-tunneling.md#inner-vs-outer-packet
	buf := make([]byte, 65535)
	for {
		n, err := dev.Read(buf)
		if err != nil {
			return err
		}
		if n > tunnelMTU {
			log.Printf("drop %d-byte IP packet: above the tunnel MTU (%d), set the interface MTU to %d", n, tunnelMTU, tunnelMTU)
			continue
		}
		pkt := packet{typ: typeData, payload: buf[:n]}.encode()
		if _, err := conn.WriteTo(pkt, peer); err != nil {
			log.Println("outer write:", err)
			continue
		}
		log.Printf("TUN %d-byte IP packet %s, encapsulated %d bytes -> %s", n, innerAddrs(buf[:n]), len(pkt), peer)
	}
}

// innerAddrs reads the source and destination of an IPv4 packet for the logs ("10.10.0.1 -> 10.10.0.2").
// The tunnel itself never needs them: it carries the packet as opaque bytes.
func innerAddrs(ip []byte) string {
	if len(ip) < 20 || ip[0]>>4 != 4 {
		return "(not IPv4)"
	}
	return netip.AddrFrom4([4]byte(ip[12:16])).String() + " -> " + netip.AddrFrom4([4]byte(ip[16:20])).String()
}
