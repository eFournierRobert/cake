package encryptor

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testKey = "1234567890123456"

func TestEncryptor_NewMissingSecret(t *testing.T) {
	t.Setenv("ENCRYPTION_SECRET", "")
	_, err := New()
	assert.Error(t, err)
}

func TestEncryptor_NewInvalidKeyLength(t *testing.T) {
	// 15 bytes: not a valid AES key size (16/24/32).
	t.Setenv("ENCRYPTION_SECRET", "123456789012345")
	_, err := New()
	assert.Error(t, err)
}

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	t.Setenv("ENCRYPTION_SECRET", testKey)
	e, err := New()
	require.NoError(t, err)
	plaintext := "hello world"
	ciphertext, err := e.Encrypt(plaintext)
	require.NoError(t, err)
	decrypted, err := e.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, string(decrypted))
}

func TestEncryptor_EncryptDecryptEmptyString(t *testing.T) {
	t.Setenv("ENCRYPTION_SECRET", testKey)
	e, err := New()
	require.NoError(t, err)
	ciphertext, err := e.Encrypt("")
	require.NoError(t, err)
	decrypted, err := e.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, "", string(decrypted))
}

func TestEncryptor_EncryptDecryptUnicode(t *testing.T) {
	t.Setenv("ENCRYPTION_SECRET", testKey)
	e, err := New()
	require.NoError(t, err)
	plaintext := "héllo wörld — naïve café ÆØ 🧁"
	ciphertext, err := e.Encrypt(plaintext)
	require.NoError(t, err)
	decrypted, err := e.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, string(decrypted))
}

func TestEncryptor_DecryptWithWrongKey(t *testing.T) {
	t.Setenv("ENCRYPTION_SECRET", testKey)
	e, err := New()
	require.NoError(t, err)
	ciphertext, err := e.Encrypt("secret sauce")
	require.NoError(t, err)

	t.Setenv("ENCRYPTION_SECRET", "6543210987654321")
	eWrong, err := New()
	require.NoError(t, err)
	_, err = eWrong.Decrypt(ciphertext)
	assert.Error(t, err)
}

func TestEncryptor_DecryptTampered(t *testing.T) {
	t.Setenv("ENCRYPTION_SECRET", testKey)
	e, err := New()
	require.NoError(t, err)
	plaintext := "tamper test"
	ciphertext, err := e.Encrypt(plaintext)
	require.NoError(t, err)
	// Tamper with the ciphertext
	ciphertext[len(ciphertext)-1] ^= 0xFF
	_, err = e.Decrypt(ciphertext)
	assert.Error(t, err)
}

func TestEncryptor_DecryptShortCiphertext(t *testing.T) {
	t.Setenv("ENCRYPTION_SECRET", testKey)
	e, err := New()
	require.NoError(t, err)
	nonceSize := e.gcm.NonceSize()
	short := make([]byte, nonceSize-1)
	_, err = e.Decrypt(short)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func TestEncryptor_DecryptNonceOnlyCiphertext(t *testing.T) {
	// Exactly nonceSize bytes: passes the length check, but GCM
	// authentication must still reject it.
	t.Setenv("ENCRYPTION_SECRET", testKey)
	e, err := New()
	require.NoError(t, err)
	nonceOnly := make([]byte, e.gcm.NonceSize())
	_, err = e.Decrypt(nonceOnly)
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "ciphertext too short")
}

func TestEncryptor_EncryptSamePlaintextDifferentCiphertext(t *testing.T) {
	t.Setenv("ENCRYPTION_SECRET", testKey)
	e, err := New()
	require.NoError(t, err)
	plaintext := "same plaintext"
	c1, err := e.Encrypt(plaintext)
	require.NoError(t, err)
	c2, err := e.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, string(c1), string(c2))
}

func TestEncryptor_ConcurrentEncryptDecrypt(t *testing.T) {
	// The Encryptor is a shared singleton (constructed once in cmd and held
	// by every service), so it must be safe under concurrent use. Run this
	// with -race to catch any data race.
	t.Setenv("ENCRYPTION_SECRET", testKey)
	e, err := New()
	require.NoError(t, err)

	const workers = 16
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				plaintext := fmt.Sprintf("worker-%d-iteration-%d", id, i)
				ciphertext, err := e.Encrypt(plaintext)
				if err != nil {
					t.Error(err)
					return
				}
				decrypted, err := e.Decrypt(ciphertext)
				if err != nil {
					t.Error(err)
					return
				}
				if string(decrypted) != plaintext {
					t.Errorf("round-trip mismatch: got %q, want %q", decrypted, plaintext)
					return
				}
			}
		}(w)
	}
	wg.Wait()
}
