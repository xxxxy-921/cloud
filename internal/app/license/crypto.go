package license

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
)

var (
	ErrNoEncryptionKey = errors.New("neither LICENSE_KEY_SECRET nor JWT_SECRET is set, cannot encrypt private key")
)

// GenerateKeyPair generates a new Ed25519 key pair and returns
// (publicKeyBase64, encryptedPrivateKeyBase64, error).
func GenerateKeyPair(encKey []byte) (string, string, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate ed25519 key: %w", err)
	}

	pubB64 := base64.StdEncoding.EncodeToString(pub)
	privB64 := base64.StdEncoding.EncodeToString(priv)

	encrypted, err := encryptAESGCM([]byte(privB64), encKey)
	if err != nil {
		return "", "", fmt.Errorf("encrypt private key: %w", err)
	}

	return pubB64, base64.StdEncoding.EncodeToString(encrypted), nil
}

// GetEncryptionKey returns the 32-byte AES key.
// Priority: LICENSE_KEY_SECRET env var > SHA-256 of JWT_SECRET env var.
func GetEncryptionKey() ([]byte, error) {
	if secret := os.Getenv("LICENSE_KEY_SECRET"); secret != "" {
		h := sha256.Sum256([]byte(secret))
		return h[:], nil
	}
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		h := sha256.Sum256([]byte(secret))
		return h[:], nil
	}
	return nil, ErrNoEncryptionKey
}

// GetEncryptionKeyWithFallback returns the encryption key, using jwtSecret as fallback.
func GetEncryptionKeyWithFallback(jwtSecret []byte) ([]byte, error) {
	if secret := os.Getenv("LICENSE_KEY_SECRET"); secret != "" {
		h := sha256.Sum256([]byte(secret))
		return h[:], nil
	}
	if len(jwtSecret) > 0 {
		h := sha256.Sum256(jwtSecret)
		return h[:], nil
	}
	return nil, ErrNoEncryptionKey
}

func encryptAESGCM(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

func decryptAESGCM(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return aead.Open(nil, nonce, ct, nil)
}

// Canonicalize produces a deterministic JSON string by recursively sorting object keys.
func Canonicalize(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("canonicalize marshal: %w", err)
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", fmt.Errorf("canonicalize unmarshal: %w", err)
	}
	return canonicalizeValue(raw), nil
}

func canonicalizeValue(v any) string {
	switch val := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		result := "{"
		for i, k := range keys {
			if i > 0 {
				result += ","
			}
			kb, _ := json.Marshal(k)
			result += string(kb) + ":" + canonicalizeValue(val[k])
		}
		return result + "}"
	case []any:
		result := "["
		for i, item := range val {
			if i > 0 {
				result += ","
			}
			result += canonicalizeValue(item)
		}
		return result + "]"
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

// SignLicense signs a canonicalized payload with the decrypted Ed25519 private key.
// Returns a base64url-encoded signature (no padding).
func SignLicense(payload map[string]any, encryptedPrivateKey string, encKey []byte) (string, error) {
	// Decrypt the private key
	encBytes, err := base64.StdEncoding.DecodeString(encryptedPrivateKey)
	if err != nil {
		return "", fmt.Errorf("decode encrypted private key: %w", err)
	}
	privB64Bytes, err := decryptAESGCM(encBytes, encKey)
	if err != nil {
		return "", fmt.Errorf("decrypt private key: %w", err)
	}
	privBytes, err := base64.StdEncoding.DecodeString(string(privB64Bytes))
	if err != nil {
		return "", fmt.Errorf("decode private key base64: %w", err)
	}

	// Canonicalize payload
	canonical, err := Canonicalize(payload)
	if err != nil {
		return "", err
	}

	// Sign
	sig := ed25519.Sign(ed25519.PrivateKey(privBytes), []byte(canonical))
	return base64.RawURLEncoding.EncodeToString(sig), nil
}

// VerifyLicenseSignature verifies an Ed25519 signature against a canonicalized payload.
func VerifyLicenseSignature(payload map[string]any, signatureBase64url string, publicKeyBase64 string) (bool, error) {
	pubBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return false, fmt.Errorf("decode public key: %w", err)
	}
	sigBytes, err := base64.RawURLEncoding.DecodeString(signatureBase64url)
	if err != nil {
		return false, fmt.Errorf("decode signature: %w", err)
	}
	canonical, err := Canonicalize(payload)
	if err != nil {
		return false, err
	}
	return ed25519.Verify(ed25519.PublicKey(pubBytes), []byte(canonical), sigBytes), nil
}

// GenerateActivationCode combines payload + signature into a base64url-encoded JSON string.
func GenerateActivationCode(payload map[string]any, signature string) (string, error) {
	full := make(map[string]any, len(payload)+1)
	for k, v := range payload {
		full[k] = v
	}
	full["sig"] = signature
	data, err := json.Marshal(full)
	if err != nil {
		return "", fmt.Errorf("marshal activation code: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

// DecodeActivationCode decodes a base64url-encoded activation code back to a map.
func DecodeActivationCode(code string) (map[string]any, error) {
	data, err := base64.RawURLEncoding.DecodeString(code)
	if err != nil {
		return nil, fmt.Errorf("decode activation code: %w", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal activation code: %w", err)
	}
	return result, nil
}
