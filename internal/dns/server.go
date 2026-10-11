// Package dns is a tiny authoritative DNS server: it answers A queries for a few lab names.
// Run on the VPN server, it gives clients a resolver they can only reach through the tunnel.
//
// Theory: _networking-lessons/1-network-fundamentals.md#18-dns
// Theory: _networking-lessons/7-vpn-architecture.md#dns-through-a-vpn
package dns

import (
	"log"
	"net"
	"net/netip"
)

// Addr is the default listen address: a high port, so it runs without root outside the lab.
const Addr = "127.0.0.1:5353"

// Records is the zone this server is authoritative for: it owns these names and answers them
// itself, instead of asking other servers like a recursive resolver would.
//
// Theory: _networking-lessons/1-network-fundamentals.md#recursive-vs-authoritative
// Theory: _networking-lessons/1-network-fundamentals.md#important-record-types
var Records = map[string]netip.Addr{
	"host.lab": netip.MustParseAddr("203.0.113.10"), // the machine behind the VPN server
	"vpn.lab":  netip.MustParseAddr("10.10.0.2"),    // the VPN server's inner address
}

// Run answers DNS queries on addr over UDP. It only returns if the socket cannot be opened.
func Run(addr string) error {
	// DNS queries are single UDP datagrams to port 53: no handshake, one question, one answer.
	// Theory: _networking-lessons/2-transport-networking.md#udp-essentials
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer pc.Close()
	log.Println("DNS server listening on", pc.LocalAddr(), "for host.lab and vpn.lab")

	// 512 bytes: the classic limit of a DNS message over UDP. Queries are far smaller.
	buf := make([]byte, 512)
	for {
		n, from, err := pc.ReadFrom(buf)
		if err != nil {
			log.Println("read:", err)
			continue
		}
		q, err := parseQuery(buf[:n])
		if err != nil {
			log.Printf("bad query from %s: %v", from, err)
			continue
		}
		resp, result := answer(buf[:n], q, Records)
		if _, err := pc.WriteTo(resp, from); err != nil {
			log.Println("write:", err)
			continue
		}
		log.Printf("%s asked %s %s -> %s", from, typeName(q.qtype), q.name, result)
	}
}
