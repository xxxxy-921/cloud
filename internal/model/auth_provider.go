package model

import "time"

type AuthProvider struct {
	BaseModel
	ProviderKey  string `json:"providerKey" gorm:"uniqueIndex;size:32;not null"`
	DisplayName  string `json:"displayName" gorm:"size:64;not null"`
	Enabled      bool   `json:"enabled" gorm:"not null;default:false"`
	ClientID     string `json:"clientId" gorm:"size:255"`
	ClientSecret string `json:"-" gorm:"size:255"`
	Scopes       string `json:"scopes" gorm:"size:255"`
	CallbackURL  string `json:"callbackUrl" gorm:"size:512"`
	SortOrder    int    `json:"sortOrder" gorm:"not null;default:0"`
}

type AuthProviderResponse struct {
	ID           uint      `json:"id"`
	ProviderKey  string    `json:"providerKey"`
	DisplayName  string    `json:"displayName"`
	Enabled      bool      `json:"enabled"`
	ClientID     string    `json:"clientId"`
	ClientSecret string    `json:"clientSecret"`
	Scopes       string    `json:"scopes"`
	CallbackURL  string    `json:"callbackUrl"`
	SortOrder    int       `json:"sortOrder"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (p *AuthProvider) ToResponse() AuthProviderResponse {
	secret := ""
	if p.ClientSecret != "" {
		secret = "••••••"
	}
	return AuthProviderResponse{
		ID:           p.ID,
		ProviderKey:  p.ProviderKey,
		DisplayName:  p.DisplayName,
		Enabled:      p.Enabled,
		ClientID:     p.ClientID,
		ClientSecret: secret,
		Scopes:       p.Scopes,
		CallbackURL:  p.CallbackURL,
		SortOrder:    p.SortOrder,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

// PublicInfo returns only the fields safe to show on the login page.
type AuthProviderPublicInfo struct {
	ProviderKey string `json:"providerKey"`
	DisplayName string `json:"displayName"`
	SortOrder   int    `json:"sortOrder"`
}

func (p *AuthProvider) ToPublicInfo() AuthProviderPublicInfo {
	return AuthProviderPublicInfo{
		ProviderKey: p.ProviderKey,
		DisplayName: p.DisplayName,
		SortOrder:   p.SortOrder,
	}
}
