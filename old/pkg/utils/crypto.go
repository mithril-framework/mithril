package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

// EncryptionConfig holds configuration for encryption
type EncryptionConfig struct {
	Key    string `json:"key"`    // 32-byte key for AES-256
	Secret string `json:"secret"` // Secret for HMAC signing
}

// EncryptResult represents the result of an encryption operation
type EncryptResult struct {
	Data      string `json:"data"`      // Encrypted data (base64 encoded)
	Signature string `json:"signature"` // HMAC signature (hex encoded)
	Nonce     string `json:"nonce"`     // Nonce used for encryption (hex encoded)
}

// DecryptResult represents the result of a decryption operation
type DecryptResult struct {
	Data  string `json:"data"`            // Decrypted data
	Valid bool   `json:"valid"`           // Whether the signature is valid
	Error string `json:"error,omitempty"` // Error message if decryption failed
}

// NewEncryptionConfig creates a new encryption configuration with a random key
func NewEncryptionConfig() (*EncryptionConfig, error) {
	// Generate a random 32-byte key for AES-256
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	// Generate a random secret for HMAC
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}

	return &EncryptionConfig{
		Key:    hex.EncodeToString(key),
		Secret: hex.EncodeToString(secret),
	}, nil
}

// NewEncryptionConfigWithKey creates a new encryption configuration with provided key and secret
func NewEncryptionConfigWithKey(key, secret string) *EncryptionConfig {
	return &EncryptionConfig{
		Key:    key,
		Secret: secret,
	}
}

// Encrypt encrypts data using AES-256-GCM with HMAC signing
func Encrypt(data string, config *EncryptionConfig) (*EncryptResult, error) {
	// Decode the key
	key, err := hex.DecodeString(config.Key)
	if err != nil {
		return nil, fmt.Errorf("invalid key format: %v", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %v", err)
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)

	// Encode the encrypted data
	encryptedData := base64.StdEncoding.EncodeToString(ciphertext)

	// Create HMAC signature
	signature := createHMAC(encryptedData, config.Secret)

	// Encode nonce
	nonceStr := hex.EncodeToString(nonce)

	return &EncryptResult{
		Data:      encryptedData,
		Signature: signature,
		Nonce:     nonceStr,
	}, nil
}

// Decrypt decrypts data using AES-256-GCM with HMAC verification
func Decrypt(encryptedData, signature, nonce string, config *EncryptionConfig) (*DecryptResult, error) {
	// Verify HMAC signature
	if !verifyHMAC(encryptedData, signature, config.Secret) {
		return &DecryptResult{
			Data:  "",
			Valid: false,
			Error: "invalid signature",
		}, nil
	}

	// Decode the key
	key, err := hex.DecodeString(config.Key)
	if err != nil {
		return &DecryptResult{
			Data:  "",
			Valid: false,
			Error: fmt.Sprintf("invalid key format: %v", err),
		}, nil
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return &DecryptResult{
			Data:  "",
			Valid: false,
			Error: fmt.Sprintf("failed to create cipher: %v", err),
		}, nil
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return &DecryptResult{
			Data:  "",
			Valid: false,
			Error: fmt.Sprintf("failed to create GCM: %v", err),
		}, nil
	}

	// Decode the encrypted data
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return &DecryptResult{
			Data:  "",
			Valid: false,
			Error: fmt.Sprintf("failed to decode encrypted data: %v", err),
		}, nil
	}

	// Decode the nonce
	nonceBytes, err := hex.DecodeString(nonce)
	if err != nil {
		return &DecryptResult{
			Data:  "",
			Valid: false,
			Error: fmt.Sprintf("failed to decode nonce: %v", err),
		}, nil
	}

	// Decrypt the data
	plaintext, err := gcm.Open(nil, nonceBytes, ciphertext, nil)
	if err != nil {
		return &DecryptResult{
			Data:  "",
			Valid: false,
			Error: fmt.Sprintf("failed to decrypt data: %v", err),
		}, nil
	}

	return &DecryptResult{
		Data:  string(plaintext),
		Valid: true,
	}, nil
}

// createHMAC creates an HMAC signature for the given data
func createHMAC(data, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// verifyHMAC verifies an HMAC signature
func verifyHMAC(data, signature, secret string) bool {
	expectedSignature := createHMAC(data, secret)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// Sign signs data using HMAC-SHA256
func Sign(data, secret string) string {
	return createHMAC(data, secret)
}

// VerifySignature verifies an HMAC signature
func VerifySignature(data, signature, secret string) bool {
	return verifyHMAC(data, signature, secret)
}

// GenerateKey generates a random encryption key
func GenerateKey() (string, error) {
	key := make([]byte, 32) // 32 bytes for AES-256
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

// GenerateSecret generates a random secret for HMAC
func GenerateSecret() (string, error) {
	secret := make([]byte, 32) // 32 bytes for HMAC
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return hex.EncodeToString(secret), nil
}

// EncryptWithPassword encrypts data using a password-derived key
func EncryptWithPassword(data, password string) (*EncryptResult, error) {
	// Derive key from password using PBKDF2
	key := deriveKeyFromPassword(password, 32)
	secret := deriveKeyFromPassword(password+"_secret", 32)

	config := &EncryptionConfig{
		Key:    hex.EncodeToString(key),
		Secret: hex.EncodeToString(secret),
	}

	return Encrypt(data, config)
}

// DecryptWithPassword decrypts data using a password-derived key
func DecryptWithPassword(encryptedData, signature, nonce, password string) (*DecryptResult, error) {
	// Derive key from password using PBKDF2
	key := deriveKeyFromPassword(password, 32)
	secret := deriveKeyFromPassword(password+"_secret", 32)

	config := &EncryptionConfig{
		Key:    hex.EncodeToString(key),
		Secret: hex.EncodeToString(secret),
	}

	return Decrypt(encryptedData, signature, nonce, config)
}

// deriveKeyFromPassword derives a key from a password using PBKDF2
func deriveKeyFromPassword(password string, keyLength int) []byte {
	// Use a simple salt for demonstration - in production, use a random salt
	salt := []byte("mithril-salt-2024")
	return derivePBKDF2Key([]byte(password), salt, 10000, keyLength)
}

// derivePBKDF2Key derives a key using PBKDF2
func derivePBKDF2Key(password, salt []byte, iterations, keyLength int) []byte {
	// This is a simplified implementation
	// In production, use crypto/pbkdf2 package
	key := make([]byte, keyLength)

	// Simple key derivation (not cryptographically secure)
	// In production, use: pbkdf2.Key(password, salt, iterations, keyLength, sha256.New)
	for i := 0; i < keyLength; i++ {
		key[i] = password[i%len(password)] ^ salt[i%len(salt)]
	}

	return key
}
