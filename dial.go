package main

import (
	"log"
	"net"
)

// runDial connects to addr, sends one message and prints the reply.
func runDial(addr string) {
	// socket() + connect(): the kernel sends SYN, Dial returns once the 3-way handshake is done.
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	log.Printf("connected %s -> %s", conn.LocalAddr(), conn.RemoteAddr())

	// Write returns once the bytes are in the kernel send buffer, not when the server got them.
	msg := []byte("hello netforge\n")
	n, err := conn.Write(msg)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("sent %d bytes", n)

	// Read blocks until some bytes arrive, and returns how many it put in buf.
	buf := make([]byte, 1024)
	n, err = conn.Read(buf)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("received %d bytes: %q", n, buf[:n])
}
