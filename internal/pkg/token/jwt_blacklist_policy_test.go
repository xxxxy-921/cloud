package token

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndParseAccessToken(t *testing.T) {
	secret := []byte("secret")
	changedAt := time.Now().Add(-time.Minute).UTC()

	tokenString, claims, err := GenerateAccessToken(7, "admin", secret, WithPasswordMeta(&changedAt, true))
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}
	if claims.UserID != 7 || claims.Role != "admin" || claims.PasswordChangedAt != changedAt.Unix() || !claims.ForcePasswordReset {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	parsed, err := ParseToken(tokenString, secret)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}
	if parsed.UserID != 7 || parsed.Role != "admin" || parsed.ID == "" {
		t.Fatalf("unexpected parsed claims: %+v", parsed)
	}
}

func TestGenerateAndParseTwoFactorToken(t *testing.T) {
	secret := []byte("secret")
	tokenString, err := GenerateTwoFactorToken(9, secret)
	if err != nil {
		t.Fatalf("GenerateTwoFactorToken returned error: %v", err)
	}

	parsed, err := ParseToken(tokenString, secret)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}
	if parsed.UserID != 9 || parsed.Purpose != "2fa" {
		t.Fatalf("unexpected 2FA claims: %+v", parsed)
	}
}

func TestParseTokenRejectsWrongSecret(t *testing.T) {
	tokenString, _, err := GenerateAccessToken(1, "user", []byte("secret"))
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}

	if _, err := ParseToken(tokenString, []byte("wrong")); err == nil {
		t.Fatal("expected ParseToken to fail with wrong secret")
	}
}

func TestParseTokenRejectsUnexpectedSigningMethod(t *testing.T) {
	claims := TokenClaims{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}}
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("SignedString returned error: %v", err)
	}

	if _, err := ParseToken(tokenString, []byte("secret")); err == nil {
		t.Fatal("expected ParseToken to reject non-HS256 token")
	}
}

func TestParseTokenRejectsExpiredToken(t *testing.T) {
	claims := TokenClaims{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))}}
	signed := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := signed.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("SignedString returned error: %v", err)
	}

	if _, err := ParseToken(tokenString, []byte("secret")); err == nil {
		t.Fatal("expected ParseToken to reject expired token")
	}
}

func TestGenerateRefreshTokenAndPasswordHelpers(t *testing.T) {
	tokenA, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken returned error: %v", err)
	}
	tokenB, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken returned error: %v", err)
	}
	if tokenA == tokenB {
		t.Fatal("expected refresh tokens to be random")
	}
	decoded, err := base64.URLEncoding.DecodeString(tokenA)
	if err != nil {
		t.Fatalf("expected refresh token to be base64 encoded: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("expected 32 random bytes, got %d", len(decoded))
	}

	hash, err := HashPassword("Password123!")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if !CheckPassword(hash, "Password123!") {
		t.Fatal("expected password check to succeed")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("expected password check to fail for wrong secret")
	}
}

func TestTokenBlacklistLifecycle(t *testing.T) {
	blacklist := NewBlacklist()
	if blacklist.Count() != 0 {
		t.Fatalf("expected empty blacklist, got %d", blacklist.Count())
	}

	if blacklist.IsBlocked("") {
		t.Fatal("expected blank token to never be blocked")
	}

	blacklist.Add("", time.Now().Add(time.Minute))
	if blacklist.Count() != 0 {
		t.Fatalf("expected empty blacklist after blank add, got %d", blacklist.Count())
	}

	blacklist.Add("live", time.Now().Add(time.Minute))
	blacklist.Add("expired", time.Now().Add(-time.Minute))
	if !blacklist.IsBlocked("live") {
		t.Fatal("expected live token to be blocked")
	}
	if blacklist.IsBlocked("expired") {
		t.Fatal("expected expired token to be lazily cleaned")
	}
	if removed := blacklist.Cleanup(); removed != 0 {
		t.Fatalf("expected no remaining expired entries, got %d", removed)
	}
	if blacklist.Count() != 1 {
		t.Fatalf("expected one live blacklist entry, got %d", blacklist.Count())
	}

	blacklist.Add("expired-2", time.Now().Add(-time.Minute))
	if removed := blacklist.Cleanup(); removed != 1 {
		t.Fatalf("expected one expired entry to be removed, got %d", removed)
	}
}

func TestValidatePassword(t *testing.T) {
	policy := PasswordPolicy{
		MinLength:      10,
		RequireUpper:   true,
		RequireLower:   true,
		RequireNumber:  true,
		RequireSpecial: true,
	}

	violations := ValidatePassword("short", policy)
	if len(violations) != 4 {
		t.Fatalf("expected all violations, got %v", violations)
	}

	if got := ValidatePassword("Valid123 !", policy); len(got) != 0 {
		t.Fatalf("expected valid password, got %v", got)
	}
}
