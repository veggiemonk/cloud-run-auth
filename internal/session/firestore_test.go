package session

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"testing"
	"time"

	"github.com/veggiemonk/cloud-run-auth/internal/is"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	block, err := aes.NewCipher(key)
	is.NoErr(t, err)
	aead, err := cipher.NewGCM(block)
	is.NoErr(t, err)
	return &Store{aead: aead}
}

func TestEncryptDecrypt(t *testing.T) {
	s := newTestStore(t)
	plaintext := []byte("super-secret-token-value")

	encrypted, err := s.encrypt(plaintext)
	is.NoErr(t, err)

	is.True(t, !bytes.Equal(encrypted, plaintext))

	decrypted, err := s.decrypt(encrypted)
	is.NoErr(t, err)

	is.True(t, bytes.Equal(decrypted, plaintext))
}

func TestEncryptDecrypt_DifferentCiphertexts(t *testing.T) {
	s := newTestStore(t)
	plaintext := []byte("same-token")

	enc1, _ := s.encrypt(plaintext)
	enc2, _ := s.encrypt(plaintext)

	is.True(t, !bytes.Equal(enc1, enc2))

	dec1, _ := s.decrypt(enc1)
	dec2, _ := s.decrypt(enc2)
	is.True(t, bytes.Equal(dec1, dec2))
}

func TestDecrypt_TooShort(t *testing.T) {
	s := newTestStore(t)
	_, err := s.decrypt([]byte("short"))
	is.True(t, err != nil)
}

func TestSessionToken(t *testing.T) {
	sess := &Session{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		TokenExpiry:  time.Now().Add(time.Hour),
	}

	tok := sess.Token()
	is.Equal(t, tok.AccessToken, "access-123", "")
	is.Equal(t, tok.RefreshToken, "refresh-456", "")
	is.Equal(t, tok.TokenType, "Bearer", "")
}
