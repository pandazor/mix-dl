// Package backup implements the portable authenticated backup format.
package backup

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"strings"
	"unicode/utf8"
)

const MaxSize = 32 * 1024 * 1024
const magic = "DLMBK001"
const saltSize = 16
const nonceSize = 12
const headerSize = len(magic) + saltSize + nonceSize
const iterations = 600000

var ErrDecrypt = errors.New("Phrase secrète incorrecte ou sauvegarde altérée")

func ValidatePassphrase(phrase string) error {
	if !utf8.ValidString(phrase) || utf8.RuneCountInString(phrase) < 16 || len(phrase) > 1024 || strings.TrimSpace(phrase) == "" {
		return errors.New("La phrase secrète doit contenir au moins 16 caractères, sans dépasser 1024 octets")
	}
	return nil
}
func aead(phrase string, salt []byte) (cipher.AEAD, error) {
	key, e := pbkdf2.Key(sha256.New, phrase, salt, iterations, 32)
	if e != nil {
		return nil, e
	}
	defer clear(key)
	block, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	return cipher.NewGCM(block)
}
func Encrypt(plain []byte, phrase string) ([]byte, error) {
	if e := ValidatePassphrase(phrase); e != nil {
		return nil, e
	}
	if len(plain) > MaxSize-headerSize-16 {
		return nil, errors.New("Sauvegarde trop volumineuse (32 Mio maximum)")
	}
	header := make([]byte, headerSize)
	copy(header, magic)
	if _, e := rand.Read(header[len(magic):]); e != nil {
		return nil, e
	}
	gcm, e := aead(phrase, header[len(magic):len(magic)+saltSize])
	if e != nil {
		return nil, e
	}
	return gcm.Seal(header, header[len(magic)+saltSize:], plain, header), nil
}
func Decrypt(blob []byte, phrase string) ([]byte, error) {
	if e := ValidatePassphrase(phrase); e != nil {
		return nil, e
	}
	if len(blob) < headerSize+16 || len(blob) > MaxSize || !bytes.Equal(blob[:len(magic)], []byte(magic)) {
		return nil, ErrDecrypt
	}
	header := blob[:headerSize]
	gcm, e := aead(phrase, header[len(magic):len(magic)+saltSize])
	if e != nil {
		return nil, ErrDecrypt
	}
	plain, e := gcm.Open(nil, header[len(magic)+saltSize:], blob[headerSize:], header)
	if e != nil {
		return nil, ErrDecrypt
	}
	return plain, nil
}
