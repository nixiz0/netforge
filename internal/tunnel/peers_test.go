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

func TestPeerTableRejectsDuplicateAddress(t *testing.T) {
	srv, c1, c2 := newKey(t), newKey(t), newKey(t)
	a := netip.MustParseAddr("10.10.0.1")
	if _, err := newPeerTable(srv, []Peer{{c1.PublicKey(), a}, {c2.PublicKey(), a}}, false); err == nil {
		t.Fatal("two peers with the same VPN address accepted")
	}
}
