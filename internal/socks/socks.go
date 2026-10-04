package socks

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
)

// Addr is where the SOCKS5 server listens: loopback only, standard SOCKS port.
const Addr = "127.0.0.1:1080"

// SOCKS5 is binary: every field is a byte value, not text.
const (
	version5             = 0x05 // first byte of every SOCKS5 message
	methodNoAuth         = 0x00 // "no authentication required"
	methodNoneAcceptable = 0xFF // "none of your methods is acceptable"

	cmdConnect = 0x01 // "open a TCP connection to this address"

	atypIPv4   = 0x01 // address = 4 bytes
	atypDomain = 0x03 // address = 1 length byte + that many bytes of name
	atypIPv6   = 0x04 // address = 16 bytes
)

// Run accepts SOCKS5 clients on Addr.
// It only returns if the listener cannot start.
//
// Theory: _networking-lessons/3-http-tls-proxies.md#socks5-concept
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

	target, err := readRequest(client)
	if err != nil {
		log.Printf("%s request: %v", client.RemoteAddr(), err)
		return
	}
	log.Printf("%s CONNECT %s", client.RemoteAddr(), target)
	// Next (step 3.4): dial target, send the reply, relay.
}

// greet runs the SOCKS5 method negotiation:
//
//	client -> server:  VER  NMETHODS  METHODS...   e.g. 05 01 00
//	server -> client:  VER  METHOD                 e.g. 05 00 (no authentication)
//
// Theory: _networking-lessons/3-http-tls-proxies.md#socks5-handshake
func greet(conn net.Conn) error {
	// Length-prefix framing: a fixed 2-byte header, and its second byte says how many bytes follow.
	// io.ReadFull keeps reading until it has exactly len(buf) bytes, however TCP splits them.
	// Theory: _networking-lessons/2-transport-networking.md#tcp-is-a-byte-stream
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

// readRequest reads the CONNECT request and returns the target as "host:port":
//
//	VER  CMD  RSV  ATYP  DST.ADDR                DST.PORT
//	05   01   00   03    0b "example.com"        00 50     (domain, port 80)
//	05   01   00   01    5d b8 d7 0e             01 bb     (IPv4, port 443)
//
// Theory: _networking-lessons/3-http-tls-proxies.md#socks5-concept
func readRequest(conn net.Conn) (string, error) {
	head := make([]byte, 4)
	if _, err := io.ReadFull(conn, head); err != nil {
		return "", err
	}
	if head[0] != version5 {
		return "", fmt.Errorf("bad version byte %#x", head[0])
	}
	if head[1] != cmdConnect {
		return "", fmt.Errorf("unsupported command %#x", head[1])
	}

	// The address type (ATYP) says how many bytes the address takes.
	var host string
	switch head[3] {
	case atypIPv4, atypIPv6:
		size := net.IPv4len
		if head[3] == atypIPv6 {
			size = net.IPv6len
		}
		ip := make([]byte, size)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", err
		}
		host = net.IP(ip).String()
	case atypDomain:
		// A name has no fixed size, so it is length-prefixed: 1 byte of length, then the name.
		// Theory: _networking-lessons/3-http-tls-proxies.md#dns-with-socks5
		n := make([]byte, 1)
		if _, err := io.ReadFull(conn, n); err != nil {
			return "", err
		}
		name := make([]byte, n[0])
		if _, err := io.ReadFull(conn, name); err != nil {
			return "", err
		}
		host = string(name)
	default:
		return "", fmt.Errorf("unknown address type %#x", head[3])
	}

	// Port: 2 bytes in network byte order (big-endian, high byte first): 00 50 = 0*256 + 80 = 80.
	port := make([]byte, 2)
	if _, err := io.ReadFull(conn, port); err != nil {
		return "", err
	}
	return net.JoinHostPort(host, strconv.Itoa(int(binary.BigEndian.Uint16(port)))), nil
}
