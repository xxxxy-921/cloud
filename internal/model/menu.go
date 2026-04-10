package model

type MenuType string

const (
	MenuTypeDirectory MenuType = "directory"
	MenuTypeMenu      MenuType = "menu"
	MenuTypeButton    MenuType = "button"
)

type Menu struct {
	BaseModel
	ParentID   *uint    `json:"parentId" gorm:"index"`
	Name       string   `json:"name" gorm:"size:64;not null"`
	Type       MenuType `json:"type" gorm:"size:16;not null"`
	Path       string   `json:"path" gorm:"size:255"`
	Icon       string   `json:"icon" gorm:"size:64"`
	Permission string   `json:"permission" gorm:"size:128;uniqueIndex:idx_permission_nonempty"`
	Sort       int      `json:"sort" gorm:"default:0"`
	IsHidden   bool     `json:"isHidden" gorm:"not null;default:false"`
	Children   []Menu   `json:"children,omitempty" gorm:"foreignKey:ParentID"`
}
