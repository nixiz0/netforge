package tunnel

import (
	"encoding/binary"
	"fmt"
)

// A tunnel packet is what travels inside each outer UDP datagram: a 4-byte header, then the
// inner traffic as payload.
//
//	VER  TYPE  LENGTH (2 bytes, big-endian)  PAYLOAD...
//	01   01    00 09                         "hello udp"
//
// Theory: _networking-lessons/5-tunneling.md#encapsulation
const (
	version1   = 0x01
	headerSize = 4

	typeData      = 0x01 // payload = one inner datagram
	typeKeepalive = 0x02 // no payload: "I'm still here" (step 5.5)
)

// packet is one tunnel message, decoded.
type packet struct {
	typ     byte
	payload []byte
}

// encode wraps the payload in the tunnel header: encapsulation.
//
// LENGTH is redundant over UDP, which already keeps datagram boundaries, but it lets decode
// catch damaged packets, and it is the framing the tunnel would need over a TCP byte stream.
// Theory: _networking-lessons/5-tunneling.md#tcp-tunneling
func (p packet) encode() []byte {
	b := make([]byte, 0, headerSize+len(p.payload))
	b = append(b, version1, p.typ)
	b = binary.BigEndian.AppendUint16(b, uint16(len(p.payload)))
	return append(b, p.payload...)
}

// decode parses one outer datagram back into a packet: decapsulation.
// It rejects anything that is not a well-formed version-1 tunnel packet.
func decode(b []byte) (packet, error) {
	if len(b) < headerSize {
		return packet{}, fmt.Errorf("tunnel packet too short: %d bytes", len(b))
	}
	if b[0] != version1 {
		return packet{}, fmt.Errorf("unknown tunnel version %#x", b[0])
	}
	n := int(binary.BigEndian.Uint16(b[2:4]))
	if n != len(b)-headerSize {
		return packet{}, fmt.Errorf("length field says %d bytes, datagram carries %d", n, len(b)-headerSize)
	}
	return packet{typ: b[1], payload: b[headerSize:]}, nil
}
