package tun

import (
	"encoding/binary"
	"fmt"
	"net/netip"
)

// ipv4Header holds the IPv4 header fields netforge looks at.
type ipv4Header struct {
	headerLen int // IHL × 4: 20 bytes without options
	totalLen  int // header + payload
	ttl       byte
	protocol  byte
	src, dst  netip.Addr
}

// parseIPv4 decodes the fixed part of an IPv4 header (RFC 791). Every field is big-endian:
//
//	byte 0     1     2  3           4  5     6  7            8    9      10 11      12-15  16-19
//	     VER|IHL TOS  TOTAL LENGTH  IDENT    FLAGS|FRAGMENT  TTL  PROTO  CHECKSUM   SRC    DST
//	     45      00   00 54         3a 1c    40 00           40   01     ec 6a      0a0a0001 0a0a0002
//
// Theory: _networking-lessons/6-tun-tap.md#packet-parsing
// Theory: _networking-lessons/1-network-fundamentals.md#headers
func parseIPv4(b []byte) (ipv4Header, error) {
	if len(b) < 20 {
		return ipv4Header{}, fmt.Errorf("too short for an IPv4 header: %d bytes", len(b))
	}
	if v := b[0] >> 4; v != 4 { // high 4 bits of byte 0
		return ipv4Header{}, fmt.Errorf("IP version %d, not 4", v)
	}
	headerLen := int(b[0]&0x0f) * 4 // low 4 bits: header length in 32-bit words
	if headerLen < 20 || headerLen > len(b) {
		return ipv4Header{}, fmt.Errorf("bad header length %d", headerLen)
	}
	totalLen := int(binary.BigEndian.Uint16(b[2:4]))
	if totalLen < headerLen || totalLen > len(b) {
		return ipv4Header{}, fmt.Errorf("total length %d, but the packet has %d bytes", totalLen, len(b))
	}
	return ipv4Header{
		headerLen: headerLen,
		totalLen:  totalLen,
		ttl:       b[8],
		protocol:  b[9],
		src:       netip.AddrFrom4([4]byte(b[12:16])),
		dst:       netip.AddrFrom4([4]byte(b[16:20])),
	}, nil
}

// String formats the header for logs: "10.10.0.1 -> 10.10.0.2 ICMP ttl=64, 84 bytes".
func (h ipv4Header) String() string {
	return fmt.Sprintf("%s -> %s %s ttl=%d, %d bytes", h.src, h.dst, protocolName(h.protocol), h.ttl, h.totalLen)
}

// protocolName names the transport protocol carried in the payload (the IPv4 PROTOCOL field).
func protocolName(p byte) string {
	switch p {
	case 1:
		return "ICMP"
	case 6:
		return "TCP"
	case 17:
		return "UDP"
	default:
		return fmt.Sprintf("protocol %d", p)
	}
}
