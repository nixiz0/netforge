package tunnel

import (
	"errors"
	"log"
	"net"
	"time"
)

// ServerAddr is the tunnel's outer endpoint: where encapsulated packets arrive.
const ServerAddr = "127.0.0.1:7000"

// RunServer is the far end of the tunnel. It decapsulates every packet arriving on ServerAddr,
// forwards the inner datagram to target, and sends target's replies back encapsulated.
// It only returns if a socket cannot be opened.
//
// Theory: _networking-lessons/5-tunneling.md#tunnel-endpoints
func RunServer(target string) error {
	// Outer side: the socket the tunnel client talks to, across the real network.
	outer, err := net.ListenPacket("udp", ServerAddr)
	if err != nil {
		return err
	}
	defer outer.Close()

	// Inner side: one UDP socket toward the target. The target only ever sees the tunnel server,
	// never the original client: inner and outer addresses are separate worlds.
	// Theory: _networking-lessons/5-tunneling.md#inner-vs-outer-packet
	inner, err := net.Dial("udp", target)
	if err != nil {
		return err
	}
	defer inner.Close()
	log.Printf("tunnel server: outer %s, inner %s -> %s", outer.LocalAddr(), inner.LocalAddr(), inner.RemoteAddr())

	// Tunnel state: the client's outer address and when it was last heard (see peer in state.go).
	var client peer

	// Watchdog: a client silent for peerTimeout (no data, no keepalive) is forgotten.
	go func() {
		t := time.NewTicker(keepaliveInterval)
		defer t.Stop()
		for range t.C {
			if gone, ok := client.expire(peerTimeout); ok {
				log.Printf("peer %s silent for %s: forgotten", gone, peerTimeout)
			}
		}
	}()

	// Inner -> outer, in the background: wrap each reply from the target and send it to the peer.
	go func() {
		buf := make([]byte, 65535)
		for {
			n, err := inner.Read(buf)
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if err != nil {
				log.Println("inner read:", err)
				continue
			}
			p := client.get()
			if p == nil {
				continue // no live peer to send it to
			}
			// Same rule on the way back: a reply above the tunnel MTU would not fit one frame once wrapped.
			if n > tunnelMTU {
				log.Printf("drop %d-byte reply from %s: above the tunnel MTU (%d)", n, target, tunnelMTU)
				continue
			}
			pkt := packet{typ: typeData, payload: buf[:n]}.encode()
			if _, err := outer.WriteTo(pkt, p); err != nil {
				log.Println("outer write:", err)
				continue
			}
			log.Printf("inner %d bytes <- %s, encapsulated %d bytes (%d on the wire) -> %s", n, target, len(pkt), len(pkt)+udpHeader+ipv4Header, p)
		}
	}()

	// Outer -> inner, here: unwrap each tunnel packet and forward its payload to the target.
	buf := make([]byte, 65535)
	for {
		n, from, err := outer.ReadFrom(buf)
		if err != nil {
			log.Println("outer read:", err)
			continue
		}
		pkt, err := decode(buf[:n])
		if err != nil {
			log.Printf("drop %d bytes from %s: %v", n, from, err)
			continue
		}
		if client.seen(from) {
			log.Printf("peer %s joined", from)
		}

		switch pkt.typ {
		case typeData:
			log.Printf("outer %d bytes <- %s, decapsulated %d bytes -> %s", n, from, len(pkt.payload), target)
			if _, err := inner.Write(pkt.payload); err != nil {
				log.Println("inner write:", err)
			}
		case typeKeepalive:
			// Answer, so the client also knows the server is alive.
			if _, err := outer.WriteTo(packet{typ: typeKeepalive}.encode(), from); err != nil {
				log.Println("keepalive reply:", err)
			}
			log.Printf("keepalive <- %s, answered", from)
		default:
			log.Printf("unknown tunnel packet type %#x from %s", pkt.typ, from)
		}
	}
}
