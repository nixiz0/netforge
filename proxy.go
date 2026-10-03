package main

import (
	"io"
	"log"
	"net"
	"time"
)

const proxyAddr = "127.0.0.1:8080"

// runProxy accepts clients on proxyAddr and relays each one to target (a plain TCP forwarder for now).
func runProxy(target string) {
	// Server toward the client: same Listen/Accept loop as the echo server.
	ln, err := net.Listen("tcp", proxyAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Printf("proxy listening on %s, forwarding to %s", ln.Addr(), target)

	for {
		client, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go handleProxy(client, target)
	}
}

func handleProxy(client net.Conn, target string) {
	defer client.Close()

	// Client toward the target: a second, separate TCP connection (its own handshake, its own 4-tuple).
	upstream, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		log.Println("dial:", err)
		return
	}
	defer upstream.Close()
	log.Printf("%s -> proxy -> %s", client.RemoteAddr(), upstream.RemoteAddr())

	// Relay both ways, like dial does. Client -> target in the background;
	// when the client sends FIN, pass it on to the target with a half-close.
	go func() {
		io.Copy(upstream, client)
		upstream.(*net.TCPConn).CloseWrite()
	}()

	// Target -> client here. When the target sends FIN we return, and the defers close both connections.
	io.Copy(client, upstream)
	log.Printf("%s closed", client.RemoteAddr())
}
