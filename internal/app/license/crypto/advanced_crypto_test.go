package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateKeyPairSignVerifyAndActivationRoundTrip(t *testing.T) {
	encKey := sha256.Sum256([]byte("license-secret"))
	pub, encPriv, err := GenerateKeyPair(encKey[:])
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	if pub == "" || encPriv == "" {
		t.Fatal("expected generated key pair to be non-empty")
	}
	if _, err := base64.StdEncoding.DecodeString(pub); err != nil {
		t.Fatalf("public key is not valid base64: %v", err)
	}

	payload := map[string]any{
		"pid": "metis-ent",
		"lic": "LS-001",
		"kv":  1,
	}
	sig, err := SignLicense(payload, encPriv, encKey[:])
	if err != nil {
		t.Fatalf("SignLicense: %v", err)
	}
	valid, err := VerifyLicenseSignature(payload, sig, pub)
	if err != nil {
		t.Fatalf("VerifyLicenseSignature: %v", err)
	}
	if !valid {
		t.Fatal("expected signature to verify")
	}

	code, err := GenerateActivationCode(payload, sig)
	if err != nil {
		t.Fatalf("GenerateActivationCode: %v", err)
	}
	decoded, err := DecodeActivationCode(code)
	if err != nil {
		t.Fatalf("DecodeActivationCode: %v", err)
	}
	if decoded["pid"] != "metis-ent" || decoded["sig"] != sig {
		t.Fatalf("decoded activation payload = %+v", decoded)
	}
}

func TestGetEncryptionKeyWithFallbackPrefersLicenseSecretAndFallsBackToJWT(t *testing.T) {
	licenseSecretKey, err := GetEncryptionKeyWithFallback([]byte("license"), []byte("jwt"))
	if err != nil {
		t.Fatalf("GetEncryptionKeyWithFallback with license secret: %v", err)
	}
	jwtKey, err := GetEncryptionKey([]byte("jwt"))
	if err != nil {
		t.Fatalf("GetEncryptionKey: %v", err)
	}
	if string(licenseSecretKey) == string(jwtKey) {
		t.Fatal("expected license secret to override jwt secret")
	}

	fallbackKey, err := GetEncryptionKeyWithFallback(nil, []byte("jwt"))
	if err != nil {
		t.Fatalf("fallback to jwt secret: %v", err)
	}
	if string(fallbackKey) != string(jwtKey) {
		t.Fatal("expected fallback key to match jwt key")
	}

	if _, err := GetEncryptionKeyWithFallback(nil, nil); err != ErrNoEncryptionKey {
		t.Fatalf("nil secrets error = %v, want %v", err, ErrNoEncryptionKey)
	}
}

func TestGenerateLicenseKeyAndDualKeyDerivation(t *testing.T) {
	licenseKey, err := GenerateLicenseKey()
	if err != nil {
		t.Fatalf("GenerateLicenseKey: %v", err)
	}
	if len(licenseKey) == 0 {
		t.Fatal("expected generated license key to be non-empty")
	}
	if _, err := base64.RawURLEncoding.DecodeString(licenseKey); err != nil {
		t.Fatalf("generated license key is not valid base64url: %v", err)
	}

	v1Key := DeriveLicenseFileKey("RG-001", "METIS1")
	v2Key := DeriveLicenseFileKeyV2("RG-001", "METIS1", licenseKey)
	if string(v1Key) == string(v2Key) {
		t.Fatal("expected v2 key derivation to incorporate license key")
	}

	payload := []byte(`{"product":"metis","seats":20}`)
	encrypted, err := EncryptLicenseFileV2(payload, "RG-001", "Metis Enterprise", licenseKey)
	if err != nil {
		t.Fatalf("EncryptLicenseFileV2: %v", err)
	}
	decrypted, err := DecryptLicenseFileV2(encrypted, "RG-001", licenseKey)
	if err != nil {
		t.Fatalf("DecryptLicenseFileV2: %v", err)
	}
	if string(decrypted) != string(payload) {
		t.Fatalf("decrypted payload = %s, want %s", decrypted, payload)
	}
}

func TestLicenseCrypto_NormalizationCanonicalizationAndVerificationGuards(t *testing.T) {
	t.Run("normalize product names into license file tokens", func(t *testing.T) {
		if got := normalizeLicenseFileToken(" Metis Enterprise "); got != "METISENTERPRISE1" {
			t.Fatalf("normalizeLicenseFileToken returned %q", got)
		}
		if got := normalizeLicenseFileToken("产品%%%"); !strings.HasPrefix(got, "LIC") || !strings.HasSuffix(got, "1") {
			t.Fatalf("expected hashed fallback token, got %q", got)
		}
		if got := normalizeLicenseFileToken("   "); got != "LICENSE1" {
			t.Fatalf("empty product token = %q, want LICENSE1", got)
		}
	})

	t.Run("canonicalization is stable regardless of map order", func(t *testing.T) {
		first := map[string]any{
			"b": []any{2, map[string]any{"y": 2, "x": 1}},
			"a": "alpha",
		}
		second := map[string]any{
			"a": "alpha",
			"b": []any{2, map[string]any{"x": 1, "y": 2}},
		}
		left, err := Canonicalize(first)
		if err != nil {
			t.Fatalf("Canonicalize first: %v", err)
		}
		right, err := Canonicalize(second)
		if err != nil {
			t.Fatalf("Canonicalize second: %v", err)
		}
		if left != right {
			t.Fatalf("expected canonical forms to match, got %q vs %q", left, right)
		}
	})

	t.Run("signature verification rejects tampered payload and broken keys", func(t *testing.T) {
		encKey := sha256.Sum256([]byte("verify-secret"))
		pub, encPriv, err := GenerateKeyPair(encKey[:])
		if err != nil {
			t.Fatalf("GenerateKeyPair: %v", err)
		}
		payload := map[string]any{"pid": "metis", "seats": 20}
		sig, err := SignLicense(payload, encPriv, encKey[:])
		if err != nil {
			t.Fatalf("SignLicense: %v", err)
		}
		tampered := map[string]any{"pid": "metis", "seats": 30}
		valid, err := VerifyLicenseSignature(tampered, sig, pub)
		if err != nil {
			t.Fatalf("VerifyLicenseSignature tampered: %v", err)
		}
		if valid {
			t.Fatal("expected tampered payload verification to fail")
		}
		if _, err := VerifyLicenseSignature(payload, sig, "not-base64"); err == nil {
			t.Fatal("expected invalid public key to be rejected")
		}
		if _, err := SignLicense(payload, encPriv, []byte("short")); err == nil {
			t.Fatal("expected invalid AES key length to be rejected when decrypting private key")
		}
	})
}

func TestLicenseCrypto_DecryptAndDecodeRejectCorruption(t *testing.T) {
	t.Run("decrypt license file rejects malformed envelopes and wrong keys", func(t *testing.T) {
		if _, err := DecryptLicenseFileV2("", "RG-001", ""); err == nil {
			t.Fatal("expected empty license file to be rejected")
		}
		if _, err := DecryptLicenseFileV2("missingdot", "RG-001", ""); err == nil {
			t.Fatal("expected invalid envelope format to be rejected")
		}

		payload := []byte(`{"product":"metis"}`)
		encrypted, err := EncryptLicenseFileV2(payload, "RG-001", "Metis Enterprise", "license-key")
		if err != nil {
			t.Fatalf("EncryptLicenseFileV2: %v", err)
		}
		if _, err := DecryptLicenseFileV2(encrypted, "RG-999", "license-key"); err == nil {
			t.Fatal("expected wrong registration code to fail decryption")
		}
		if _, err := DecryptLicenseFileV2(encrypted, "RG-001", "wrong-license-key"); err == nil {
			t.Fatal("expected wrong license key to fail decryption")
		}
	})

	t.Run("activation code rejects invalid payloads", func(t *testing.T) {
		if _, err := DecodeActivationCode("%%%"); err == nil {
			t.Fatal("expected invalid activation code encoding to fail")
		}
		garbage := base64.RawURLEncoding.EncodeToString([]byte("not-json"))
		if _, err := DecodeActivationCode(garbage); err == nil {
			t.Fatal("expected non-json activation payload to fail")
		}
	})
}

func TestLicenseCrypto_AESAndCanonicalizeHelperContracts(t *testing.T) {
	t.Run("aes gcm round-trip rejects short keys and malformed ciphertext", func(t *testing.T) {
		key := sha256.Sum256([]byte("aes-helper-secret"))
		ciphertext, err := encryptAESGCM([]byte("secret payload"), key[:])
		if err != nil {
			t.Fatalf("encryptAESGCM: %v", err)
		}
		plaintext, err := decryptAESGCM(ciphertext, key[:])
		if err != nil {
			t.Fatalf("decryptAESGCM: %v", err)
		}
		if string(plaintext) != "secret payload" {
			t.Fatalf("decryptAESGCM plaintext = %q", plaintext)
		}

		if _, err := encryptAESGCM([]byte("payload"), []byte("short")); err == nil {
			t.Fatal("expected short AES key to be rejected")
		}
		if _, err := decryptAESGCM([]byte("tiny"), key[:]); err == nil {
			t.Fatal("expected ciphertext shorter than nonce size to be rejected")
		}

		corrupted := append([]byte(nil), ciphertext...)
		corrupted[len(corrupted)-1] ^= 0xFF
		if _, err := decryptAESGCM(corrupted, key[:]); err == nil {
			t.Fatal("expected tampered ciphertext to be rejected")
		}
	})

	t.Run("canonicalize supports scalars arrays and nested maps deterministically", func(t *testing.T) {
		scalars := []struct {
			input any
			want  string
		}{
			{input: "vpn", want: `"vpn"`},
			{input: 42, want: `42`},
			{input: true, want: `true`},
		}
		for _, tt := range scalars {
			got, err := Canonicalize(tt.input)
			if err != nil {
				t.Fatalf("Canonicalize(%T): %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("Canonicalize(%T) = %s, want %s", tt.input, got, tt.want)
			}
		}

		left := []any{"vpn", map[string]any{"b": 2, "a": 1}}
		right := []any{"vpn", map[string]any{"a": 1, "b": 2}}
		l, err := Canonicalize(left)
		if err != nil {
			t.Fatalf("Canonicalize left array: %v", err)
		}
		r, err := Canonicalize(right)
		if err != nil {
			t.Fatalf("Canonicalize right array: %v", err)
		}
		if l != r {
			t.Fatalf("expected canonical array payloads to match, got %q vs %q", l, r)
		}

		var decoded any
		if err := json.Unmarshal([]byte(l), &decoded); err != nil {
			t.Fatalf("canonicalized output should remain valid json: %v", err)
		}
	})
}
