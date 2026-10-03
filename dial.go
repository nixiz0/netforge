package main

import (
	"io"
	"log"
	"net"
	"os"
)

// runDial connects to addr and relays bytes both ways: keyboard -> server, server -> screen.
func runDial(addr string) {
	// socket() + connect(): the kernel sends SYN, Dial returns once the 3-way handshake is done.
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	log.Printf("connected %s -> %s", conn.LocalAddr(), conn.RemoteAddr())

	// TCP is full-duplex: both directions run at the same time, so each one needs its own goroutine.
	// Direction 1: server -> screen, in the background.
	go func() {
		if _, err := io.Copy(os.Stdout, conn); err != nil {
			log.Println("receive:", err)
		}
		log.Println("server closed the connection")
	}()

	// Direction 2: keyboard -> server, until stdin ends (Ctrl+Z then Enter on Windows).
	if _, err := io.Copy(conn, os.Stdin); err != nil {
		log.Println("send:", err)
	}
}
