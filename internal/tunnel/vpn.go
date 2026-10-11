package tunnel

import (
	"crypto/ecdh"
	"errors"
	"io"
	"log"
	"net"
	"net/netip"
	"strings"
	"time"
)

// VPNPort is the UDP port the VPN server listens on.
const VPNPort = "7000"

// RunTUN carries IP packets between a TUN interface and the other ends of the VPN. Every packet the
// kernel routes into dev is encrypted and sent over UDP; every tunnel packet received is checked,
// decrypted and written into dev, where the kernel receives it as if from a network card.
//
// Client mode (serverAddr set, one peer: the server): the client starts a handshake with that
// fixed server, then sends a keepalive every 5 s that keeps NAT state open and tells the server
// where the client is. If the server stops answering, the client starts a new handshake.
// Server mode (serverAddr == ""): one row per peer in a peer table. The server answers each peer's
// handshakes, learns each peer's endpoint from its authentic packets (NAT, roaming), and sends
// every inner packet to the peer that owns its destination address. Hub and spoke: two clients
// talk to each other through the server.
//
// priv is this end's private key (keys.go). The handshake (handshake.go) turns it and each peer's
// public key into fresh session keys; every packet is sealed with them (crypto.go), and anything
// that fails the check is dropped before it can change anything.
//
// dev is any io.ReadWriter, so this package never depends on how the TUN interface was opened.
//
// Theory: _networking-lessons/6-tun-tap.md#vpn-send-path
// Theory: _networking-lessons/8-security-cryptography.md#full-vpn-security-model
// Theory: _networking-lessons/9-multi-peer-networking.md#hub-and-spoke
func RunTUN(dev io.ReadWriter, listen, serverAddr string, priv *ecdh.PrivateKey, peers []Peer) error {
	isClient := serverAddr != ""
	if isClient && len(peers) != 1 {
		return errors.New("a VPN client has exactly one peer: the server")
	}
	table, err := newPeerTable(priv, peers, isClient)
	if err != nil {
		return err
	}
	conn, err := net.ListenPacket("udp", listen)
	if err != nil {
		return err
	}
	defer conn.Close()

	var server *net.UDPAddr // client mode only: where to send
	if isClient {
		if server, err = net.ResolveUDPAddr("udp", serverAddr); err != nil {
			return err
		}
		srv := table.all[0]
		log.Printf("VPN client: outer %s -> server %s", conn.LocalAddr(), server)
		// Handshake now. Then every 5 s: a keepalive while the session is up, a new handshake if not.
		// Theory: _networking-lessons/5-tunneling.md#keepalives
		conn.WriteTo(srv.hs.start(), server)
		go func() {
			t := time.NewTicker(keepaliveInterval)
			defer t.Stop()
			for range t.C {
				if _, ok := srv.ep.expire(peerTimeout); ok {
					srv.session.Store(nil)
					log.Printf("tunnel down: no answer from the server for %s, new handshake", peerTimeout)
				}
				if s := srv.session.Load(); s != nil {
					conn.WriteTo(s.seal(typeKeepalive, nil), server)
				} else {
					conn.WriteTo(srv.hs.start(), server)
				}
			}
		}()
	} else {
		var addrs []string
		for _, p := range peers {
			addrs = append(addrs, p.Addr.String())
		}
		log.Printf("VPN server: listening on %s, peers %s", conn.LocalAddr(), strings.Join(addrs, ", "))
		// Watchdog: a peer silent for 15 s (no data, no keepalive) is forgotten, with its keys.
		go func() {
			t := time.NewTicker(keepaliveInterval)
			defer t.Stop()
			for range t.C {
				for _, p := range table.all {
					if gone, ok := p.ep.expire(peerTimeout); ok {
						p.session.Store(nil)
						log.Printf("peer %s at %s silent for %s: forgotten", p.Addr, gone, peerTimeout)
					}
				}
			}
		}()
	}

	// Outer -> TUN, in the background: find whose keys open the packet, then act on it.
	// Theory: _networking-lessons/9-multi-peer-networking.md#receive-flow
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
			p, keys, pkt, err := table.identify(buf[:n], from)
			if err != nil {
				log.Printf("drop %d bytes from %s: %v", n, from, err)
				continue
			}

			switch pkt.typ {
			case typeInit:
				if isClient {
					continue
				}
				resp, s, err := p.hs.respond(pkt)
				if err != nil {
					log.Printf("drop handshake from %s: %v", from, err)
					continue
				}
				p.session.Store(s)
				conn.WriteTo(resp, from)
				log.Printf("handshake from %s (peer %s): new session keys", from, p.Addr)
			case typeResp:
				if !isClient {
					continue
				}
				s, err := p.hs.finish(pkt)
				if err != nil {
					log.Printf("drop handshake from %s: %v", from, err)
					continue
				}
				p.session.Store(s)
				log.Printf("handshake done: new session keys")
			case typeKeepalive:
				if !isClient {
					conn.WriteTo(keys.seal(typeKeepalive, nil), from) // the server answers, so the client knows it is up
				}
			case typeData:
				if _, err := dev.Write(pkt.payload); err != nil {
					log.Println("TUN write:", err)
					continue
				}
				log.Printf("tunnel %d bytes <- %s, decrypted %d-byte IP packet %s -> TUN", n, from, len(pkt.payload), innerAddrs(pkt.payload))
			}

			// Only now, after the packet passed every check, may it update this peer's endpoint:
			// first contact, back after a timeout, or roaming.
			// Theory: _networking-lessons/7-vpn-architecture.md#endpoint-changes--roaming
			if p.ep.seen(from) {
				if isClient {
					log.Printf("tunnel up: server %s is answering", from)
				} else {
					log.Printf("peer %s joined from %s", p.Addr, from)
				}
			}
		}
	}()

	// TUN -> outer, here: pick the peer (the server, or the owner of the destination address),
	// encrypt the IP packet with that peer's session keys, send it to that peer's endpoint.
	// Theory: _networking-lessons/9-multi-peer-networking.md#send-flow
	buf := make([]byte, 65535)
	for {
		n, err := dev.Read(buf)
		if err != nil {
			return err
		}
		p := table.all[0]
		var dest net.Addr = server
		if !isClient {
			if p = table.route(buf[:n]); p == nil {
				log.Printf("drop %d-byte IP packet %s: no peer owns this destination", n, innerAddrs(buf[:n]))
				continue
			}
			dest = p.ep.get()
		}
		s := p.session.Load()
		if dest == nil || s == nil {
			log.Printf("drop %d-byte IP packet %s: no session yet", n, innerAddrs(buf[:n]))
			continue
		}
		if n > vpnMTU {
			log.Printf("drop %d-byte IP packet: above the VPN MTU (%d), set the interface MTU to %d", n, vpnMTU, vpnMTU)
			continue
		}
		pkt := s.seal(typeData, buf[:n])
		if _, err := conn.WriteTo(pkt, dest); err != nil {
			log.Println("outer write:", err)
			continue
		}
		log.Printf("TUN %d-byte IP packet %s, encrypted %d bytes -> %s", n, innerAddrs(buf[:n]), len(pkt), dest)
	}
}

// innerAddrs reads the source and destination of an IPv4 packet for the logs ("10.10.0.1 -> 10.10.0.2").
func innerAddrs(ip []byte) string {
	if len(ip) < 20 || ip[0]>>4 != 4 {
		return "(not IPv4)"
	}
	return netip.AddrFrom4([4]byte(ip[12:16])).String() + " -> " + netip.AddrFrom4([4]byte(ip[16:20])).String()
}
