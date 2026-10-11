package tun

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// The worked example from RFC 1071: these 8 bytes sum to 0xddf2, so their checksum is 0x220d.
func TestChecksumRFC1071(t *testing.T) {
	if got := checksum([]byte{0x00, 0x01, 0xf2, 0x03, 0xf4, 0xf5, 0xf6, 0xf7}); got != 0x220d {
		t.Fatalf("checksum = %#04x, want 0x220d", got)
	}
}

// A header that carries its correct checksum must sum to zero.
func TestChecksumOfValidHeader(t *testing.T) {
	if got := checksum(pingHeader); got != 0 {
		t.Fatalf("checksum over a valid IPv4 header = %#04x, want 0", got)
	}
}

// echoRequest builds a full ping packet: pingHeader + ICMP type 8, id 0x1234, seq 1, 56 bytes of data.
func echoRequest() []byte {
	icmp := make([]byte, 64)
	icmp[0] = icmpEchoRequest
	binary.BigEndian.PutUint16(icmp[4:6], 0x1234)
	binary.BigEndian.PutUint16(icmp[6:8], 1)
	copy(icmp[8:], "netforge ping data")
	binary.BigEndian.PutUint16(icmp[2:4], checksum(icmp))
	return append(append([]byte{}, pingHeader...), icmp...)
}

func TestEchoReply(t *testing.T) {
	req := echoRequest()
	h, err := parseIPv4(req)
	if err != nil {
		t.Fatal(err)
	}
	r := echoReply(req, h)
	if r == nil {
		t.Fatal("echoReply returned nil for a ping")
	}
	if !bytes.Equal(r[12:16], req[16:20]) || !bytes.Equal(r[16:20], req[12:16]) {
		t.Fatal("source and destination were not swapped")
	}
	if checksum(r[:20]) != 0 {
		t.Fatal("reply IPv4 header checksum is wrong")
	}
	if r[20] != icmpEchoReply {
		t.Fatalf("ICMP type = %d, want %d (echo reply)", r[20], icmpEchoReply)
	}
	if checksum(r[20:]) != 0 {
		t.Fatal("reply ICMP checksum is wrong")
	}
	if !bytes.Equal(r[24:], req[24:]) {
		t.Fatal("identifier, sequence or data changed")
	}
}

func TestEchoReplyIgnores(t *testing.T) {
	cases := []struct {
		name   string
		change func(p []byte)
	}{
		{"not ICMP (TCP)", func(p []byte) { p[9] = 6 }},
		{"ICMP but not an echo request", func(p []byte) { p[20] = icmpEchoReply }},
		{"a fragment", func(p []byte) { p[6] = 0x20 }}, // "more fragments" flag set
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := echoRequest()
			c.change(p)
			h, err := parseIPv4(p)
			if err != nil {
				t.Fatal(err)
			}
			if echoReply(p, h) != nil {
				t.Fatal("echoReply answered something that is not a plain ping")
			}
		})
	}
}
