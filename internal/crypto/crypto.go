// Package crypto provides the cryptographic primitives used by the credential
// server: ed25519 keypair generation for nodes and random bearer tokens.
package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Keypair is an ed25519 keypair, base64-encoded for storage and transport.
type Keypair struct {
	PublicKey  string // base64 std-encoded ed25519 public key (32 bytes)
	PrivateKey string // base64 std-encoded ed25519 private key (64 bytes)
}

// GenerateKeypair creates a fresh ed25519 keypair for a node.
func GenerateKeypair() (*Keypair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}
	return &Keypair{
		PublicKey:  base64.StdEncoding.EncodeToString(pub),
		PrivateKey: base64.StdEncoding.EncodeToString(priv),
	}, nil
}

// GenerateToken returns a cryptographically random URL-safe bearer token.
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Verify checks an ed25519 signature against a base64-encoded public key.
func Verify(pubKeyB64 string, message, signature []byte) bool {
	pub, err := base64.StdEncoding.DecodeString(pubKeyB64)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(pub), message, signature)
}
