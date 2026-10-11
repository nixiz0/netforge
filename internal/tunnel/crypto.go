package tunnel

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"
)

// In VPN mode every tunnel packet is sealed with an AEAD (AES-256-GCM). The 4-byte header stays
// readable but is authenticated; the payload is encrypted and authenticated:
//
//	VER TYPE LENGTH | NONCE (12) | CIPHERTEXT (same size as the inner packet) | TAG (16)
//	visible, authenticated       encrypted + authenticated
//
// Theory: _networking-lessons/8-security-cryptography.md#aead
// Theory: _networking-lessons/8-security-cryptography.md#associated-data
// Theory: _networking-lessons/8-security-cryptography.md#vpn-packet-security
const (
	keySize      = 32 // AES-256
	nonceSize    = 12
	tagSize      = 16
	sealOverhead = nonceSize + tagSize      // 28 bytes added by encryption
	vpnMTU       = tunnelMTU - sealOverhead // 1440: the largest inner packet in VPN mode
)

// errAuth: the tag does not match. Wrong keys, or a modified packet.
var errAuth = errors.New("authentication failed: wrong key or modified packet")

// sealer encrypts and decrypts tunnel packets: one key to seal what we send, another to open what
// we receive. Both come from a handshake (handshake.go).
type sealer struct {
	send, recv cipher.AEAD
	// A nonce must never repeat with the same key. Each one is a random 4-byte prefix, drawn once
	// at start, plus a counter: unique within a run, and a restart (counter back to 0) gets a new prefix.
	// Theory: _networking-lessons/8-security-cryptography.md#nonces
	prefix  [4]byte
	counter atomic.Uint64 // shared by the send loop and the keepalive goroutine

	window replayWindow // counters already accepted from the peer; only the receive loop touches it
}

func newSealer(sendKey, recvKey []byte) (*sealer, error) {
	send, err := newAEAD(sendKey)
	if err != nil {
		return nil, err
	}
	recv, err := newAEAD(recvKey)
	if err != nil {
		return nil, err
	}
	s := &sealer{send: send, recv: recv}
	rand.Read(s.prefix[:]) // crypto/rand never fails: on a broken system it crashes the program instead
	return s, nil
}

// newAEAD: symmetric cryptography, the key on our side seals what the same key on the other side opens.
// Theory: _networking-lessons/8-security-cryptography.md#symmetric-cryptography
func newAEAD(key []byte) (cipher.AEAD, error) {
	if len(key) != keySize {
		return nil, fmt.Errorf("key is %d bytes, want %d", len(key), keySize)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// seal builds a complete tunnel packet: header, fresh nonce, then ciphertext + tag.
func (s *sealer) seal(typ byte, plaintext []byte) []byte {
	var header [headerSize]byte
	header[0], header[1] = version1, typ
	binary.BigEndian.PutUint16(header[2:], uint16(sealOverhead+len(plaintext)))

	var nonce [nonceSize]byte
	copy(nonce[:], s.prefix[:])
	binary.BigEndian.PutUint64(nonce[4:], s.counter.Add(1))

	b := make([]byte, 0, headerSize+sealOverhead+len(plaintext))
	b = append(b, header[:]...)
	b = append(b, nonce[:]...)
	// The header is "associated data": not encrypted, but changing one bit of it breaks the tag.
	return s.send.Seal(b, nonce[:], plaintext, header[:])
}

// open checks and decrypts a sealed tunnel packet. Any change to the header, nonce, ciphertext or
// tag, or a packet sealed with another key, fails: the packet did not come from our peer.
// A valid packet seen before fails too: it is a replay.
// Theory: _networking-lessons/8-security-cryptography.md#integrity
// Theory: _networking-lessons/8-security-cryptography.md#authentication
func (s *sealer) open(b []byte) (packet, error) {
	pkt, err := decode(b)
	if err != nil {
		return packet{}, err
	}
	if len(pkt.payload) < sealOverhead {
		return packet{}, fmt.Errorf("sealed payload too short: %d bytes", len(pkt.payload))
	}
	nonce, sealed := pkt.payload[:nonceSize], pkt.payload[nonceSize:]
	plaintext, err := s.recv.Open(nil, nonce, sealed, b[:headerSize])
	if err != nil {
		return packet{}, errAuth
	}
	// Authentic, but is it new? Checked after the tag, so only the real peer moves the window.
	prefix, counter := [4]byte(nonce[:4]), binary.BigEndian.Uint64(nonce[4:])
	if !s.window.accept(prefix, counter) {
		return packet{}, fmt.Errorf("replayed packet (counter %d already seen or too old)", counter)
	}
	return packet{typ: pkt.typ, payload: plaintext}, nil
}
