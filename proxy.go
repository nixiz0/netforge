package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"net/url"
	"strings"
	"time"
)

const proxyAddr = "127.0.0.1:8080"

// runProxy is an HTTP forward proxy on proxyAddr: each request says where it wants to go.
func runProxy() {
	// Server toward the client: same Listen/Accept loop as the echo server.
	ln, err := net.Listen("tcp", proxyAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Println("HTTP proxy listening on", ln.Addr())

	for {
		client, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go handleProxy(client)
	}
}

func handleProxy(client net.Conn) {
	defer client.Close()

	// Read the HTTP request head (request line + headers) before connecting anywhere.
	br := bufio.NewReader(client)
	head, err := readHead(br)
	if err != nil {
		log.Println("read request:", err)
		return
	}
	log.Printf("%s request line: %q", client.RemoteAddr(), head[0])

	// A client talking to a proxy sends the full URL: "GET http://example.com/path HTTP/1.1".
	parts := strings.Fields(head[0])
	if len(parts) != 3 {
		log.Println("bad request line")
		return
	}
	method, rawURL, version := parts[0], parts[1], parts[2]

	// HTTPS: "CONNECT example.com:443 HTTP/1.1". The client wants a raw TCP tunnel and will do TLS
	// through it, end to end with the website: after the 200 reply we can only relay encrypted bytes.
	if method == "CONNECT" {
		upstream, err := net.DialTimeout("tcp", rawURL, 5*time.Second)
		if err != nil {
			log.Println("dial:", err)
			io.WriteString(client, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
			return
		}
		defer upstream.Close()
		log.Printf("%s -> proxy -> %s (CONNECT tunnel)", client.RemoteAddr(), upstream.RemoteAddr())

		io.WriteString(client, "HTTP/1.1 200 Connection Established\r\n\r\n")
		relay(client, br, upstream)
		return
	}

	// Plain HTTP: the full URL in the request line tells us where to go.
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		log.Println("not a proxy request:", rawURL)
		return
	}

	// The destination comes from the URL; port 80 is the default for http://.
	target := u.Host
	if u.Port() == "" {
		target = net.JoinHostPort(u.Hostname(), "80")
	}

	// The website expects the short form "GET /path HTTP/1.1", so rewrite the request line.
	head[0] = method + " " + u.RequestURI() + " " + version + "\r\n"

	// Client toward the target: a second, separate TCP connection (its own handshake, its own 4-tuple).
	upstream, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		log.Println("dial:", err)
		io.WriteString(client, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
		return
	}
	defer upstream.Close()
	log.Printf("%s -> proxy -> %s", client.RemoteAddr(), upstream.RemoteAddr())

	// We consumed the head from the client, so send it to the target ourselves, plus the empty line.
	for _, line := range head {
		io.WriteString(upstream, line)
	}
	io.WriteString(upstream, "\r\n")

	relay(client, br, upstream)
}

// relay copies bytes both ways until the target closes, like dial does.
func relay(client net.Conn, br *bufio.Reader, upstream net.Conn) {
	// Client -> target in the background, reading from br and not client: bufio may already hold
	// bytes that came after the head (a body, or the start of the TLS handshake). Pass the client's FIN on.
	go func() {
		io.Copy(upstream, br)
		upstream.(*net.TCPConn).CloseWrite()
	}()

	// Target -> client here. When the target sends FIN we return, and the caller's defers close both connections.
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
