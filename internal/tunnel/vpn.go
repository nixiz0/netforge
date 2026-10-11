package tunnel

import (
	"errors"
	"io"
	"log"
	"net"
	"net/netip"
	"time"
)

// VPNPort is the UDP port the VPN server listens on.
const VPNPort = "7000"

// RunTUN carries IP packets between a TUN interface and the other end of the VPN. Every packet the
// kernel routes into dev is encapsulated and sent over UDP; every tunnel packet received is
// decapsulated and written into dev, where the kernel receives it as if from a network card.
//
// Client mode (serverAddr set): packets go to that fixed server, plus a keepalive every 5 s that
// keeps NAT state open and tells the server where the client currently is.
// Server mode (serverAddr == ""): the client's address is learned from its packets, so the client
// can sit behind NAT and change address (roaming) without any configuration.
//
// dev is any io.ReadWriter, so this package never depends on how the TUN interface was opened.
//
// Theory: _networking-lessons/6-tun-tap.md#vpn-send-path
// Theory: _networking-lessons/7-vpn-architecture.md#vpn-endpoint-vs-vpn-address
func RunTUN(dev io.ReadWriter, listen, serverAddr string) error {
	conn, err := net.ListenPacket("udp", listen)
	if err != nil {
		return err
	}
	defer conn.Close()

	var server *net.UDPAddr // client mode only: where to send
	if serverAddr != "" {
		if server, err = net.ResolveUDPAddr("udp", serverAddr); err != nil {
			return err
		}
	}
	// Tunnel state: who is on the other side and when we last heard from it.
	var remote peer
	keepalive := packet{typ: typeKeepalive}.encode()

	if server != nil {
		log.Printf("VPN client: outer %s -> server %s", conn.LocalAddr(), server)
		// Keepalives: one now, then every 5 s. A server silent for 15 s means the tunnel is down.
		// Theory: _networking-lessons/5-tunneling.md#keepalives
		conn.WriteTo(keepalive, server)
		go func() {
			t := time.NewTicker(keepaliveInterval)
			defer t.Stop()
			for range t.C {
				conn.WriteTo(keepalive, server)
				if _, ok := remote.expire(peerTimeout); ok {
					log.Printf("tunnel down: no answer from the server for %s", peerTimeout)
				}
			}
		}()
	} else {
		log.Printf("VPN server: listening on %s, waiting for a client", conn.LocalAddr())
		// Watchdog: a client silent for 15 s (no data, no keepalive) is forgotten.
		go func() {
			t := time.NewTicker(keepaliveInterval)
			defer t.Stop()
			for range t.C {
				if gone, ok := remote.expire(peerTimeout); ok {
					log.Printf("client %s silent for %s: forgotten", gone, peerTimeout)
				}
			}
		}()
	}

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
			// A new address for the other side: first contact, back after a timeout, or roaming.
			// Theory: _networking-lessons/7-vpn-architecture.md#endpoint-changes--roaming
			if remote.seen(from) {
				if server != nil {
					log.Printf("tunnel up: server %s is answering", from)
				} else {
					log.Printf("client %s joined", from)
				}
			}
			switch pkt.typ {
			case typeKeepalive:
				if server == nil {
					conn.WriteTo(keepalive, from) // the server answers, so the client knows it is up
				}
			case typeData:
				if _, err := dev.Write(pkt.payload); err != nil {
					log.Println("TUN write:", err)
					continue
				}
				log.Printf("tunnel %d bytes <- %s, decapsulated %d-byte IP packet %s -> TUN", n, from, len(pkt.payload), innerAddrs(pkt.payload))
			}
		}
	}()

	// TUN -> outer, here: wrap every IP packet the kernel routes into dev and send it to the other
	// side: the fixed server (client mode) or the last address the client was seen at (server mode).
	// The inner packet travels untouched: only the outer UDP/IP headers cross the real network.
	// Theory: _networking-lessons/5-tunneling.md#inner-vs-outer-packet
	buf := make([]byte, 65535)
	for {
		n, err := dev.Read(buf)
		if err != nil {
			return err
		}
		var dest net.Addr = server
		if server == nil {
			dest = remote.get()
		}
		if dest == nil {
			log.Printf("drop %d-byte IP packet %s: no client connected", n, innerAddrs(buf[:n]))
			continue
		}
		if n > tunnelMTU {
			log.Printf("drop %d-byte IP packet: above the tunnel MTU (%d), set the interface MTU to %d", n, tunnelMTU, tunnelMTU)
			continue
		}
		pkt := packet{typ: typeData, payload: buf[:n]}.encode()
		if _, err := conn.WriteTo(pkt, dest); err != nil {
			log.Println("outer write:", err)
			continue
		}
		log.Printf("TUN %d-byte IP packet %s, encapsulated %d bytes -> %s", n, innerAddrs(buf[:n]), len(pkt), dest)
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
