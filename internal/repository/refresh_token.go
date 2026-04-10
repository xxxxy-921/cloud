package repository

import (
	"time"

	"github.com/samber/do/v2"

	"metis/internal/database"
	"metis/internal/model"
)

type RefreshTokenRepo struct {
	db *database.DB
}

func NewRefreshToken(i do.Injector) (*RefreshTokenRepo, error) {
	db := do.MustInvoke[*database.DB](i)
	return &RefreshTokenRepo{db: db}, nil
}

func (r *RefreshTokenRepo) Create(rt *model.RefreshToken) error {
	return r.db.Create(rt).Error
}

// FindValid returns the refresh token if it exists, is not revoked, and not expired.
func (r *RefreshTokenRepo) FindValid(token string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.db.Where("token = ? AND revoked = ? AND expires_at > ?", token, false, time.Now()).
		First(&rt).Error
	return &rt, err
}

// FindByToken returns the refresh token regardless of status (for reuse detection).
func (r *RefreshTokenRepo) FindByToken(token string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.db.Where("token = ?", token).First(&rt).Error
	return &rt, err
}

func (r *RefreshTokenRepo) Revoke(token string) error {
	return r.db.Model(&model.RefreshToken{}).
		Where("token = ?", token).
		Update("revoked", true).Error
}

func (r *RefreshTokenRepo) RevokeAllForUser(userID uint) error {
	return r.db.Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked = ?", userID, false).
		Update("revoked", true).Error
}

// GetActiveByUserID returns all non-revoked, non-expired refresh tokens for a user,
// ordered by LastSeenAt ascending (oldest activity first).
func (r *RefreshTokenRepo) GetActiveByUserID(userID uint) ([]model.RefreshToken, error) {
	var tokens []model.RefreshToken
	err := r.db.Where("user_id = ? AND revoked = ? AND expires_at > ?", userID, false, time.Now()).
		Order("last_seen_at ASC").
		Find(&tokens).Error
	return tokens, err
}

// ActiveSession is a joined view of refresh_token + user for the sessions list API.
type ActiveSession struct {
	ID         uint      `json:"id"`
	UserID     uint      `json:"userId"`
	Username   string    `json:"username"`
	IPAddress  string    `json:"ipAddress"`
	UserAgent  string    `json:"userAgent"`
	LoginAt    time.Time `json:"loginAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	AccessTokenJTI string `json:"accessTokenJti"`
}

// GetActiveSessions returns all active sessions joined with user info, supporting pagination.
func (r *RefreshTokenRepo) GetActiveSessions(page, pageSize int) ([]ActiveSession, int64, error) {
	var total int64
	r.db.Model(&model.RefreshToken{}).
		Where("refresh_tokens.revoked = ? AND refresh_tokens.expires_at > ?", false, time.Now()).
		Count(&total)

	var sessions []ActiveSession
	err := r.db.Model(&model.RefreshToken{}).
		Select("refresh_tokens.id, refresh_tokens.user_id, users.username, refresh_tokens.ip_address, refresh_tokens.user_agent, refresh_tokens.created_at as login_at, refresh_tokens.last_seen_at, refresh_tokens.access_token_jti").
		Joins("JOIN users ON users.id = refresh_tokens.user_id").
		Where("refresh_tokens.revoked = ? AND refresh_tokens.expires_at > ?", false, time.Now()).
		Order("refresh_tokens.last_seen_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&sessions).Error
	return sessions, total, err
}

// GetActiveTokenJTIsByUserID returns all active access token JTIs for a user.
func (r *RefreshTokenRepo) GetActiveTokenJTIsByUserID(userID uint) ([]string, error) {
	var jtis []string
	err := r.db.Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked = ? AND expires_at > ? AND access_token_jti != ''", userID, false, time.Now()).
		Pluck("access_token_jti", &jtis).Error
	return jtis, err
}

// FindByID returns a refresh token by its primary key ID.
func (r *RefreshTokenRepo) FindByID(id uint) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.db.First(&rt, id).Error
	return &rt, err
}

// RevokeByID revokes a refresh token by its primary key ID.
func (r *RefreshTokenRepo) RevokeByID(id uint) error {
	return r.db.Model(&model.RefreshToken{}).
		Where("id = ?", id).
		Update("revoked", true).Error
}

// DeleteExpiredTokens hard-deletes refresh tokens that are expired or revoked for more than the given duration.
func (r *RefreshTokenRepo) DeleteExpiredTokens(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result := r.db.Unscoped().
		Where("(expires_at < ?) OR (revoked = ? AND updated_at < ?)", cutoff, true, cutoff).
		Delete(&model.RefreshToken{})
	return result.RowsAffected, result.Error
}
