package tunnel

import (
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"
)

// The handshake: two messages that give both ends fresh session keys without ever sending them.
//
//	client → server   INIT  sealed with the static keys: timestamp + client ephemeral public key
//	server → client   RESP  sealed with the static keys: same timestamp + server ephemeral public key
//	both              session keys = HKDF( X25519(own ephemeral private, peer ephemeral public) )
//
// Static keys: X25519(own private, peer public) gives both ends the same secret, and nobody else
// can compute it. Keys derived from it seal the handshake, so only the two key owners can run one.
// Ephemeral keys: new for every handshake and forgotten after it. A key stolen later cannot
// decrypt recorded sessions (forward secrecy), and packets from an older session fail the tag.
//
// Theory: _networking-lessons/8-security-cryptography.md#key-exchange
// Theory: _networking-lessons/8-security-cryptography.md#diffiehellman-concept
// Theory: _networking-lessons/8-security-cryptography.md#session-keys
// Theory: _networking-lessons/8-security-cryptography.md#forward-secrecy
const (
	typeInit = 0x03
	typeResp = 0x04

	handshakeSize = 8 + keySize // payload: timestamp, ephemeral public key
)

// handshake is one end's handshake state.
type handshake struct {
	static   *sealer // seals INIT and RESP with keys derived from the two static key pairs
	isClient bool

	mu  sync.Mutex       // client: written when sending INIT, read when RESP arrives
	eph *ecdh.PrivateKey // client: ephemeral key of the handshake in progress
	ts  uint64           // client: its timestamp

	lastTS uint64 // server: newest INIT accepted; an older one is a replay
}

func newHandshake(priv *ecdh.PrivateKey, peer *ecdh.PublicKey, isClient bool) (*handshake, error) {
	shared, err := priv.ECDH(peer)
	if err != nil {
		return nil, err
	}
	static, err := deriveSealer(shared, nil, "netforge static", isClient)
	if err != nil {
		return nil, err
	}
	return &handshake{static: static, isClient: isClient}, nil
}

// deriveSealer turns a shared secret into two keys, one per direction, with HKDF.
// Two keys mean the two ends never seal with the same key, so their nonces can never collide.
func deriveSealer(secret, salt []byte, info string, isClient bool) (*sealer, error) {
	okm, err := hkdf.Key(sha256.New, secret, salt, info, 2*keySize)
	if err != nil {
		return nil, err
	}
	c2s, s2c := okm[:keySize], okm[keySize:]
	if isClient {
		return newSealer(c2s, s2c)
	}
	return newSealer(s2c, c2s)
}

// start (client) creates a fresh ephemeral key and returns the INIT packet that carries it.
func (h *handshake) start() []byte {
	eph, _ := ecdh.X25519().GenerateKey(rand.Reader)
	h.mu.Lock()
	// Strictly increasing: two handshakes in the same clock tick (coarse on some systems) must
	// still differ, or the server would take the second one for a replay.
	ts := max(uint64(time.Now().UnixNano()), h.ts+1)
	h.eph, h.ts = eph, ts
	h.mu.Unlock()
	return h.static.seal(typeInit, handshakePayload(ts, eph.PublicKey()))
}

// respond (server) handles an authentic INIT: it returns the RESP packet and the new session.
func (h *handshake) respond(init packet) ([]byte, *sealer, error) {
	ts, clientEph, err := parseHandshake(init.payload)
	if err != nil {
		return nil, nil, err
	}
	// The static sealer's replay window cannot stop an INIT recorded from an earlier client run
	// (another nonce prefix). The timestamp can: it must be newer than the last one accepted.
	if ts <= h.lastTS {
		return nil, nil, errors.New("handshake not newer than the last one: replayed")
	}
	eph, _ := ecdh.X25519().GenerateKey(rand.Reader)
	session, err := newSession(eph, clientEph, clientEph.Bytes(), eph.PublicKey().Bytes(), false)
	if err != nil {
		return nil, nil, err
	}
	h.lastTS = ts
	return h.static.seal(typeResp, handshakePayload(ts, eph.PublicKey())), session, nil
}

// finish (client) handles an authentic RESP to the handshake in progress, and returns the session.
func (h *handshake) finish(resp packet) (*sealer, error) {
	ts, serverEph, err := parseHandshake(resp.payload)
	if err != nil {
		return nil, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.eph == nil || ts != h.ts {
		return nil, errors.New("response to an older handshake")
	}
	session, err := newSession(h.eph, serverEph, h.eph.PublicKey().Bytes(), serverEph.Bytes(), true)
	h.eph = nil // forget the ephemeral private key: forward secrecy
	return session, err
}

// newSession derives the session keys from the ephemeral exchange. Both public keys go into the
// HKDF salt, so the keys belong to exactly this handshake.
func newSession(eph *ecdh.PrivateKey, peerEph *ecdh.PublicKey, clientPub, serverPub []byte, isClient bool) (*sealer, error) {
	shared, err := eph.ECDH(peerEph)
	if err != nil {
		return nil, err
	}
	return deriveSealer(shared, slices.Concat(clientPub, serverPub), "netforge session", isClient)
}

func handshakePayload(ts uint64, pub *ecdh.PublicKey) []byte {
	return append(binary.BigEndian.AppendUint64(nil, ts), pub.Bytes()...)
}

func parseHandshake(b []byte) (uint64, *ecdh.PublicKey, error) {
	if len(b) != handshakeSize {
		return 0, nil, fmt.Errorf("handshake payload is %d bytes, want %d", len(b), handshakeSize)
	}
	pub, err := ecdh.X25519().NewPublicKey(b[8:])
	return binary.BigEndian.Uint64(b), pub, err
}
