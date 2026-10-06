package dial

import (
	"bufio"
	"errors"
	"log"
	"net"
	"os"
	"time"
)

// RunUDP sends each line typed on stdin as one UDP datagram to addr and prints every datagram
// that comes back. Once stdin ends it waits 2 s for late replies, then reports how many were lost.
//
// Theory: _networking-lessons/2-transport-networking.md#udp-essentials
func RunUDP(addr string) error {
	// "Connected" UDP: this only records a default peer locally, so Write needs no address.
	// Nothing is sent on the wire: no SYN, no handshake.
	// Theory: _networking-lessons/2-transport-networking.md#udp-lifecycle
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("UDP socket %s -> %s (no handshake sent)", conn.LocalAddr(), conn.RemoteAddr())

	// Replies in the background: one Read = exactly one datagram. The goroutine sends its count
	// back on the done channel when the read deadline ends it.
	done := make(chan int)
	go func() {
		received := 0
		buf := make([]byte, 65535)
		for {
			n, err := conn.Read(buf)
			switch {
			case err == nil:
				received++
				log.Printf("<- %d bytes: %q", n, buf[:n])
			case errors.Is(err, os.ErrDeadlineExceeded), errors.Is(err, net.ErrClosed):
				done <- received
				return
			default:
				// The kernel got an ICMP "port unreachable" back for an earlier datagram. It shows up
				// on a later Read, never on the Write. Not fatal: keep reading.
				// Theory: _networking-lessons/1-network-fundamentals.md#19-icmp-ping-traceroute-and-ip-ttl
				log.Println("receive:", err, "(ICMP error for an earlier datagram)")
			}
		}
	}()

	// One line = one Write = one datagram. Its boundaries survive the trip, unlike TCP's byte stream.
	// Write succeeds even if nobody listens: UDP never checks.
	// Theory: _networking-lessons/2-transport-networking.md#udp-preserves-datagram-boundaries
	sent := 0
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		msg := sc.Bytes()
		if _, err := conn.Write(msg); err != nil {
			log.Println("send:", err)
			continue
		}
		sent++
		log.Printf("-> %d bytes: %q", len(msg), msg)
	}

	// UDP has no FIN and no acknowledgments: only we can decide when to stop waiting.
	// A read deadline makes the blocked Read return an error after 2 s.
	// Theory: _networking-lessons/2-transport-networking.md#tcp-vs-udp
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	received := <-done
	log.Printf("sent %d datagrams, got %d replies, %d lost or unanswered (UDP never says which)", sent, received, sent-received)
	return sc.Err()
}
