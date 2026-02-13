package utils

import (
	"encoding/base64"
	"strings"
)

// EncodeBase64 returns standard base64 encoding of data.
func EncodeBase64(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

// DecodeBase64 decodes standard base64 data.
func DecodeBase64(data string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(data)
	return string(decoded), err
}

// EncodeBase64URL returns URL-safe base64 encoding (no padding issues in URLs).
func EncodeBase64URL(data string) string {
	return base64.URLEncoding.EncodeToString([]byte(data))
}

// DecodeBase64URL decodes URL-safe base64 data.
func DecodeBase64URL(data string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(data)
	return string(decoded), err
}

// EncodeBase64URLSafe returns URL-safe base64 without trailing padding.
func EncodeBase64URLSafe(data string) string {
	encoded := base64.URLEncoding.EncodeToString([]byte(data))
	return strings.TrimRight(encoded, "=")
}

// DecodeBase64URLSafe decodes URL-safe base64 (adds padding if needed).
func DecodeBase64URLSafe(data string) (string, error) {
	switch len(data) % 4 {
	case 2:
		data += "=="
	case 3:
		data += "="
	}
	decoded, err := base64.URLEncoding.DecodeString(data)
	return string(decoded), err
}
