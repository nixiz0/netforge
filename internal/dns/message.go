package dns

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

// DNS wire format (RFC 1035), all numbers big-endian:
//
//	header    ID(2) FLAGS(2) QDCOUNT(2) ANCOUNT(2) NSCOUNT(2) ARCOUNT(2)   = 12 bytes
//	question  QNAME (labels: len, bytes, ..., 0) QTYPE(2) QCLASS(2)
//	answer    NAME TYPE(2) CLASS(2) TTL(4) RDLENGTH(2) RDATA
const (
	headerSize = 12
	maxLabel   = 63
	maxName    = 255

	typeA    = 1  // IPv4 address
	typeAAAA = 28 // IPv6 address
	classIN  = 1  // Internet

	flagQR     = 0x8000 // set in responses
	opcodeMask = 0x7800 // 0 = standard query
	flagAA     = 0x0400 // authoritative answer: this server owns the name
	flagRD     = 0x0100 // recursion desired: copied back from the query

	rcodeNXDomain = 3 // the name does not exist

	ttl = 60 // seconds a resolver may cache our answer
)

// question is a parsed DNS query: its header fields and its one question.
type question struct {
	id    uint16
	flags uint16
	name  string // lowercase, without the final dot: "host.lab"
	qtype uint16
	class uint16
	end   int // offset just after the question, so msg[headerSize:end] is the raw question
}

// parseQuery decodes a standard query with exactly one question, which is what resolvers send.
func parseQuery(msg []byte) (question, error) {
	if len(msg) < headerSize {
		return question{}, fmt.Errorf("%d bytes, shorter than a DNS header", len(msg))
	}
	q := question{
		id:    binary.BigEndian.Uint16(msg[0:2]),
		flags: binary.BigEndian.Uint16(msg[2:4]),
	}
	if q.flags&flagQR != 0 {
		return q, errors.New("a response, not a query")
	}
	if q.flags&opcodeMask != 0 {
		return q, errors.New("not a standard query")
	}
	if n := binary.BigEndian.Uint16(msg[4:6]); n != 1 {
		return q, fmt.Errorf("%d questions, want 1", n)
	}

	name, off, err := readName(msg, headerSize)
	if err != nil {
		return q, err
	}
	if off+4 > len(msg) {
		return q, errors.New("question cut short")
	}
	q.name = name
	q.qtype = binary.BigEndian.Uint16(msg[off:])
	q.class = binary.BigEndian.Uint16(msg[off+2:])
	q.end = off + 4
	return q, nil
}

// readName decodes a name written as labels: 4 'h' 'o' 's' 't' 3 'l' 'a' 'b' 0 = "host.lab".
// It returns the name and the offset just after it.
func readName(msg []byte, off int) (string, int, error) {
	var labels []string
	size := 1 // the final zero byte
	for {
		if off >= len(msg) {
			return "", 0, errors.New("name cut short")
		}
		n := int(msg[off])
		off++
		if n == 0 {
			break
		}
		// A length byte starting with bits 11 (0xC0) is a compression pointer. Clients never
		// need one in a question, so anything above 63 is rejected.
		if n > maxLabel {
			return "", 0, fmt.Errorf("label length %d, max %d", n, maxLabel)
		}
		if off+n > len(msg) {
			return "", 0, errors.New("label cut short")
		}
		if size += n + 1; size > maxName {
			return "", 0, fmt.Errorf("name longer than %d bytes", maxName)
		}
		labels = append(labels, string(msg[off:off+n]))
		off += n
	}
	// Names are case-insensitive: HOST.lab and host.lab are the same name.
	return strings.ToLower(strings.Join(labels, ".")), off, nil
}

// answer builds the response to q, and says in words what it answered (for the log).
//
//	known name, A query  → the address (TTL 60 s)
//	known name, other    → no answer, but no error either ("NODATA")
//	unknown name         → NXDOMAIN
func answer(msg []byte, q question, records map[string]netip.Addr) ([]byte, string) {
	ip, known := records[q.name]
	withA := known && q.qtype == typeA && q.class == classIN

	flags := flagQR | flagAA | q.flags&flagRD
	result := "no " + typeName(q.qtype) + " record"
	switch {
	case !known:
		flags |= rcodeNXDomain
		result = "NXDOMAIN"
	case withA:
		result = ip.String()
	}
	var ancount uint16
	if withA {
		ancount = 1
	}

	b := make([]byte, 0, q.end+16)
	b = binary.BigEndian.AppendUint16(b, q.id) // same ID: the client matches the reply to its query
	b = binary.BigEndian.AppendUint16(b, flags)
	b = binary.BigEndian.AppendUint16(b, 1) // QDCOUNT: the question is sent back
	b = binary.BigEndian.AppendUint16(b, ancount)
	b = binary.BigEndian.AppendUint16(b, 0) // NSCOUNT
	b = binary.BigEndian.AppendUint16(b, 0) // ARCOUNT
	b = append(b, msg[headerSize:q.end]...)
	if withA {
		// NAME: a compression pointer (0xC0 | offset) to the name in the question, at offset 12.
		b = binary.BigEndian.AppendUint16(b, 0xC000|headerSize)
		b = binary.BigEndian.AppendUint16(b, typeA)
		b = binary.BigEndian.AppendUint16(b, classIN)
		b = binary.BigEndian.AppendUint32(b, ttl)
		b = binary.BigEndian.AppendUint16(b, 4) // RDLENGTH: an IPv4 address
		a := ip.As4()
		b = append(b, a[:]...)
	}
	return b, result
}

func typeName(t uint16) string {
	switch t {
	case typeA:
		return "A"
	case typeAAAA:
		return "AAAA"
	}
	return fmt.Sprintf("TYPE%d", t)
}
