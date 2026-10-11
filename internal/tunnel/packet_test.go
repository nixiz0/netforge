package tunnel

import (
	"bytes"
	"testing"
)

// The exact bytes on the wire for one data packet: header 01 01 00 09, then the payload.
func TestEncode(t *testing.T) {
	got := packet{typ: typeData, payload: []byte("hello udp")}.encode()
	want := append([]byte{0x01, 0x01, 0x00, 0x09}, "hello udp"...)
	if !bytes.Equal(got, want) {
		t.Fatalf("encode = % x, want % x", got, want)
	}
}

// Whatever goes in through encode must come out of decode unchanged.
func TestRoundTrip(t *testing.T) {
	for _, in := range []packet{
		{typ: typeData, payload: []byte("hello udp")},
		{typ: typeData, payload: bytes.Repeat([]byte("x"), 1400)},
		{typ: typeKeepalive, payload: []byte{}},
	} {
		out, err := decode(in.encode())
		if err != nil {
			t.Fatalf("decode(encode(%v)): %v", in.typ, err)
		}
		if out.typ != in.typ || !bytes.Equal(out.payload, in.payload) {
			t.Fatalf("round trip changed the packet: got type %d, %d bytes", out.typ, len(out.payload))
		}
	}
}

// A full-size inner datagram, once wrapped, must take exactly one Ethernet frame on the wire.
func TestTunnelMTU(t *testing.T) {
	pkt := packet{typ: typeData, payload: make([]byte, tunnelMTU)}.encode()
	if wire := len(pkt) + udpHeader + ipv4Header; wire != linkMTU {
		t.Fatalf("a %d-byte inner datagram takes %d bytes on the wire, want exactly %d", tunnelMTU, wire, linkMTU)
	}
}

// decode must refuse anything malformed instead of passing garbage on.
func TestDecodeRejects(t *testing.T) {
	cases := []struct {
		name string
		b    []byte
	}{
		{"empty", nil},
		{"shorter than the header", []byte{0x01, 0x01}},
		{"unknown version", []byte{0x02, 0x01, 0x00, 0x00}},
		{"length too big", []byte{0x01, 0x01, 0x00, 0x05, 'a'}},
		{"length too small", []byte{0x01, 0x01, 0x00, 0x00, 'a'}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := decode(c.b); err == nil {
				t.Fatalf("decode(% x) accepted a malformed packet", c.b)
			}
		})
	}
}
