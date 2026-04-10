package model

import "time"

// TwoFactorSecret stores TOTP secrets and backup codes for 2FA.
type TwoFactorSecret struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	UserID      uint      `json:"userId" gorm:"uniqueIndex;not null"`
	Secret      string    `json:"-" gorm:"size:255;not null"`
	BackupCodes string    `json:"-" gorm:"type:text"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
