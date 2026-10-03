package main

import (
	"bufio"
	"errors"
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

	// Read the HTTP request head (request line + headers) before connecting anywhere.
	br := bufio.NewReader(client)
	head, err := readHead(br)
	if err != nil {
		log.Println("read request:", err)
		return
	}
	log.Printf("%s request line: %q", client.RemoteAddr(), head[0])
	for _, h := range head[1:] {
		log.Printf("  header: %q", h)
	}

	// Client toward the target: a second, separate TCP connection (its own handshake, its own 4-tuple).
	upstream, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		log.Println("dial:", err)
		return
	}
	defer upstream.Close()
	log.Printf("%s -> proxy -> %s", client.RemoteAddr(), upstream.RemoteAddr())

	// We consumed the head from the client, so send it to the target ourselves, plus the empty line.
	for _, line := range head {
		io.WriteString(upstream, line)
	}
	io.WriteString(upstream, "\r\n")

	// Relay both ways, like dial does. Client -> target in the background, reading from br and not
	// client: bufio may already hold bytes that came after the head (a body). Pass the client's FIN on.
	go func() {
		io.Copy(upstream, br)
		upstream.(*net.TCPConn).CloseWrite()
	}()

	// Target -> client here. When the target sends FIN we return, and the defers close both connections.
	io.Copy(client, upstream)
	log.Printf("%s closed", client.RemoteAddr())
}

// readHead reads the request line and the headers, up to the empty line that ends them.
// HTTP framing: every line ends with \r\n, and an empty line marks the end of the head.
func readHead(br *bufio.Reader) ([]string, error) {
	var lines []string
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if line == "\r\n" || line == "\n" {
			if len(lines) == 0 {
				return nil, errors.New("empty request")
			}
			return lines, nil
		}
		lines = append(lines, line)
	}
}
