package main

import (
	"bufio"
	"log"
	"net"
	"time"
)

// runDial connects to addr, sends two messages and reads them back one line at a time.
func runDial(addr string) {
	// socket() + connect(): the kernel sends SYN, Dial returns once the 3-way handshake is done.
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	log.Printf("connected %s -> %s", conn.LocalAddr(), conn.RemoteAddr())

	// Two separate messages, two separate Write calls, with a pause after each.
	// Write returns once the bytes are in the kernel send buffer, not when the server got them.
	for _, msg := range []string{"hello", "netforge\n"} {
		n, err := conn.Write([]byte(msg))
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("sent %d bytes: %q", n, msg)
		time.Sleep(100 * time.Millisecond)
	}

	// A single Read returns whatever bytes are waiting (here both messages glued together).
	// Delimiter framing: read up to and including '\n', exactly one message per call,
	// however TCP merged or split the bytes. bufio keeps leftover bytes for the next call.
	r := bufio.NewReader(conn)
	for i := 0; i < 2; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("message %d: %q", i+1, line)
	}
}
