package itsm

import (
	"time"

	"metis/internal/model"
)

// FormDefinition 表单定义
type FormDefinition struct {
	model.BaseModel
	Name        string `json:"name" gorm:"size:128;not null"`
	Code        string `json:"code" gorm:"size:64;uniqueIndex;not null"`
	Description string `json:"description" gorm:"size:512"`
	Schema      string `json:"schema" gorm:"type:text;not null"`
	Version     int    `json:"version" gorm:"not null;default:1"`
	Scope       string `json:"scope" gorm:"size:16;not null;default:global"` // global | service
	ServiceID   *uint  `json:"serviceId" gorm:"index"`
	IsActive    bool   `json:"isActive" gorm:"not null;default:true"`
}

func (FormDefinition) TableName() string { return "itsm_form_definitions" }

type FormDefinitionResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Schema      JSONField `json:"schema"`
	Version     int       `json:"version"`
	Scope       string    `json:"scope"`
	ServiceID   *uint     `json:"serviceId"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (f *FormDefinition) ToResponse() FormDefinitionResponse {
	return FormDefinitionResponse{
		ID:          f.ID,
		Name:        f.Name,
		Code:        f.Code,
		Description: f.Description,
		Schema:      JSONField(f.Schema),
		Version:     f.Version,
		Scope:       f.Scope,
		ServiceID:   f.ServiceID,
		IsActive:    f.IsActive,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}
}
