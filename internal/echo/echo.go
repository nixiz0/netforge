package echo

import (
	"io"  // general tools for reading and writing (io.Copy, io.EOF)
	"log" // printing with timestamps
	"net" // sockets
)

// Run accepts TCP connections on addr and sends every byte back.
// It only returns if the listener cannot start.
func Run(addr string) error {
	// socket() + bind() + listen(): the kernel now accepts TCP handshakes on this port.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	log.Println("echo server listening on", ln.Addr())

	for {
		// Blocks until the kernel has a fully ESTABLISHED connection in its backlog queue.
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		// Handle each client in its own goroutine so the loop goes straight back to Accept().
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()
	log.Println("new connection from", conn.RemoteAddr())

	// Read from conn, write back to conn, until the client sends FIN (io.EOF).
	n, err := io.Copy(conn, conn)
	if err != nil {
		log.Println("copy:", err)
	}
	log.Printf("%s closed, echoed %d bytes", conn.RemoteAddr(), n)
}
