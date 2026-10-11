package tun

import (
	"net/netip"
	"testing"
)

// A real ping captured on nf0 in step 6.1: a 20-byte header, then 64 bytes of ICMP.
var pingHeader = []byte{
	0x45, 0x00, 0x00, 0x54, 0x3a, 0x1c, 0x40, 0x00, 0x40, 0x01, 0xec, 0x6a,
	0x0a, 0x0a, 0x00, 0x01, 0x0a, 0x0a, 0x00, 0x02,
}

func pingPacket() []byte { return append(append([]byte{}, pingHeader...), make([]byte, 64)...) }

func TestParseIPv4(t *testing.T) {
	h, err := parseIPv4(pingPacket())
	if err != nil {
		t.Fatal(err)
	}
	want := ipv4Header{
		headerLen: 20, totalLen: 84, ttl: 64, protocol: 1,
		src: netip.MustParseAddr("10.10.0.1"), dst: netip.MustParseAddr("10.10.0.2"),
	}
	if h != want {
		t.Fatalf("parseIPv4 = %+v, want %+v", h, want)
	}
	if got := h.String(); got != "10.10.0.1 -> 10.10.0.2 ICMP ttl=64, 84 bytes" {
		t.Fatalf("String() = %q", got)
	}
}

func TestParseIPv4Rejects(t *testing.T) {
	withByte := func(i int, v byte) []byte { p := pingPacket(); p[i] = v; return p }
	cases := []struct {
		name string
		b    []byte
	}{
		{"shorter than a header", pingHeader[:19]},
		{"IPv6 packet", withByte(0, 0x60)},
		{"header length below 20", withByte(0, 0x44)},
		{"total length beyond the packet", withByte(3, 0xff)},
		{"total length smaller than the header", withByte(3, 0x10)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := parseIPv4(c.b); err == nil {
				t.Fatal("parseIPv4 accepted a malformed packet")
			}
		})
	}
}
