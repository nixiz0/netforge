package echo

import (
	"log"
	"net"
)

// RunUDP answers every datagram received on addr with the same bytes, sent back to whoever sent it.
// It only returns if the socket cannot be opened.
//
// Theory: _networking-lessons/2-transport-networking.md#udp-essentials
func RunUDP(addr string) error {
	// socket() + bind() only: no listen, no accept, no handshake. One socket serves every client.
	// UDP ports are a separate space from TCP ports, so this can share 9000 with the TCP echo server.
	// Theory: _networking-lessons/2-transport-networking.md#udp-lifecycle
	// Theory: _networking-lessons/2-transport-networking.md#ports-and-connection-identity
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer pc.Close()
	log.Println("UDP echo server listening on", pc.LocalAddr())

	// 65535 bytes: large enough for any UDP datagram, so nothing gets cut.
	buf := make([]byte, 65535)
	for {
		// One ReadFrom = exactly one datagram, plus the address it came from.
		// Theory: _networking-lessons/2-transport-networking.md#udp-preserves-datagram-boundaries
		n, from, err := pc.ReadFrom(buf)
		if err != nil {
			log.Println("read:", err)
			continue
		}
		log.Printf("%d bytes from %s: %q", n, from, buf[:n])

		// No connection to answer on: every reply must name its destination.
		if _, err := pc.WriteTo(buf[:n], from); err != nil {
			log.Println("write:", err)
		}
	}
}
