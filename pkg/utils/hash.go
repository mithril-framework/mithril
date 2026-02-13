package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a password using bcrypt (DefaultCost).
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash returns true if password matches the bcrypt hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// HashPasswordArgon2 hashes a password using Argon2id (salt + hash combined, hex-encoded).
func HashPasswordArgon2(password string) (string, error) {
	salt := GenerateRandomBytes(16)
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	combined := make([]byte, 16+32)
	copy(combined[:16], salt)
	copy(combined[16:], hash)
	return hex.EncodeToString(combined), nil
}

// CheckPasswordArgon2 returns true if password matches the Argon2 hash (combined salt+hash hex).
func CheckPasswordArgon2(password, hash string) bool {
	decoded, err := hex.DecodeString(hash)
	if err != nil || len(decoded) != 48 {
		return false
	}
	salt := decoded[:16]
	storedHash := decoded[16:]
	computedHash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return hmac.Equal(storedHash, computedHash)
}

// SHA256 returns the SHA-256 hash of data as a hex string.
func SHA256(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// SHA512 returns the SHA-512 hash of data as a hex string.
func SHA512(data string) string {
	h := sha512.Sum512([]byte(data))
	return hex.EncodeToString(h[:])
}

// HMACSHA256 returns HMAC-SHA256 of data with key as hex string.
func HMACSHA256(data, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// HMACSHA512 returns HMAC-SHA512 of data with key as hex string.
func HMACSHA512(data, key string) string {
	h := hmac.New(sha512.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
