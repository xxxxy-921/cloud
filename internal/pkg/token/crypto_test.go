package token

import (
	"encoding/hex"
	"fmt"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"metis/internal/model"
)

func resetEncryptionKey() {
	encKeyOnce = sync.Once{}
	encKey = nil
}

func newTestDBForCrypto(t *testing.T) *gorm.DB {
	t.Helper()
	resetEncryptionKey()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(&model.SystemConfig{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return gdb
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	db := newTestDBForCrypto(t)
	plaintext := "my-secret-password"

	ciphertext, err := Encrypt(db, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if ciphertext == "" {
		t.Fatal("expected non-empty ciphertext")
	}
	if ciphertext == plaintext {
		t.Fatal("ciphertext should not equal plaintext")
	}

	decrypted, err := Decrypt(db, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncrypt_AutoGeneratesKey(t *testing.T) {
	db := newTestDBForCrypto(t)

	_, err := Encrypt(db, "anything")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	var cfg model.SystemConfig
	if err := db.Where("\"key\" = ?", "security.encryption_key").First(&cfg).Error; err != nil {
		t.Fatalf("expected encryption key to be generated: %v", err)
	}
	if cfg.Value == "" {
		t.Fatal("expected non-empty encryption key value")
	}
	decoded, err := hex.DecodeString(cfg.Value)
	if err != nil {
		t.Fatalf("expected hex-encoded key: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(decoded))
	}
}

func TestDecrypt_InvalidHex(t *testing.T) {
	db := newTestDBForCrypto(t)
	_, err := Decrypt(db, "not-hex!!!")
	if err == nil {
		t.Fatal("expected error for invalid hex")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	db := newTestDBForCrypto(t)
	ciphertext, err := Encrypt(db, "secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	bytes, err := hex.DecodeString(ciphertext)
	if err != nil {
		t.Fatalf("decode hex: %v", err)
	}
	// Tamper with the last byte
	bytes[len(bytes)-1] ^= 0xFF
	tampered := hex.EncodeToString(bytes)

	_, err = Decrypt(db, tampered)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestGetEncryptionKey_UsesEnvVar(t *testing.T) {
	db := newTestDBForCrypto(t)
	keyHex := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	t.Setenv("ENCRYPTION_KEY", keyHex)

	key, err := GetEncryptionKey(db)
	if err != nil {
		t.Fatalf("GetEncryptionKey: %v", err)
	}
	if hex.EncodeToString(key) != keyHex {
		t.Fatalf("expected env key, got %s", hex.EncodeToString(key))
	}
}

func TestGetEncryptionKey_RejectsInvalidEnvVar(t *testing.T) {
	db := newTestDBForCrypto(t)
	t.Setenv("ENCRYPTION_KEY", "bad-key")

	_, err := GetEncryptionKey(db)
	if err == nil {
		t.Fatal("expected invalid env key error")
	}
}

func TestEncryptDecrypt_PropagateInvalidEnvKey(t *testing.T) {
	t.Run("encrypt", func(t *testing.T) {
		db := newTestDBForCrypto(t)
		t.Setenv("ENCRYPTION_KEY", "bad-key")

		if _, err := Encrypt(db, "secret"); err == nil {
			t.Fatal("expected Encrypt to fail with invalid env key")
		}
	})

	t.Run("decrypt", func(t *testing.T) {
		db := newTestDBForCrypto(t)
		t.Setenv("ENCRYPTION_KEY", "bad-key")

		if _, err := Decrypt(db, "00"); err == nil {
			t.Fatal("expected Decrypt to fail with invalid env key")
		}
	})
}

func TestEncryptDecrypt_EmptyAndShortCiphertext(t *testing.T) {
	db := newTestDBForCrypto(t)

	ciphertext, err := Encrypt(db, "")
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}
	if ciphertext != "" {
		t.Fatalf("expected empty ciphertext, got %q", ciphertext)
	}

	plaintext, err := Decrypt(db, "")
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}
	if plaintext != "" {
		t.Fatalf("expected empty plaintext, got %q", plaintext)
	}

	short := hex.EncodeToString([]byte{1, 2, 3})
	if _, err := Decrypt(db, short); err == nil {
		t.Fatal("expected short ciphertext error")
	}
}

func TestGetEncryptionKey_UsesStoredSystemConfig(t *testing.T) {
	db := newTestDBForCrypto(t)
	keyHex := "ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100"
	if err := db.Create(&model.SystemConfig{
		Key:   "security.encryption_key",
		Value: keyHex,
	}).Error; err != nil {
		t.Fatalf("seed system config: %v", err)
	}

	key, err := GetEncryptionKey(db)
	if err != nil {
		t.Fatalf("GetEncryptionKey: %v", err)
	}
	if got := hex.EncodeToString(key); got != keyHex {
		t.Fatalf("expected stored key %s, got %s", keyHex, got)
	}
}

func TestGetEncryptionKey_InvalidStoredConfigFallsBackToGeneratedKey(t *testing.T) {
	db := newTestDBForCrypto(t)
	if err := db.Create(&model.SystemConfig{
		Key:   "security.encryption_key",
		Value: "not-valid-hex",
	}).Error; err != nil {
		t.Fatalf("seed invalid system config: %v", err)
	}

	key, err := GetEncryptionKey(db)
	if err != nil {
		t.Fatalf("GetEncryptionKey: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected generated 32-byte key, got %d", len(key))
	}

	var cfg model.SystemConfig
	if err := db.Where("\"key\" = ?", "security.encryption_key").First(&cfg).Error; err != nil {
		t.Fatalf("reload generated key: %v", err)
	}
	if _, err := hex.DecodeString(cfg.Value); err != nil {
		t.Fatalf("expected generated key to be valid hex, got %q", cfg.Value)
	}
}
