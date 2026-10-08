package encryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"os"
)

type Encryptor struct {
	gcm cipher.AEAD
}

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

func (e *Encryptor) Encrypt(value string) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())

	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return e.gcm.Seal(nonce, nonce, []byte(value), nil), nil
}

func (e *Encryptor) Decrypt(cipherText []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()

	if len(cipherText) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := cipherText[:nonceSize]
	cipherText = cipherText[nonceSize:]

	return e.gcm.Open(nil, nonce, cipherText, nil)
}
