package token

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"gorm.io/gorm"

	"metis/internal/model"
)

var (
	encKey     []byte
	encKeyOnce sync.Once
)

// GetEncryptionKey returns the 32-byte AES-256 key, lazily initialised.
// Priority: ENCRYPTION_KEY env var → SystemConfig row → auto-generate & persist.
func GetEncryptionKey(db *gorm.DB) ([]byte, error) {
	var initErr error
	encKeyOnce.Do(func() {
		// 1. Try env var
		if envKey := os.Getenv("ENCRYPTION_KEY"); envKey != "" {
			decoded, err := hex.DecodeString(envKey)
			if err != nil || len(decoded) != 32 {
				initErr = fmt.Errorf("ENCRYPTION_KEY must be 64 hex chars (32 bytes)")
				return
			}
			encKey = decoded
			return
		}

		// 2. Try SystemConfig
		const cfgKey = "security.encryption_key"
		var cfg model.SystemConfig
		if err := db.Where("`key` = ?", cfgKey).First(&cfg).Error; err == nil {
			decoded, err := hex.DecodeString(cfg.Value)
			if err == nil && len(decoded) == 32 {
				encKey = decoded
				return
			}
		}

		// 3. Auto-generate and persist
		key := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			initErr = fmt.Errorf("generate encryption key: %w", err)
			return
		}
		hexKey := hex.EncodeToString(key)
		db.Save(&model.SystemConfig{Key: cfgKey, Value: hexKey, Remark: "Auto-generated AES-256 encryption key"})
		encKey = key
	})
	return encKey, initErr
}

// Encrypt encrypts plaintext using AES-256-GCM. Returns hex-encoded ciphertext.
func Encrypt(db *gorm.DB, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key, err := GetEncryptionKey(db)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a hex-encoded AES-256-GCM ciphertext. Returns plaintext.
func Decrypt(db *gorm.DB, ciphertextHex string) (string, error) {
	if ciphertextHex == "" {
		return "", nil
	}
	key, err := GetEncryptionKey(db)
	if err != nil {
		return "", err
	}

	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", fmt.Errorf("invalid ciphertext encoding: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt failed: %w", err)
	}
	return string(plaintext), nil
}
