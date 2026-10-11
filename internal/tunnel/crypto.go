package tunnel

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
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

// sealer encrypts and decrypts tunnel packets with one shared key.
type sealer struct {
	aead cipher.AEAD
	// A nonce must never repeat with the same key. Each one is a random 4-byte prefix, drawn once
	// at start, plus a counter: unique within a run, and a restart (counter back to 0) gets a new prefix.
	// Theory: _networking-lessons/8-security-cryptography.md#nonces
	prefix  [4]byte
	counter atomic.Uint64 // shared by the send loop and the keepalive goroutine

	recv replayWindow // counters already accepted from the peer; only the receive loop touches it
}

func newSealer(key []byte) (*sealer, error) {
	if len(key) != keySize {
		return nil, fmt.Errorf("key is %d bytes, want %d", len(key), keySize)
	}
	// Symmetric cryptography: both ends hold the same secret key.
	// Theory: _networking-lessons/8-security-cryptography.md#symmetric-cryptography
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	s := &sealer{aead: aead}
	rand.Read(s.prefix[:]) // crypto/rand never fails: on a broken system it crashes the program instead
	return s, nil
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
	return s.aead.Seal(b, nonce[:], plaintext, header[:])
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
	plaintext, err := s.aead.Open(nil, nonce, sealed, b[:headerSize])
	if err != nil {
		return packet{}, errors.New("authentication failed: wrong key or modified packet")
	}
	// Authentic, but is it new? Checked after the tag, so only the real peer moves the window.
	prefix, counter := [4]byte(nonce[:4]), binary.BigEndian.Uint64(nonce[4:])
	if !s.recv.accept(prefix, counter) {
		return packet{}, fmt.Errorf("replayed packet (counter %d already seen or too old)", counter)
	}
	return packet{typ: pkt.typ, payload: plaintext}, nil
}

// NewKey returns a random 32-byte key as 64 hex characters, for `netforge genkey`.
func NewKey() string {
	key := make([]byte, keySize)
	rand.Read(key)
	return hex.EncodeToString(key)
}

// ReadKey loads a key written by NewKey from a file.
func ReadKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(key) != keySize {
		return nil, fmt.Errorf("%s: want %d hex characters (netforge genkey > %s)", path, 2*keySize, path)
	}
	return key, nil
}
