package dial

import (
	"io"
	"log"
	"net"
	"os"
	"time"
)

// Run connects to addr and relays bytes both ways: keyboard -> server, server -> screen.
// It returns an error if the connection cannot be made, nil once the server has closed.
func Run(addr string) error {
	// DNS lookup if addr is a name, then socket() + connect(): the kernel sends SYN,
	// Dial returns once the 3-way handshake is done, or fails after 5 s without an answer.
	// Theory: _networking-lessons/2-transport-networking.md#tcp-client-lifecycle
	// Theory: _networking-lessons/1-network-fundamentals.md#18-dns
	// Theory: _networking-lessons/2-transport-networking.md#retransmission
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("connected %s -> %s", conn.LocalAddr(), conn.RemoteAddr())

	// TCP is full-duplex: both directions run at the same time, so each one needs its own goroutine.
	// Theory: _networking-lessons/2-transport-networking.md#tcp-essentials
	// Direction 1: keyboard -> server, in the background, until stdin ends (Ctrl+Z then Enter on Windows).
	go func() {
		if _, err := io.Copy(conn, os.Stdin); err != nil {
			log.Println("send:", err)
		}
		// Half-close: send FIN ("I won't send anymore") but keep reading what the server still sends.
		// Theory: _networking-lessons/2-transport-networking.md#closing-tcp
		log.Println("stdin closed, sending FIN")
		if err := conn.(*net.TCPConn).CloseWrite(); err != nil {
			log.Println("close write:", err)
		}
	}()

	// Direction 2: server -> screen, here. Returns when the server sends its own FIN, then we exit.
	if _, err := io.Copy(os.Stdout, conn); err != nil {
		log.Println("receive:", err)
	}
	log.Println("server closed the connection")
	return nil
}
