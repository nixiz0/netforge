package tunnel

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// Each VPN end has an X25519 key pair. The private key never leaves its machine; only the public
// key is copied to the other side. Key files hold 32 bytes as 64 hex characters.
//
// Theory: _networking-lessons/8-security-cryptography.md#asymmetric-cryptography

// GenerateKey returns a new private key as hex, for `netforge genkey`.
func GenerateKey() string {
	k, _ := ecdh.X25519().GenerateKey(rand.Reader) // only fails if crypto/rand does, which crashes instead
	return hex.EncodeToString(k.Bytes())
}

// PublicKey reads a private key file and returns its public key as hex, for `netforge pubkey`.
func PublicKey(path string) (string, error) {
	k, err := ReadPrivateKey(path)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(k.PublicKey().Bytes()), nil
}

// ReadPrivateKey loads a key file written by `netforge genkey`.
func ReadPrivateKey(path string) (*ecdh.PrivateKey, error) {
	b, err := readKeyFile(path)
	if err != nil {
		return nil, err
	}
	return ecdh.X25519().NewPrivateKey(b)
}

// ReadPublicKey loads a key file written by `netforge pubkey`.
func ReadPublicKey(path string) (*ecdh.PublicKey, error) {
	b, err := readKeyFile(path)
	if err != nil {
		return nil, err
	}
	return ecdh.X25519().NewPublicKey(b)
}

func readKeyFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	b, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(b) != keySize {
		return nil, fmt.Errorf("%s: want %d hex characters", path, 2*keySize)
	}
	return b, nil
}
