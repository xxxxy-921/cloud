package model

import "time"

// AuditCategory represents the type of audit log entry.
type AuditCategory string

const (
	AuditCategoryAuth        AuditCategory = "auth"
	AuditCategoryOperation   AuditCategory = "operation"
	AuditCategoryApplication AuditCategory = "application"
)

// AuditLevel represents the severity level of an audit log entry.
type AuditLevel string

const (
	AuditLevelInfo  AuditLevel = "info"
	AuditLevelWarn  AuditLevel = "warn"
	AuditLevelError AuditLevel = "error"
)

// AuditLog is an append-only audit record. It does NOT embed BaseModel
// because audit logs have no UpdatedAt or soft-delete.
type AuditLog struct {
	ID         uint          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt  time.Time     `json:"createdAt" gorm:"index;not null"`
	Category   AuditCategory `json:"category" gorm:"type:varchar(20);not null;index:idx_audit_category_created"`
	UserID     *uint         `json:"userId" gorm:"index:idx_audit_user_created"`
	Username   string        `json:"username" gorm:"type:varchar(64)"`
	Action     string        `json:"action" gorm:"type:varchar(64);not null;index"`
	Resource   string        `json:"resource" gorm:"type:varchar(32)"`
	ResourceID string        `json:"resourceId" gorm:"type:varchar(64)"`
	Summary    string        `json:"summary" gorm:"type:text;not null"`
	Level      AuditLevel    `json:"level" gorm:"type:varchar(10);not null;default:'info'"`
	IPAddress  string        `json:"ipAddress" gorm:"type:varchar(45)"`
	UserAgent  string        `json:"userAgent" gorm:"type:varchar(512)"`
	Detail     *string       `json:"detail,omitempty" gorm:"type:text"`
}

// AuditLogResponse is the API response representation.
type AuditLogResponse struct {
	ID         uint          `json:"id"`
	CreatedAt  time.Time     `json:"createdAt"`
	Category   AuditCategory `json:"category"`
	UserID     *uint         `json:"userId"`
	Username   string        `json:"username"`
	Action     string        `json:"action"`
	Resource   string        `json:"resource"`
	ResourceID string        `json:"resourceId"`
	Summary    string        `json:"summary"`
	Level      AuditLevel    `json:"level"`
	IPAddress  string        `json:"ipAddress"`
	UserAgent  string        `json:"userAgent"`
	Detail     *string       `json:"detail,omitempty"`
}

func (a *AuditLog) ToResponse() AuditLogResponse {
	return AuditLogResponse{
		ID:         a.ID,
		CreatedAt:  a.CreatedAt,
		Category:   a.Category,
		UserID:     a.UserID,
		Username:   a.Username,
		Action:     a.Action,
		Resource:   a.Resource,
		ResourceID: a.ResourceID,
		Summary:    a.Summary,
		Level:      a.Level,
		IPAddress:  a.IPAddress,
		UserAgent:  a.UserAgent,
		Detail:     a.Detail,
	}
}
