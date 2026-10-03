package socks

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
)

// Addr is where the SOCKS5 server listens: loopback only, standard SOCKS port.
const Addr = "127.0.0.1:1080"

// SOCKS5 is binary: every field is a byte value, not text.
const (
	version5             = 0x05 // first byte of every SOCKS5 message
	methodNoAuth         = 0x00 // "no authentication required"
	methodNoneAcceptable = 0xFF // "none of your methods is acceptable"
)

// Run accepts SOCKS5 clients on Addr.
// It only returns if the listener cannot start.
func Run() error {
	ln, err := net.Listen("tcp", Addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	log.Println("SOCKS5 server listening on", ln.Addr())

	for {
		client, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go handle(client)
	}
}

func handle(client net.Conn) {
	defer client.Close()

	if err := greet(client); err != nil {
		log.Printf("%s greeting: %v", client.RemoteAddr(), err)
		return
	}
	log.Printf("%s greeting done, no authentication", client.RemoteAddr())
	// Next (step 3.2): read the CONNECT request.
}

// greet runs the SOCKS5 method negotiation:
//
//	client -> server:  VER  NMETHODS  METHODS...   e.g. 05 01 00
//	server -> client:  VER  METHOD                 e.g. 05 00 (no authentication)
func greet(conn net.Conn) error {
	// Length-prefix framing: a fixed 2-byte header, and its second byte says how many bytes follow.
	// io.ReadFull keeps reading until it has exactly len(buf) bytes, however TCP splits them.
	head := make([]byte, 2)
	if _, err := io.ReadFull(conn, head); err != nil {
		return err
	}
	if head[0] != version5 {
		return fmt.Errorf("not SOCKS5 (version byte %#x)", head[0])
	}

	methods := make([]byte, head[1])
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}
	log.Printf("%s greeting: % x % x", conn.RemoteAddr(), head, methods)

	for _, m := range methods {
		if m == methodNoAuth {
			_, err := conn.Write([]byte{version5, methodNoAuth})
			return err
		}
	}
	conn.Write([]byte{version5, methodNoneAcceptable})
	return errors.New("client offers no method we support")
}
