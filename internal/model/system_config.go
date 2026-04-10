package model

import "time"

type SystemConfig struct {
	Key       string    `json:"key" gorm:"primaryKey;size:255"`
	Value     string    `json:"value" gorm:"type:text"`
	Remark    string    `json:"remark" gorm:"size:500"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
