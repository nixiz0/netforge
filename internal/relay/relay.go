package relay

import (
	"io"
	"net"
)

// Pipe copies bytes both ways between client and upstream, and returns once upstream has closed.
//
// fromClient is where the client's bytes are read: the connection itself, or a bufio.Reader
// wrapping it when a protocol header was already parsed and bufio may hold bytes past it.
// The client's FIN is passed on to upstream with a half-close. The caller closes both connections.
//
// Theory: _networking-lessons/3-http-tls-proxies.md#forward-proxy
// Theory: _networking-lessons/2-transport-networking.md#tcp-essentials
func Pipe(client net.Conn, fromClient io.Reader, upstream net.Conn) {
	// Client -> upstream in the background. When the client sends FIN, pass it on.
	go func() {
		io.Copy(upstream, fromClient)
		// Any connection that supports a half-close (TCP does) gets CloseWrite instead of a full Close.
		// Theory: _networking-lessons/2-transport-networking.md#closing-tcp
		if hc, ok := upstream.(interface{ CloseWrite() error }); ok {
			hc.CloseWrite()
		}
	}()

	// Upstream -> client here: the last event is upstream's FIN, so this side waits for it.
	io.Copy(client, upstream)
}
