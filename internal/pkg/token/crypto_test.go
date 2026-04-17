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
