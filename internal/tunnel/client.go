package tunnel

import (
	"errors"
	"log"
	"net"
	"sync"
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

	// Tunnel state on this side: the local app that should receive the replies.
	var mu sync.Mutex
	var app net.Addr

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
			if pkt.typ != typeData {
				continue
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

		pkt := packet{typ: typeData, payload: buf[:n]}.encode()
		if _, err := outer.Write(pkt); err != nil {
			log.Println("outer write:", err)
			continue
		}
		log.Printf("local %d bytes <- %s, encapsulated %d bytes -> tunnel %s", n, from, len(pkt), outer.RemoteAddr())
	}
}
