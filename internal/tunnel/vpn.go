package tunnel

import (
	"crypto/ecdh"
	"errors"
	"io"
	"log"
	"net"
	"net/netip"
	"sync/atomic"
	"time"
)

// VPNPort is the UDP port the VPN server listens on.
const VPNPort = "7000"

// RunTUN carries IP packets between a TUN interface and the other end of the VPN. Every packet the
// kernel routes into dev is encrypted and sent over UDP; every tunnel packet received is checked,
// decrypted and written into dev, where the kernel receives it as if from a network card.
//
// Client mode (serverAddr set): the client starts a handshake with that fixed server, then sends a
// keepalive every 5 s that keeps NAT state open and tells the server where the client is. If the
// server stops answering, the client starts a new handshake.
// Server mode (serverAddr == ""): the server answers handshakes, and learns the client's address
// from its authentic packets, so the client can sit behind NAT and roam.
//
// priv is this end's private key, peerKey the other end's public key (keys.go). The handshake
// (handshake.go) turns them into fresh session keys; every packet is sealed with them (crypto.go),
// and anything that fails the check is dropped before it can change anything.
//
// dev is any io.ReadWriter, so this package never depends on how the TUN interface was opened.
//
// Theory: _networking-lessons/6-tun-tap.md#vpn-send-path
// Theory: _networking-lessons/7-vpn-architecture.md#vpn-endpoint-vs-vpn-address
// Theory: _networking-lessons/8-security-cryptography.md#full-vpn-security-model
func RunTUN(dev io.ReadWriter, listen, serverAddr string, priv *ecdh.PrivateKey, peerKey *ecdh.PublicKey) error {
	isClient := serverAddr != ""
	hs, err := newHandshake(priv, peerKey, isClient)
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
	}
	// Tunnel state: who is on the other side and when we last heard from it.
	var remote peer
	// The current session keys: nil until a handshake completes. The receive goroutine swaps them,
	// the send loop and the keepalive goroutine read them, so the pointer is atomic.
	var session atomic.Pointer[sealer]

	if isClient {
		log.Printf("VPN client: outer %s -> server %s", conn.LocalAddr(), server)
		// Handshake now. Then every 5 s: a keepalive while the session is up, a new handshake if not.
		// Theory: _networking-lessons/5-tunneling.md#keepalives
		conn.WriteTo(hs.start(), server)
		go func() {
			t := time.NewTicker(keepaliveInterval)
			defer t.Stop()
			for range t.C {
				if _, ok := remote.expire(peerTimeout); ok {
					session.Store(nil)
					log.Printf("tunnel down: no answer from the server for %s, new handshake", peerTimeout)
				}
				if s := session.Load(); s != nil {
					conn.WriteTo(s.seal(typeKeepalive, nil), server)
				} else {
					conn.WriteTo(hs.start(), server)
				}
			}
		}()
	} else {
		log.Printf("VPN server: listening on %s, waiting for a handshake", conn.LocalAddr())
		// Watchdog: a client silent for 15 s (no data, no keepalive) is forgotten, with its keys.
		go func() {
			t := time.NewTicker(keepaliveInterval)
			defer t.Stop()
			for range t.C {
				if gone, ok := remote.expire(peerTimeout); ok {
					session.Store(nil)
					log.Printf("client %s silent for %s: forgotten", gone, peerTimeout)
				}
			}
		}()
	}

	// Outer -> TUN, in the background: check and decrypt each tunnel packet, then act on it.
	// Theory: _networking-lessons/6-tun-tap.md#vpn-receive-path
	// Theory: _networking-lessons/8-security-cryptography.md#vpn-packet-security
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
			// Handshake messages are sealed with the static keys, everything else with the session keys.
			keys := session.Load()
			if n > 1 && (buf[1] == typeInit || buf[1] == typeResp) {
				keys = hs.static
			}
			if keys == nil {
				log.Printf("drop %d bytes from %s: no session yet", n, from)
				continue
			}
			pkt, err := keys.open(buf[:n])
			if err != nil {
				log.Printf("drop %d bytes from %s: %v", n, from, err)
				continue
			}

			switch pkt.typ {
			case typeInit:
				if isClient {
					continue
				}
				resp, s, err := hs.respond(pkt)
				if err != nil {
					log.Printf("drop handshake from %s: %v", from, err)
					continue
				}
				session.Store(s)
				conn.WriteTo(resp, from)
				log.Printf("handshake from %s: new session keys", from)
			case typeResp:
				if !isClient {
					continue
				}
				s, err := hs.finish(pkt)
				if err != nil {
					log.Printf("drop handshake from %s: %v", from, err)
					continue
				}
				session.Store(s)
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

			// Only now, after the packet passed every check, may it update the other side's address:
			// first contact, back after a timeout, or roaming.
			// Theory: _networking-lessons/7-vpn-architecture.md#endpoint-changes--roaming
			if remote.seen(from) {
				if isClient {
					log.Printf("tunnel up: server %s is answering", from)
				} else {
					log.Printf("client %s joined", from)
				}
			}
		}
	}()

	// TUN -> outer, here: encrypt every IP packet the kernel routes into dev and send it to the
	// other side: the fixed server (client mode) or the last address the client was seen at.
	// Theory: _networking-lessons/5-tunneling.md#inner-vs-outer-packet
	buf := make([]byte, 65535)
	for {
		n, err := dev.Read(buf)
		if err != nil {
			return err
		}
		var dest net.Addr = server
		if !isClient {
			dest = remote.get()
		}
		s := session.Load()
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
// The tunnel itself never needs them: it carries the packet as opaque bytes.
func innerAddrs(ip []byte) string {
	if len(ip) < 20 || ip[0]>>4 != 4 {
		return "(not IPv4)"
	}
	return netip.AddrFrom4([4]byte(ip[12:16])).String() + " -> " + netip.AddrFrom4([4]byte(ip[16:20])).String()
}
