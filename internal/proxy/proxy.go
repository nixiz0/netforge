package proxy

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/nixiz0/netforge/internal/relay"
	"github.com/nixiz0/netforge/internal/resolve"
)

// Addr is where the proxy listens: loopback only.
//
// Theory: _networking-lessons/1-network-fundamentals.md#5-loopback
const Addr = "127.0.0.1:8080"

// Run is an HTTP forward proxy on Addr: each request says where it wants to go.
// It only returns if the listener cannot start.
//
// Theory: _networking-lessons/3-http-tls-proxies.md#forward-proxy
func Run() error {
	// Server toward the client: same Listen/Accept loop as the echo server.
	ln, err := net.Listen("tcp", Addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	log.Println("HTTP proxy listening on", ln.Addr())

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

	// Read the HTTP request head (request line + headers) before connecting anywhere.
	// Theory: _networking-lessons/3-http-tls-proxies.md#http-request-structure
	br := bufio.NewReader(client)
	head, err := readHead(br)
	if err != nil {
		log.Println("read request:", err)
		return
	}
	log.Printf("%s request line: %q", client.RemoteAddr(), head[0])

	// A client talking to a proxy sends the full URL: "GET http://example.com/path HTTP/1.1".
	// Theory: _networking-lessons/3-http-tls-proxies.md#plain-http-proxying
	parts := strings.Fields(head[0])
	if len(parts) != 3 {
		log.Println("bad request line")
		return
	}
	method, rawURL, version := parts[0], parts[1], parts[2]

	// HTTPS: "CONNECT example.com:443 HTTP/1.1". The client wants a raw TCP tunnel and will do TLS
	// through it, end to end with the website: after the 200 reply we can only relay encrypted bytes.
	// Theory: _networking-lessons/3-http-tls-proxies.md#http-connect
	// Theory: _networking-lessons/3-http-tls-proxies.md#what-the-connect-proxy-can-still-know
	if method == "CONNECT" {
		upstream, lk, err := resolve.Dial(rawURL, 5*time.Second)
		if lk != nil {
			log.Printf("%s resolved %s", client.RemoteAddr(), lk)
		}
		if err != nil {
			log.Println("dial:", err)
			io.WriteString(client, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
			return
		}
		defer upstream.Close()
		log.Printf("%s -> proxy -> %s (CONNECT tunnel)", client.RemoteAddr(), upstream.RemoteAddr())

		io.WriteString(client, "HTTP/1.1 200 Connection Established\r\n\r\n")
		// Read the client through br, not client: bufio may already hold the start of the TLS handshake.
		relay.Pipe(client, br, upstream)
		log.Printf("%s closed", client.RemoteAddr())
		return
	}

	// Plain HTTP: the full URL in the request line tells us where to go.
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		log.Println("not a proxy request:", rawURL)
		return
	}

	// The destination comes from the URL; port 80 is the default for http://.
	// Theory: _networking-lessons/3-http-tls-proxies.md#url-breakdown
	target := u.Host
	if u.Port() == "" {
		target = net.JoinHostPort(u.Hostname(), "80")
	}

	// The website expects the short form "GET /path HTTP/1.1", so rewrite the request line.
	head[0] = method + " " + u.RequestURI() + " " + version + "\r\n"

	// Client toward the target: a second, separate TCP connection (its own handshake, its own 4-tuple).
	// Theory: _networking-lessons/2-transport-networking.md#ports-and-connection-identity
	upstream, lk, err := resolve.Dial(target, 5*time.Second)
	if lk != nil {
		log.Printf("%s resolved %s", client.RemoteAddr(), lk)
	}
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

	// Read the client through br, not client: bufio may already hold bytes after the head (a body).
	relay.Pipe(client, br, upstream)
	log.Printf("%s closed", client.RemoteAddr())
}

// readHead reads the request line and the headers, up to the empty line that ends them.
// HTTP framing: every line ends with \r\n, and an empty line marks the end of the head.
//
// Theory: _networking-lessons/3-http-tls-proxies.md#http-framing
// Theory: _networking-lessons/2-transport-networking.md#tcp-is-a-byte-stream
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
