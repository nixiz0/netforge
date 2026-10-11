package tunnel

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"testing"
)

func newKey(t *testing.T) *ecdh.PrivateKey {
	t.Helper()
	k, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// pair returns the two ends of a VPN: each knows its own private key and the other's public key.
func pair(t *testing.T) (client, server *handshake) {
	t.Helper()
	c, s := newKey(t), newKey(t)
	client, err := newHandshake(c, s.PublicKey(), true)
	if err != nil {
		t.Fatal(err)
	}
	server, err = newHandshake(s, c.PublicKey(), false)
	if err != nil {
		t.Fatal(err)
	}
	return client, server
}

// run does one full handshake, as the two goroutines would over UDP.
func run(t *testing.T, client, server *handshake) (clientSession, serverSession *sealer) {
	t.Helper()
	init, err := server.static.open(client.start())
	if err != nil {
		t.Fatal("INIT:", err)
	}
	resp, serverSession, err := server.respond(init)
	if err != nil {
		t.Fatal("respond:", err)
	}
	opened, err := client.static.open(resp)
	if err != nil {
		t.Fatal("RESP:", err)
	}
	clientSession, err = client.finish(opened)
	if err != nil {
		t.Fatal("finish:", err)
	}
	return clientSession, serverSession
}

func TestHandshake(t *testing.T) {
	client, server := pair(t)
	c, s := run(t, client, server)

	// Both ends derived the same keys without sending them: data flows both ways.
	for _, dir := range []struct {
		name     string
		from, to *sealer
	}{{"client to server", c, s}, {"server to client", s, c}} {
		pkt, err := dir.to.open(dir.from.seal(typeData, []byte("ping")))
		if err != nil || !bytes.Equal(pkt.payload, []byte("ping")) {
			t.Fatalf("%s: %v", dir.name, err)
		}
	}

	// A new handshake gives new keys: a packet from the previous session no longer opens.
	old := c.seal(typeData, []byte("recorded"))
	_, s2 := run(t, client, server)
	if _, err := s2.open(old); err == nil {
		t.Fatal("a packet from the previous session was accepted")
	}
}

func TestHandshakeRejects(t *testing.T) {
	t.Run("stranger's key", func(t *testing.T) {
		_, server := pair(t)
		stranger, err := newHandshake(newKey(t), newKey(t).PublicKey(), true)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := server.static.open(stranger.start()); err == nil {
			t.Fatal("INIT from an unknown key accepted")
		}
	})
	t.Run("replayed INIT", func(t *testing.T) {
		client, server := pair(t)
		init, err := server.static.open(client.start())
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := server.respond(init); err != nil {
			t.Fatal(err)
		}
		if _, _, err := server.respond(init); err == nil {
			t.Fatal("the same INIT started a second session")
		}
	})
	t.Run("RESP to an older INIT", func(t *testing.T) {
		client, server := pair(t)
		init, _ := server.static.open(client.start())
		resp, _, _ := server.respond(init)
		client.start() // a newer handshake replaces the one this RESP answers
		opened, err := client.static.open(resp)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.finish(opened); err == nil {
			t.Fatal("RESP to an older INIT accepted")
		}
	})
}
