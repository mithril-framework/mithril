package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// HashAlgorithm represents the available hashing algorithms
type HashAlgorithm string

const (
	Bcrypt HashAlgorithm = "bcrypt"
	Argon2 HashAlgorithm = "argon2"
	SHA256 HashAlgorithm = "sha256"
	SHA512 HashAlgorithm = "sha512"
	HMAC   HashAlgorithm = "hmac"
)

// HashConfig holds configuration for hashing
type HashConfig struct {
	Algorithm HashAlgorithm `json:"algorithm"`
	Cost      int           `json:"cost"`       // For bcrypt
	Memory    uint32        `json:"memory"`     // For argon2 (in KB)
	Time      uint32        `json:"time"`       // For argon2
	Threads   uint8         `json:"threads"`    // For argon2
	KeyLength uint32        `json:"key_length"` // For argon2
	Secret    string        `json:"secret"`     // For HMAC
}

// DefaultHashConfig returns the default hash configuration
func DefaultHashConfig() *HashConfig {
	return &HashConfig{
		Algorithm: Bcrypt,
		Cost:      12,                   // bcrypt cost
		Memory:    64 * 1024,            // 64MB for argon2
		Time:      3,                    // 3 iterations for argon2
		Threads:   4,                    // 4 threads for argon2
		KeyLength: 32,                   // 32 bytes for argon2
		Secret:    "default-secret-key", // For HMAC
	}
}

// HashResult represents the result of a hashing operation
type HashResult struct {
	Hash      string        `json:"hash"`
	Algorithm HashAlgorithm `json:"algorithm"`
	Salt      string        `json:"salt,omitempty"`
	Config    *HashConfig   `json:"config,omitempty"`
}

// Hash hashes a password using the specified algorithm
func Hash(password string, config *HashConfig) (*HashResult, error) {
	if config == nil {
		config = DefaultHashConfig()
	}

	switch config.Algorithm {
	case Bcrypt:
		return hashBcrypt(password, config)
	case Argon2:
		return hashArgon2(password, config)
	case SHA256:
		return hashSHA256(password)
	case SHA512:
		return hashSHA512(password)
	case HMAC:
		return hashHMAC(password, config)
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", config.Algorithm)
	}
}

// Verify verifies a password against a hash
func Verify(password, hash string, config *HashConfig) (bool, error) {
	if config == nil {
		config = DefaultHashConfig()
	}

	switch config.Algorithm {
	case Bcrypt:
		return verifyBcrypt(password, hash)
	case Argon2:
		return verifyArgon2(password, hash, config)
	case SHA256:
		return verifySHA256(password, hash)
	case SHA512:
		return verifySHA512(password, hash)
	case HMAC:
		return verifyHMACHash(password, hash, config)
	default:
		return false, fmt.Errorf("unsupported hash algorithm: %s", config.Algorithm)
	}
}

// hashBcrypt hashes a password using bcrypt
func hashBcrypt(password string, config *HashConfig) (*HashResult, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), config.Cost)
	if err != nil {
		return nil, err
	}

	return &HashResult{
		Hash:      string(hash),
		Algorithm: Bcrypt,
		Config:    config,
	}, nil
}

// verifyBcrypt verifies a password against a bcrypt hash
func verifyBcrypt(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil, nil
}

// hashArgon2 hashes a password using argon2
func hashArgon2(password string, config *HashConfig) (*HashResult, error) {
	// Generate random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	// Hash the password
	hash := argon2.IDKey([]byte(password), salt, config.Time, config.Memory, config.Threads, config.KeyLength)

	// Encode salt and hash
	saltEncoded := base64.RawStdEncoding.EncodeToString(salt)
	hashEncoded := base64.RawStdEncoding.EncodeToString(hash)

	// Combine salt and hash
	combined := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, config.Memory, config.Time, config.Threads, saltEncoded, hashEncoded)

	return &HashResult{
		Hash:      combined,
		Algorithm: Argon2,
		Salt:      saltEncoded,
		Config:    config,
	}, nil
}

// verifyArgon2 verifies a password against an argon2 hash
func verifyArgon2(password, hash string, config *HashConfig) (bool, error) {
	// Parse the hash to extract salt and parameters
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, fmt.Errorf("invalid argon2 hash format")
	}

	// Decode salt
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	// Decode stored hash
	storedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	// Hash the password with the same parameters
	computedHash := argon2.IDKey([]byte(password), salt, config.Time, config.Memory, config.Threads, config.KeyLength)

	// Compare hashes
	return hmac.Equal(storedHash, computedHash), nil
}

// hashSHA256 hashes a password using SHA-256
func hashSHA256(password string) (*HashResult, error) {
	hash := sha256.Sum256([]byte(password))
	hashStr := hex.EncodeToString(hash[:])

	return &HashResult{
		Hash:      hashStr,
		Algorithm: SHA256,
	}, nil
}

// verifySHA256 verifies a password against a SHA-256 hash
func verifySHA256(password, hash string) (bool, error) {
	computedHash := sha256.Sum256([]byte(password))
	computedHashStr := hex.EncodeToString(computedHash[:])
	return computedHashStr == hash, nil
}

// hashSHA512 hashes a password using SHA-512
func hashSHA512(password string) (*HashResult, error) {
	hash := sha512.Sum512([]byte(password))
	hashStr := hex.EncodeToString(hash[:])

	return &HashResult{
		Hash:      hashStr,
		Algorithm: SHA512,
	}, nil
}

// verifySHA512 verifies a password against a SHA-512 hash
func verifySHA512(password, hash string) (bool, error) {
	computedHash := sha512.Sum512([]byte(password))
	computedHashStr := hex.EncodeToString(computedHash[:])
	return computedHashStr == hash, nil
}

// hashHMAC hashes a password using HMAC-SHA256
func hashHMAC(password string, config *HashConfig) (*HashResult, error) {
	h := hmac.New(sha256.New, []byte(config.Secret))
	h.Write([]byte(password))
	hash := h.Sum(nil)
	hashStr := hex.EncodeToString(hash)

	return &HashResult{
		Hash:      hashStr,
		Algorithm: HMAC,
		Config:    config,
	}, nil
}

// verifyHMACHash verifies a password against an HMAC hash
func verifyHMACHash(password, hash string, config *HashConfig) (bool, error) {
	h := hmac.New(sha256.New, []byte(config.Secret))
	h.Write([]byte(password))
	computedHash := h.Sum(nil)
	computedHashStr := hex.EncodeToString(computedHash)
	return computedHashStr == hash, nil
}

// Base64Encode encodes a string to base64
func Base64Encode(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

// Base64Decode decodes a base64 string
func Base64Decode(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// Base64URLEncode encodes a string to base64 URL-safe encoding
func Base64URLEncode(data string) string {
	return base64.URLEncoding.EncodeToString([]byte(data))
}

// Base64URLDecode decodes a base64 URL-safe string
func Base64URLDecode(encoded string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// GenerateRandomBytes generates random bytes of specified length
func GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) (string, error) {
	bytes, err := GenerateRandomBytes(length)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
