package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	AccessTokenDuration  = 30 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour
)

// TokenClaims holds the JWT claims for access tokens.
type TokenClaims struct {
	UserID             uint   `json:"userId"`
	Role               string `json:"role"`
	PasswordChangedAt  int64  `json:"passwordChangedAt,omitempty"`
	ForcePasswordReset bool   `json:"forcePasswordReset,omitempty"`
	Purpose            string `json:"purpose,omitempty"` // empty for normal tokens, "2fa" for 2FA tokens
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a signed JWT for the given user.
// Returns the signed token string and the claims (for access to jti, expiry, etc.).
func GenerateAccessToken(userID uint, role string, secret []byte, opts ...AccessTokenOption) (string, *TokenClaims, error) {
	now := time.Now()
	claims := TokenClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "metis",
			Subject:   strconv.FormatUint(uint64(userID), 10),
			ID:        uuid.NewString(),
		},
	}
	for _, opt := range opts {
		opt(&claims)
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(secret)
	if err != nil {
		return "", nil, err
	}
	return signed, &claims, nil
}

// AccessTokenOption configures additional claims on an access token.
type AccessTokenOption func(*TokenClaims)

// WithPasswordMeta sets password-related claims.
func WithPasswordMeta(changedAt *time.Time, forceReset bool) AccessTokenOption {
	return func(c *TokenClaims) {
		if changedAt != nil {
			c.PasswordChangedAt = changedAt.Unix()
		}
		c.ForcePasswordReset = forceReset
	}
}

// GenerateTwoFactorToken creates a short-lived JWT for 2FA verification.
func GenerateTwoFactorToken(userID uint, secret []byte) (string, error) {
	now := time.Now()
	claims := TokenClaims{
		UserID:  userID,
		Purpose: "2fa",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "metis",
			Subject:   strconv.FormatUint(uint64(userID), 10),
			ID:        uuid.NewString(),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}

// ParseToken validates and parses a JWT string, returning claims.
func ParseToken(tokenString string, secret []byte) (*TokenClaims, error) {
	claims := &TokenClaims{}
	t, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

// GenerateRefreshToken creates a cryptographically random opaque token.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a hashed password with a plaintext password.
func CheckPassword(hashedPassword, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}
