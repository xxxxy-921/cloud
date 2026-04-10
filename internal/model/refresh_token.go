package model

import "time"

type RefreshToken struct {
	BaseModel
	Token          string    `json:"token" gorm:"uniqueIndex;size:255;not null"`
	UserID         uint      `json:"userId" gorm:"index;not null"`
	ExpiresAt      time.Time `json:"expiresAt" gorm:"not null"`
	Revoked        bool      `json:"revoked" gorm:"not null;default:false"`
	IPAddress      string    `json:"ipAddress" gorm:"size:45"`
	UserAgent      string    `json:"userAgent" gorm:"size:512"`
	LastSeenAt     time.Time `json:"lastSeenAt"`
	AccessTokenJTI string    `json:"accessTokenJti" gorm:"size:36"`
}
