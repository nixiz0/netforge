package dial

import (
	"bufio"
	"log"
	"net"
	"os"
	"time"
)

// RunUDP sends each line typed on stdin as one UDP datagram to addr and prints every datagram
// that comes back. It returns once stdin ends.
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

	// Replies in the background: one Read = exactly one datagram.
	go func() {
		buf := make([]byte, 65535)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				log.Println("receive:", err)
				return
			}
			log.Printf("<- %d bytes: %q", n, buf[:n])
		}
	}()

	// One line = one Write = one datagram. Its boundaries survive the trip, unlike TCP's byte stream.
	// Theory: _networking-lessons/2-transport-networking.md#udp-preserves-datagram-boundaries
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		msg := sc.Bytes()
		if _, err := conn.Write(msg); err != nil {
			log.Println("send:", err)
			continue
		}
		log.Printf("-> %d bytes: %q", len(msg), msg)
	}

	// UDP has no FIN: nothing tells us the last replies have arrived, so give them a moment.
	// Step 4.3 replaces this guess with a read deadline.
	time.Sleep(time.Second)
	log.Println("stdin closed; UDP has no FIN, nothing to close on the wire")
	return sc.Err()
}
