package dns

import (
	"bytes"
	"net/netip"
	"testing"
)

// query builds a DNS query for name (given as labels) and qtype, with ID 0xbeef and RD set.
func query(qtype byte, labels ...string) []byte {
	b := []byte{
		0xbe, 0xef, // ID
		0x01, 0x00, // flags: RD
		0x00, 0x01, // QDCOUNT 1
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // AN, NS, AR
	}
	for _, l := range labels {
		b = append(b, byte(len(l)))
		b = append(b, l...)
	}
	return append(b, 0x00, 0x00, qtype, 0x00, 0x01) // end of name, QTYPE, QCLASS IN
}

var testRecords = map[string]netip.Addr{"host.lab": netip.MustParseAddr("203.0.113.10")}

func TestParseQuery(t *testing.T) {
	msg := query(typeA, "HOST", "lab")
	q, err := parseQuery(msg)
	if err != nil {
		t.Fatal(err)
	}
	if q.id != 0xbeef || q.name != "host.lab" || q.qtype != typeA || q.class != classIN || q.end != len(msg) {
		t.Errorf("got %+v", q)
	}
}

func TestAnswerA(t *testing.T) {
	msg := query(typeA, "host", "lab")
	q, _ := parseQuery(msg)
	resp, result := answer(msg, q, testRecords)
	want := []byte{
		0xbe, 0xef, // same ID
		0x85, 0x00, // QR + AA + RD, RCODE 0
		0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, // QD 1, AN 1, NS 0, AR 0
	}
	want = append(want, msg[headerSize:]...) // the question, copied back
	want = append(want,
		0xc0, 0x0c, // pointer to the name at offset 12
		0x00, 0x01, 0x00, 0x01, // TYPE A, CLASS IN
		0x00, 0x00, 0x00, 0x3c, // TTL 60
		0x00, 0x04, 203, 0, 113, 10, // RDLENGTH 4, the address
	)
	if !bytes.Equal(resp, want) {
		t.Errorf("got  % x\nwant % x", resp, want)
	}
	if result != "203.0.113.10" {
		t.Errorf("result %q", result)
	}
}

func TestAnswerNoRecord(t *testing.T) {
	for _, tc := range []struct {
		name   string
		msg    []byte
		flags  byte // low byte of FLAGS: RCODE
		result string
	}{
		{"unknown name", query(typeA, "nope", "lab"), 0x03, "NXDOMAIN"},
		{"known name, AAAA", query(typeAAAA, "host", "lab"), 0x00, "no AAAA record"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, _ := parseQuery(tc.msg)
			resp, result := answer(tc.msg, q, testRecords)
			if resp[3] != tc.flags || resp[7] != 0 || result != tc.result {
				t.Errorf("flags %#x, ANCOUNT %d, result %q", resp[3], resp[7], result)
			}
		})
	}
}

func TestParseQueryRejects(t *testing.T) {
	valid := query(typeA, "host", "lab")
	with := func(i int, v byte) []byte { b := bytes.Clone(valid); b[i] = v; return b }
	for _, tc := range []struct {
		name string
		msg  []byte
	}{
		{"short header", valid[:11]},
		{"a response", with(2, 0x81)},
		{"not a standard query", with(2, 0x09)},
		{"two questions", with(5, 0x02)},
		{"compression pointer", with(12, 0xc0)},
		{"label cut short", valid[:15]},
		{"question cut short", valid[:len(valid)-1]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseQuery(tc.msg); err == nil {
				t.Fatalf("parseQuery(% x) accepted a malformed query", tc.msg)
			}
		})
	}
}
