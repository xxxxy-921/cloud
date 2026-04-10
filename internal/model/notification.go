package model

import "time"

// Notification types
const (
	NotificationTypeAnnouncement = "announcement"
)

// Notification target types
const (
	NotificationTargetAll  = "all"
	NotificationTargetUser = "user"
)

type Notification struct {
	BaseModel
	Type       string `json:"type" gorm:"size:32;not null;index"`
	Source     string `json:"source" gorm:"size:64;not null"`
	Title      string `json:"title" gorm:"size:255;not null"`
	Content    string `json:"content" gorm:"type:text"`
	TargetType string `json:"targetType" gorm:"size:16;not null;default:all"`
	TargetID   *uint  `json:"targetId" gorm:"index"`
	CreatedBy  *uint  `json:"createdBy" gorm:"index"`
}

type NotificationResponse struct {
	ID        uint      `json:"id"`
	Type      string    `json:"type"`
	Source    string    `json:"source"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	IsRead    bool      `json:"isRead"`
}

type AnnouncementResponse struct {
	ID              uint      `json:"id"`
	Title           string    `json:"title"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	CreatorUsername  string    `json:"creatorUsername"`
}

// NotificationRead tracks per-user read status for notifications.
// Does not use BaseModel — no soft delete needed for fact-type records.
type NotificationRead struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	NotificationID uint      `json:"notificationId" gorm:"not null;uniqueIndex:idx_notif_user"`
	UserID         uint      `json:"userId" gorm:"not null;uniqueIndex:idx_notif_user"`
	ReadAt         time.Time `json:"readAt" gorm:"not null"`
}
