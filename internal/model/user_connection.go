package model

import "time"

type UserConnection struct {
	BaseModel
	UserID        uint   `json:"userId" gorm:"not null;uniqueIndex:idx_user_provider"`
	Provider      string `json:"provider" gorm:"size:32;not null;uniqueIndex:idx_user_provider;uniqueIndex:idx_provider_external"`
	ExternalID    string `json:"externalId" gorm:"size:255;not null;uniqueIndex:idx_provider_external"`
	ExternalName  string `json:"externalName" gorm:"size:255"`
	ExternalEmail string `json:"externalEmail" gorm:"size:255"`
	AvatarURL     string `json:"avatarUrl" gorm:"size:512"`
}

type UserConnectionResponse struct {
	ID            uint      `json:"id"`
	Provider      string    `json:"provider"`
	ExternalName  string    `json:"externalName"`
	ExternalEmail string    `json:"externalEmail"`
	AvatarURL     string    `json:"avatarUrl"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (c *UserConnection) ToResponse() UserConnectionResponse {
	return UserConnectionResponse{
		ID:            c.ID,
		Provider:      c.Provider,
		ExternalName:  c.ExternalName,
		ExternalEmail: c.ExternalEmail,
		AvatarURL:     c.AvatarURL,
		CreatedAt:     c.CreatedAt,
	}
}
