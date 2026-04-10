package license

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"metis/internal/model"
)

// JSONText is a json.RawMessage wrapper that handles SQLite TEXT columns.
// SQLite stores JSON as string, but json.RawMessage ([]byte) cannot scan from string directly.
type JSONText json.RawMessage

func (j JSONText) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "[]", nil
	}
	return string(j), nil
}

func (j *JSONText) Scan(src any) error {
	switch v := src.(type) {
	case string:
		*j = JSONText(v)
	case []byte:
		*j = append(JSONText(nil), v...)
	case nil:
		*j = JSONText("null")
	default:
		return fmt.Errorf("JSONText.Scan: unsupported type %T", src)
	}
	return nil
}

func (j JSONText) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("[]"), nil
	}
	return []byte(j), nil
}

func (j *JSONText) UnmarshalJSON(data []byte) error {
	*j = append(JSONText(nil), data...)
	return nil
}

// RawMessage returns the underlying json.RawMessage for use with json.Unmarshal etc.
func (j JSONText) RawMessage() json.RawMessage {
	return json.RawMessage(j)
}

// Product statuses
const (
	StatusUnpublished = "unpublished"
	StatusPublished   = "published"
	StatusArchived    = "archived"
)

// Licensee statuses
const (
	LicenseeStatusActive   = "active"
	LicenseeStatusArchived = "archived"
)

// Feature types
const (
	FeatureTypeNumber      = "number"
	FeatureTypeEnum        = "enum"
	FeatureTypeMultiSelect = "multiSelect"
)

// --- ConstraintSchema types ---

type ConstraintFeature struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Type    string   `json:"type"` // number, enum, multiSelect
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Default any      `json:"default,omitempty"`
	Options []string `json:"options,omitempty"`
}

type ConstraintModule struct {
	Key      string              `json:"key"`
	Label    string              `json:"label"`
	Features []ConstraintFeature `json:"features"`
}

type ConstraintSchema = []ConstraintModule

// --- Data models ---

type Product struct {
	model.BaseModel
	Name             string          `json:"name" gorm:"size:128;not null"`
	Code             string          `json:"code" gorm:"uniqueIndex;size:64;not null"`
	Description      string          `json:"description" gorm:"type:text"`
	Status           string          `json:"status" gorm:"size:16;not null;default:unpublished"`
	ConstraintSchema JSONText `json:"constraintSchema" gorm:"type:text"`
	Plans            []Plan          `json:"plans,omitempty" gorm:"foreignKey:ProductID"`
}

func (Product) TableName() string { return "license_products" }

type Plan struct {
	model.BaseModel
	ProductID       uint            `json:"productId" gorm:"not null;index"`
	Name            string          `json:"name" gorm:"size:128;not null"`
	ConstraintValues JSONText `json:"constraintValues" gorm:"type:text"`
	IsDefault       bool            `json:"isDefault" gorm:"not null;default:false"`
	SortOrder       int             `json:"sortOrder" gorm:"not null;default:0"`
}

func (Plan) TableName() string { return "license_plans" }

type ProductKey struct {
	model.BaseModel
	ProductID       uint       `json:"productId" gorm:"not null;index"`
	Version         int        `json:"version" gorm:"not null"`
	PublicKey       string     `json:"publicKey" gorm:"type:text;not null"`
	EncryptedPrivateKey string `json:"-" gorm:"type:text;not null"`
	IsCurrent       bool       `json:"isCurrent" gorm:"not null;default:false"`
	RevokedAt       *time.Time `json:"revokedAt,omitempty"`
}

func (ProductKey) TableName() string { return "license_product_keys" }

// --- Response types ---

type ProductResponse struct {
	ID               uint            `json:"id"`
	Name             string          `json:"name"`
	Code             string          `json:"code"`
	Description      string          `json:"description"`
	Status           string          `json:"status"`
	ConstraintSchema JSONText `json:"constraintSchema"`
	PlanCount        int             `json:"planCount"`
	Plans            []PlanResponse  `json:"plans,omitempty"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

func (p *Product) ToResponse() ProductResponse {
	resp := ProductResponse{
		ID:               p.ID,
		Name:             p.Name,
		Code:             p.Code,
		Description:      p.Description,
		Status:           p.Status,
		ConstraintSchema: p.ConstraintSchema,
		PlanCount:        len(p.Plans),
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
	if len(p.Plans) > 0 {
		resp.Plans = make([]PlanResponse, len(p.Plans))
		for i, plan := range p.Plans {
			resp.Plans[i] = plan.ToResponse()
		}
	}
	return resp
}

type PlanResponse struct {
	ID               uint            `json:"id"`
	ProductID        uint            `json:"productId"`
	Name             string          `json:"name"`
	ConstraintValues JSONText `json:"constraintValues"`
	IsDefault        bool            `json:"isDefault"`
	SortOrder        int             `json:"sortOrder"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

func (p *Plan) ToResponse() PlanResponse {
	return PlanResponse{
		ID:               p.ID,
		ProductID:        p.ProductID,
		Name:             p.Name,
		ConstraintValues: p.ConstraintValues,
		IsDefault:        p.IsDefault,
		SortOrder:        p.SortOrder,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

type ProductKeyResponse struct {
	ID        uint       `json:"id"`
	Version   int        `json:"version"`
	PublicKey string     `json:"publicKey"`
	IsCurrent bool       `json:"isCurrent"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

func (k *ProductKey) ToResponse() ProductKeyResponse {
	return ProductKeyResponse{
		ID:        k.ID,
		Version:   k.Version,
		PublicKey:  k.PublicKey,
		IsCurrent: k.IsCurrent,
		RevokedAt: k.RevokedAt,
		CreatedAt: k.CreatedAt,
	}
}

// License statuses
const (
	LicenseStatusIssued  = "issued"
	LicenseStatusRevoked = "revoked"
)

type License struct {
	model.BaseModel
	ProductID        *uint      `json:"productId" gorm:"index"`
	LicenseeID       *uint      `json:"licenseeId" gorm:"index"`
	PlanID           *uint      `json:"planId"`
	PlanName         string     `json:"planName" gorm:"size:128;not null"`
	RegistrationCode string     `json:"registrationCode" gorm:"size:512;not null"`
	ConstraintValues JSONText   `json:"constraintValues" gorm:"type:text"`
	ValidFrom        time.Time  `json:"validFrom" gorm:"not null"`
	ValidUntil       *time.Time `json:"validUntil"`
	ActivationCode   string     `json:"activationCode" gorm:"type:text;not null;uniqueIndex"`
	KeyVersion       int        `json:"keyVersion" gorm:"not null"`
	Signature        string     `json:"signature" gorm:"type:text;not null"`
	Status           string     `json:"status" gorm:"size:16;not null;default:issued"`
	IssuedBy         uint       `json:"issuedBy" gorm:"not null"`
	RevokedAt        *time.Time `json:"revokedAt"`
	RevokedBy        *uint      `json:"revokedBy"`
	Notes            string     `json:"notes" gorm:"type:text"`
}

func (License) TableName() string { return "license_licenses" }

type LicenseResponse struct {
	ID               uint            `json:"id"`
	ProductID        *uint           `json:"productId"`
	LicenseeID       *uint           `json:"licenseeId"`
	PlanID           *uint           `json:"planId"`
	PlanName         string          `json:"planName"`
	RegistrationCode string          `json:"registrationCode"`
	ConstraintValues json.RawMessage `json:"constraintValues"`
	ValidFrom        time.Time       `json:"validFrom"`
	ValidUntil       *time.Time      `json:"validUntil"`
	ActivationCode   string          `json:"activationCode"`
	KeyVersion       int             `json:"keyVersion"`
	Signature        string          `json:"signature"`
	Status           string          `json:"status"`
	IssuedBy         uint            `json:"issuedBy"`
	RevokedAt        *time.Time      `json:"revokedAt,omitempty"`
	RevokedBy        *uint           `json:"revokedBy,omitempty"`
	Notes            string          `json:"notes"`
	ProductName      string          `json:"productName,omitempty"`
	ProductCode      string          `json:"productCode,omitempty"`
	LicenseeName     string          `json:"licenseeName,omitempty"`
	LicenseeCode     string          `json:"licenseeCode,omitempty"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

func (l *License) ToResponse() LicenseResponse {
	cv := json.RawMessage(l.ConstraintValues)
	if len(cv) == 0 {
		cv = json.RawMessage("{}")
	}
	return LicenseResponse{
		ID:               l.ID,
		ProductID:        l.ProductID,
		LicenseeID:       l.LicenseeID,
		PlanID:           l.PlanID,
		PlanName:         l.PlanName,
		RegistrationCode: l.RegistrationCode,
		ConstraintValues: cv,
		ValidFrom:        l.ValidFrom,
		ValidUntil:       l.ValidUntil,
		ActivationCode:   l.ActivationCode,
		KeyVersion:       l.KeyVersion,
		Signature:        l.Signature,
		Status:           l.Status,
		IssuedBy:         l.IssuedBy,
		RevokedAt:        l.RevokedAt,
		RevokedBy:        l.RevokedBy,
		Notes:            l.Notes,
		CreatedAt:        l.CreatedAt,
		UpdatedAt:        l.UpdatedAt,
	}
}

// --- Licensee ---

type BusinessInfo struct {
	Address     string `json:"address,omitempty"`
	TaxID       string `json:"taxId,omitempty"`
	BankName    string `json:"bankName,omitempty"`
	BankAccount string `json:"bankAccount,omitempty"`
	Swift       string `json:"swift,omitempty"`
	IBAN        string `json:"iban,omitempty"`
}

type Licensee struct {
	model.BaseModel
	Name         string   `json:"name" gorm:"size:128;not null;uniqueIndex"`
	Code         string   `json:"code" gorm:"size:64;not null;uniqueIndex"`
	ContactName  string   `json:"contactName" gorm:"size:64"`
	ContactPhone string   `json:"contactPhone" gorm:"size:32"`
	ContactEmail string   `json:"contactEmail" gorm:"size:128"`
	BusinessInfo JSONText `json:"businessInfo" gorm:"type:text"`
	Notes        string   `json:"notes" gorm:"type:text"`
	Status       string   `json:"status" gorm:"size:16;not null;default:active"`
}

func (Licensee) TableName() string { return "license_licensees" }

const licenseeCodeCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func generateLicenseeCode() (string, error) {
	b := make([]byte, 12)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(licenseeCodeCharset))))
		if err != nil {
			return "", err
		}
		b[i] = licenseeCodeCharset[n.Int64()]
	}
	return "LS-" + string(b), nil
}

type LicenseeResponse struct {
	ID           uint            `json:"id"`
	Name         string          `json:"name"`
	Code         string          `json:"code"`
	ContactName  string          `json:"contactName"`
	ContactPhone string          `json:"contactPhone"`
	ContactEmail string          `json:"contactEmail"`
	BusinessInfo json.RawMessage `json:"businessInfo"`
	Notes        string          `json:"notes"`
	Status       string          `json:"status"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

func (l *Licensee) ToResponse() LicenseeResponse {
	bi := json.RawMessage(l.BusinessInfo)
	if len(bi) == 0 {
		bi = json.RawMessage("{}")
	}
	return LicenseeResponse{
		ID:           l.ID,
		Name:         l.Name,
		Code:         l.Code,
		ContactName:  l.ContactName,
		ContactPhone: l.ContactPhone,
		ContactEmail: l.ContactEmail,
		BusinessInfo: bi,
		Notes:        l.Notes,
		Status:       l.Status,
		CreatedAt:    l.CreatedAt,
		UpdatedAt:    l.UpdatedAt,
	}
}
