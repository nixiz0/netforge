package tunnel

import (
	"bytes"
	"testing"
)

func testSealer(t *testing.T, fill byte) *sealer {
	t.Helper()
	key := bytes.Repeat([]byte{fill}, keySize)
	s, err := newSealer(key, key) // same key both ways: one sealer can stand for either end
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSealOpen(t *testing.T) {
	a, b := testSealer(t, 1), testSealer(t, 1) // two ends, same key, different nonce prefixes
	inner := []byte("an inner IP packet")
	wire := a.seal(typeData, inner)

	if len(wire) != headerSize+sealOverhead+len(inner) {
		t.Fatalf("sealed packet is %d bytes, want %d", len(wire), headerSize+sealOverhead+len(inner))
	}
	if bytes.Contains(wire, inner) {
		t.Fatal("the inner packet is readable on the wire")
	}
	pkt, err := b.open(wire)
	if err != nil {
		t.Fatal(err)
	}
	if pkt.typ != typeData || !bytes.Equal(pkt.payload, inner) {
		t.Fatalf("got type %#x payload %q", pkt.typ, pkt.payload)
	}
}

func TestNoncesNeverRepeat(t *testing.T) {
	s := testSealer(t, 1)
	n1 := s.seal(typeKeepalive, nil)[headerSize : headerSize+nonceSize]
	n2 := s.seal(typeKeepalive, nil)[headerSize : headerSize+nonceSize]
	if bytes.Equal(n1, n2) {
		t.Fatalf("same nonce twice: % x", n1)
	}
}

// Any modified byte, or another key, must be rejected: that is what stops a hijack.
func TestOpenRejects(t *testing.T) {
	s := testSealer(t, 1)
	wire := s.seal(typeData, []byte("ping"))
	flip := func(i int) []byte { b := bytes.Clone(wire); b[i] ^= 0x01; return b }
	cases := []struct {
		name string
		b    []byte
	}{
		{"modified type (header)", flip(1)},
		{"modified nonce", flip(headerSize)},
		{"modified ciphertext", flip(headerSize + nonceSize)},
		{"modified tag", flip(len(wire) - 1)},
		{"other key", testSealer(t, 2).seal(typeData, []byte("ping"))},
		{"plaintext packet", packet{typ: typeData, payload: []byte("ping")}.encode()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := s.open(c.b); err == nil {
				t.Fatalf("open(% x) accepted it", c.b)
			}
		})
	}
}

func TestVPNMTU(t *testing.T) {
	wire := testSealer(t, 1).seal(typeData, make([]byte, vpnMTU))
	if total := len(wire) + udpHeader + ipv4Header; total != linkMTU {
		t.Fatalf("a %d-byte inner packet takes %d bytes on the wire, want exactly %d", vpnMTU, total, linkMTU)
	}
}
