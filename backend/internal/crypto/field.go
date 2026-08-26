// Package crypto provides application-level protection for sensitive
// fields (currently: NIK, insurance number) that must never sit in plain
// text in the database or in the audit hash chain.
//
// Two separate primitives on purpose:
//   - FieldCipher (this file): reversible AES-256-GCM encryption, used so
//     the app can still show the real value to someone authorized to see
//     it (the patient themselves, or a hospital with an approved request).
//   - KeyedHasher (hash.go): one-way HMAC, used where we only ever need to
//     compare values (uniqueness checks) or commit to a value without
//     revealing it (the audit chain) - it can never be reversed even by us.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// FieldCipher does application-level, at-rest encryption for individual
// sensitive fields using AES-256-GCM. This protects the data even if
// someone gets hold of a raw database dump or backup - they only see
// ciphertext, never the NIK (or whatever else) itself.
type FieldCipher struct {
	gcm cipher.AEAD
}

// NewFieldCipher builds a cipher from a 32-byte (AES-256) key. Generate a
// key with `openssl rand -base64 32` - see config.go / .env.example for how
// it's loaded.
func NewFieldCipher(key []byte) (*FieldCipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("field encryption key must be exactly 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM mode: %w", err)
	}

	return &FieldCipher{gcm: gcm}, nil
}

// Encrypt returns a base64-encoded string (nonce + ciphertext + auth tag)
// that is safe to store directly in a TEXT column. An empty input returns
// an empty string so optional fields (e.g. insurance number) don't need
// special-casing at call sites.
func (c *FieldCipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	// A fresh random nonce every call means encrypting the same NIK twice
	// produces different ciphertext - that's intentional, it stops anyone
	// from spotting "these two rows have the same NIK" just by comparing
	// ciphertext bytes.
	ciphertext := c.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt reverses Encrypt. Returns an error if the ciphertext was
// tampered with or encrypted under a different key (GCM's auth tag check
// catches both).
func (c *FieldCipher) Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}

	nonceSize := c.gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}
