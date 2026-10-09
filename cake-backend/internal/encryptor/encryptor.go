// Package encryptor provides AES-GCM encryption for values stored in
// the database, such as provider api keys.
package encryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"os"
)

// Encryptor encrypts and decrypts values with AES-GCM. It is
// constructed once at startup and shared by the services, and is
// safe for concurrent use.
type Encryptor struct {
	gcm cipher.AEAD
}

// New creates an Encryptor keyed from the ENCRYPTION_SECRET
// environment variable. The secret must be a valid AES key size (16,
// 24 or 32 bytes), otherwise aes.NewCipher rejects it.
func New() (*Encryptor, error) {
	key := []byte([]byte(os.Getenv("ENCRYPTION_SECRET")))

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Encryptor{gcm: gcm}, nil
}

// Encrypt seals value with a fresh random nonce and returns the
// nonce followed by the ciphertext, the blob layout Decrypt expects.
// Encrypting the same value twice produces different outputs.
func (e *Encryptor) Encrypt(value string) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())

	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return e.gcm.Seal(nonce, nonce, []byte(value), nil), nil
}

// Decrypt opens a blob produced by Encrypt: a leading nonce followed
// by the ciphertext. It returns an error when the blob is shorter
// than the nonce, was encrypted with a different key, or fails GCM
// authentication because it was tampered with.
func (e *Encryptor) Decrypt(cipherText []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()

	if len(cipherText) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := cipherText[:nonceSize]
	cipherText = cipherText[nonceSize:]

	return e.gcm.Open(nil, nonce, cipherText, nil)
}
