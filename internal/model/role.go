package model

type Role struct {
	BaseModel
	Name        string `json:"name" gorm:"size:64;not null"`
	Code        string `json:"code" gorm:"uniqueIndex;size:64;not null"`
	Description string `json:"description" gorm:"size:255"`
	Sort        int    `json:"sort" gorm:"default:0"`
	IsSystem    bool   `json:"isSystem" gorm:"not null;default:false"`
}

type RoleResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}
