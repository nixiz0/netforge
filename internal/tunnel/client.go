package tunnel

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

// ClientAddr is the tunnel's local entrance: apps send their datagrams here.
const ClientAddr = "127.0.0.1:5300"

// RunClient is the near end of the tunnel. Every datagram a local app sends to ClientAddr is
// encapsulated and sent to the tunnel server; every packet coming back is decapsulated and
// delivered to that app. It only returns if a socket cannot be opened.
//
// Theory: _networking-lessons/5-tunneling.md#point-to-point-tunnel
func RunClient(server string) error {
	// Inner side: where local apps send their traffic. In step 6 a TUN interface takes this role.
	local, err := net.ListenPacket("udp", ClientAddr)
	if err != nil {
		return err
	}
	defer local.Close()

	// Outer side: one connected UDP socket to the tunnel server, across the real network.
	// Theory: _networking-lessons/5-tunneling.md#udp-tunneling
	outer, err := net.Dial("udp", server)
	if err != nil {
		return err
	}
	defer outer.Close()
	log.Printf("tunnel client: entrance %s, outer %s -> %s", local.LocalAddr(), outer.LocalAddr(), outer.RemoteAddr())

	// Tunnel state on this side: the local app that should receive the replies,
	// and whether the server is answering (see endpoint in state.go).
	var mu sync.Mutex
	var app net.Addr
	var remote endpoint

	// Keepalives: one tiny packet now, then every keepaliveInterval, even when idle. They keep NAT
	// and firewall state open along the path, and the server's answers prove the tunnel is up.
	keepalive := packet{typ: typeKeepalive}.encode()
	outer.Write(keepalive)
	go func() {
		t := time.NewTicker(keepaliveInterval)
		defer t.Stop()
		for range t.C {
			if _, err := outer.Write(keepalive); err != nil {
				log.Println("keepalive:", err)
			}
			if _, ok := remote.expire(peerTimeout); ok {
				log.Printf("tunnel down: no answer from the server for %s", peerTimeout)
			}
		}
	}()

	// Outer -> local, in the background: unwrap what the server sends back and deliver it to the app.
	go func() {
		buf := make([]byte, 65535)
		for {
			n, err := outer.Read(buf)
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if err != nil {
				log.Println("outer read:", err)
				continue
			}
			pkt, err := decode(buf[:n])
			if err != nil {
				log.Printf("drop %d bytes from the server: %v", n, err)
				continue
			}
			if remote.seen(outer.RemoteAddr()) {
				log.Printf("tunnel up: server %s is answering", outer.RemoteAddr())
			}
			if pkt.typ != typeData {
				continue // a keepalive answer: it already did its job by updating the state
			}
			mu.Lock()
			a := app
			mu.Unlock()
			if a == nil {
				continue
			}
			if _, err := local.WriteTo(pkt.payload, a); err != nil {
				log.Println("local write:", err)
				continue
			}
			log.Printf("outer %d bytes <- tunnel, decapsulated %d bytes -> %s", n, len(pkt.payload), a)
		}
	}()

	// Local -> outer, here: wrap each app datagram and send it through the tunnel.
	buf := make([]byte, 65535)
	for {
		n, from, err := local.ReadFrom(buf)
		if err != nil {
			log.Println("local read:", err)
			continue
		}
		mu.Lock()
		app = from
		mu.Unlock()

		// Too big to fit one frame once wrapped: drop it here rather than let the outer packet fragment.
		if n > tunnelMTU {
			log.Printf("drop %d-byte datagram from %s: above the tunnel MTU (%d), the outer packet would exceed %d bytes", n, from, tunnelMTU, linkMTU)
			continue
		}
		pkt := packet{typ: typeData, payload: buf[:n]}.encode()
		if _, err := outer.Write(pkt); err != nil {
			log.Println("outer write:", err)
			continue
		}
		log.Printf("local %d bytes <- %s, encapsulated %d bytes (%d on the wire) -> tunnel %s", n, from, len(pkt), len(pkt)+udpHeader+ipv4Header, outer.RemoteAddr())
	}
}
