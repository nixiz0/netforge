package tunnel

import (
	"errors"
	"net"
	"net/netip"
	"testing"
)

func TestPeerTable(t *testing.T) {
	srv, c1, c3 := newKey(t), newKey(t), newKey(t)
	a1, a3 := netip.MustParseAddr("10.10.0.1"), netip.MustParseAddr("10.10.0.3")
	table, err := newPeerTable(srv, []Peer{{c1.PublicKey(), a1}, {c3.PublicKey(), a3}}, false)
	if err != nil {
		t.Fatal(err)
	}

	// Identity: the INIT opens only with client 3's static keys, so it is client 3.
	client3, err := newHandshake(c3, srv.PublicKey(), true)
	if err != nil {
		t.Fatal(err)
	}
	from := &net.UDPAddr{IP: net.IPv4(192, 168, 101, 1), Port: 41000}
	p, _, pkt, err := table.identify(client3.start(), from)
	if err != nil || p.Addr != a3 || pkt.typ != typeInit {
		t.Fatalf("identify: peer %v, type %#x, err %v", p, pkt.typ, err)
	}

	// Routing: a packet for 10.10.0.3 goes to client 3, a packet for nobody goes nowhere.
	ip := make([]byte, 20)
	ip[0] = 0x45 // IPv4, 20-byte header
	copy(ip[16:20], a3.AsSlice())
	if got := table.route(ip); got != p {
		t.Fatalf("route to %s: got %v", a3, got)
	}
	ip[19] = 9
	if got := table.route(ip); got != nil {
		t.Fatalf("route to 10.10.0.9: got peer %s, want none", got.Addr)
	}

	// A key that is not in the table opens nothing.
	stranger, err := newHandshake(newKey(t), srv.PublicKey(), true)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := table.identify(stranger.start(), from); !errors.Is(err, errAuth) {
		t.Fatalf("stranger: got %v, want %v", err, errAuth)
	}
}

func TestAllowedSource(t *testing.T) {
	packetFrom := func(src string) []byte {
		ip := make([]byte, 20)
		ip[0] = 0x45
		copy(ip[12:16], netip.MustParseAddr(src).AsSlice())
		return ip
	}
	client2 := &peerState{Peer: Peer{Addr: netip.MustParseAddr("10.10.0.3")}}
	server := &peerState{} // a client's only peer: no VPN address of its own
	cases := []struct {
		name string
		p    *peerState
		src  string
		want bool
	}{
		{"client 2, its own address", client2, "10.10.0.3", true},
		{"client 2 claiming client 1", client2, "10.10.0.1", false},
		{"client 2 claiming an outside host", client2, "203.0.113.10", false},
		{"the server, a reply from the internet", server, "1.1.1.1", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.p.allows(packetFrom(c.src)); got != c.want {
				t.Fatalf("allows(source %s) = %v, want %v", c.src, got, c.want)
			}
		})
	}
}

func TestPeerTableRejectsDuplicateAddress(t *testing.T) {
	srv, c1, c2 := newKey(t), newKey(t), newKey(t)
	a := netip.MustParseAddr("10.10.0.1")
	if _, err := newPeerTable(srv, []Peer{{c1.PublicKey(), a}, {c2.PublicKey(), a}}, false); err == nil {
		t.Fatal("two peers with the same VPN address accepted")
	}
}
