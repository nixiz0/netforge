package tun

import "encoding/binary"

const (
	protoICMP       = 1 // IPv4 PROTOCOL field value for ICMP
	icmpEchoReply   = 0 // ICMP type of a ping answer
	icmpEchoRequest = 8 // ICMP type of a ping
)

// checksum is the Internet checksum (RFC 1071) used by IPv4, ICMP, UDP and TCP: add the data as
// 16-bit big-endian words, fold every carry back into the low 16 bits, then flip all the bits.
// A block that already contains its correct checksum sums to 0: that is how a receiver checks it.
//
// Theory: _networking-lessons/1-network-fundamentals.md#headers
func checksum(b []byte) uint16 {
	var sum uint32
	for i := 0; i+1 < len(b); i += 2 {
		sum += uint32(b[i])<<8 | uint32(b[i+1])
	}
	if len(b)%2 == 1 {
		sum += uint32(b[len(b)-1]) << 8 // an odd last byte counts as the high half of a word
	}
	for sum>>16 != 0 {
		sum = sum&0xffff + sum>>16
	}
	return ^uint16(sum)
}

// echoReply turns an IPv4 ICMP echo request (a ping) into the matching echo reply, ready to be
// written back into the TUN interface. It returns nil for anything else, including fragments.
//
//	request: 10.10.0.1 -> 10.10.0.2  ICMP type 8, id, seq, data
//	reply:   10.10.0.2 -> 10.10.0.1  ICMP type 0, same id, seq, data
//
// Theory: _networking-lessons/1-network-fundamentals.md#ping
// Theory: _networking-lessons/6-tun-tap.md#vpn-receive-path
func echoReply(pkt []byte, h ipv4Header) []byte {
	if h.protocol != protoICMP || h.fragmented {
		return nil
	}
	icmp := pkt[h.headerLen:h.totalLen]
	if len(icmp) < 8 || icmp[0] != icmpEchoRequest || icmp[1] != 0 {
		return nil
	}

	r := make([]byte, h.totalLen)
	copy(r, pkt[:h.totalLen])

	// IPv4 header: swap source and destination, fresh TTL, then recompute the header checksum
	// (the checksum field must be zero while computing it).
	copy(r[12:16], pkt[16:20])
	copy(r[16:20], pkt[12:16])
	r[8] = 64
	binary.BigEndian.PutUint16(r[10:12], 0)
	binary.BigEndian.PutUint16(r[10:12], checksum(r[:h.headerLen]))

	// ICMP: type 0 = echo reply. Identifier, sequence number and data stay unchanged, so ping can
	// match the reply to its request. The ICMP checksum covers the whole ICMP message.
	ri := r[h.headerLen:]
	ri[0] = icmpEchoReply
	binary.BigEndian.PutUint16(ri[2:4], 0)
	binary.BigEndian.PutUint16(ri[2:4], checksum(ri))
	return r
}
