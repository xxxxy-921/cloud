package certificate

import (
	"encoding/json"
	"errors"
	"fmt"
	licensecrypto "metis/internal/app/license/crypto"
	"metis/internal/app/license/domain"
	licenseepkg "metis/internal/app/license/licensee"
	productpkg "metis/internal/app/license/product"
	"metis/internal/app/license/registration"
	"strings"
	"time"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
)

// timeNow is overridable in tests to make time-based logic deterministic.
var timeNow = time.Now

var (
	ErrLicenseNotFound          = errors.New("error.license.not_found")
	ErrLicenseAlreadyRevoked    = errors.New("error.license.already_revoked")
	ErrLicenseAlreadySuspended  = errors.New("error.license.already_suspended")
	ErrLicenseNotSuspended      = errors.New("error.license.not_suspended")
	ErrProductNotPublished      = errors.New("error.license.product_not_published")
	ErrLicenseeNotActive        = errors.New("error.license.licensee_not_active")
	ErrProductKeyNotFound       = domain.ErrProductKeyNotFound
	ErrRevokedLicenseNoExport   = errors.New("error.license.revoked_no_export")
	ErrRegistrationCodeRequired = errors.New("error.license.registration_code_required")
	ErrRegistrationNotFound     = errors.New("error.license.registration_not_found")
	ErrRegistrationAlreadyBound = errors.New("error.license.registration_already_bound")
	ErrRegistrationExpired      = errors.New("error.license.registration_expired")
	ErrRegistrationOwnership    = errors.New("error.license.registration_scope_mismatch")
	ErrInvalidLicenseState      = errors.New("error.license.invalid_state")
	ErrInvalidValidityPeriod    = errors.New("error.license.invalid_validity_period")
	ErrUpgradeScopeMismatch     = errors.New("error.license.upgrade_scope_mismatch")
	ErrBulkReissueTooMany       = domain.ErrBulkReissueTooMany
)

type LicenseService struct {
	licenseRepo      *LicenseRepo
	productRepo      *productpkg.ProductRepo
	licenseeRepo     *licenseepkg.LicenseeRepo
	keyRepo          *productpkg.ProductKeyRepo
	regRepo          *registration.LicenseRegistrationRepo
	db               *database.DB
	jwtSecret        []byte
	licenseKeySecret []byte
}

func NewLicenseService(i do.Injector) (*LicenseService, error) {
	licenseKeySecret, _ := do.InvokeNamed[[]byte](i, "licenseKeySecret")
	return &LicenseService{
		licenseRepo:      do.MustInvoke[*LicenseRepo](i),
		productRepo:      do.MustInvoke[*productpkg.ProductRepo](i),
		licenseeRepo:     do.MustInvoke[*licenseepkg.LicenseeRepo](i),
		keyRepo:          do.MustInvoke[*productpkg.ProductKeyRepo](i),
		regRepo:          do.MustInvoke[*registration.LicenseRegistrationRepo](i),
		db:               do.MustInvoke[*database.DB](i),
		jwtSecret:        do.MustInvoke[[]byte](i),
		licenseKeySecret: licenseKeySecret,
	}, nil
}

type IssueLicenseParams struct {
	ProductID              uint
	LicenseeID             uint
	PlanID                 *uint
	PlanName               string
	RegistrationCode       string
	AutoCreateRegistration bool
	ConstraintValues       json.RawMessage
	ValidFrom              time.Time
	ValidUntil             *time.Time
	Notes                  string
	IssuedBy               uint
}

type licensePayloadArgs struct {
	ProductCode      string
	LicenseeCode     string
	LicenseeName     string
	RegistrationCode string
	ConstraintValues domain.JSONText
	IssuedAt         time.Time
	ValidFrom        time.Time
	ValidUntil       *time.Time
	KeyVersion       int
	IssueNonce       string
}

func buildLicensePayload(args licensePayloadArgs) (map[string]any, error) {
	const issueNonceCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	var constraintMap map[string]any
	if len(args.ConstraintValues) > 0 {
		if err := json.Unmarshal(args.ConstraintValues.RawMessage(), &constraintMap); err != nil {
			return nil, fmt.Errorf("invalid constraint values: %w", err)
		}
	}
	if constraintMap == nil {
		constraintMap = make(map[string]any)
	}
	issueNonce := args.IssueNonce
	if issueNonce == "" {
		var err error
		issueNonce, err = domain.GenerateRandomCode(issueNonceCharset, 12, "LI-")
		if err != nil {
			return nil, fmt.Errorf("generate issue nonce: %w", err)
		}
	}

	payload := map[string]any{
		"v":    1,
		"pid":  args.ProductCode,
		"lic":  args.LicenseeCode,
		"licn": args.LicenseeName,
		"reg":  args.RegistrationCode,
		"con":  constraintMap,
		"iat":  args.IssuedAt.Unix(),
		"nbf":  args.ValidFrom.Unix(),
		"exp":  nil,
		"kv":   args.KeyVersion,
		"iid":  issueNonce,
	}
	if args.ValidUntil != nil {
		payload["exp"] = args.ValidUntil.Unix()
	}
	return payload, nil
}

func isRevokedLicenseState(status string, lifecycle string) bool {
	return status == domain.LicenseStatusRevoked || lifecycle == domain.LicenseLifecycleRevoked
}

func deriveLifecycleStatus(validFrom time.Time, validUntil *time.Time, now time.Time) string {
	if validFrom.After(now) {
		return domain.LicenseLifecyclePending
	}
	if validUntil != nil && !validUntil.After(now) {
		return domain.LicenseLifecycleExpired
	}
	return domain.LicenseLifecycleActive
}

func validateValidityPeriod(validFrom time.Time, validUntil *time.Time) error {
	if validUntil != nil && !validUntil.After(validFrom) {
		return ErrInvalidValidityPeriod
	}
	return nil
}

func (s *LicenseService) issueLicenseInTx(tx *gorm.DB, params IssueLicenseParams) (*domain.License, error) {
	params.RegistrationCode = strings.TrimSpace(params.RegistrationCode)
	if params.RegistrationCode == "" {
		return nil, ErrRegistrationCodeRequired
	}
	if err := validateValidityPeriod(params.ValidFrom, params.ValidUntil); err != nil {
		return nil, err
	}

	// Validate product
	product, err := s.productRepo.FindByID(params.ProductID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, productpkg.ErrProductNotFound
		}
		return nil, err
	}
	if product.Status != domain.StatusPublished {
		return nil, ErrProductNotPublished
	}

	// Validate licensee
	licensee, err := s.licenseeRepo.FindByID(params.LicenseeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, licenseepkg.ErrLicenseeNotFound
		}
		return nil, err
	}
	if licensee.Status != domain.LicenseeStatusActive {
		return nil, ErrLicenseeNotActive
	}

	// Get current key
	key, err := s.keyRepo.FindCurrentByProductID(params.ProductID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductKeyNotFound
		}
		return nil, err
	}

	// Get encryption key
	encKey, err := licensecrypto.GetEncryptionKeyWithFallback(s.licenseKeySecret, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	now := timeNow()

	// Build payload
	payload, err := buildLicensePayload(licensePayloadArgs{
		ProductCode:      product.Code,
		LicenseeCode:     licensee.Code,
		LicenseeName:     licensee.Name,
		RegistrationCode: params.RegistrationCode,
		ConstraintValues: domain.JSONText(params.ConstraintValues),
		IssuedAt:         now,
		ValidFrom:        params.ValidFrom,
		ValidUntil:       params.ValidUntil,
		KeyVersion:       key.Version,
	})
	if err != nil {
		return nil, err
	}

	// Sign
	sig, err := licensecrypto.SignLicense(payload, key.EncryptedPrivateKey, encKey)
	if err != nil {
		return nil, fmt.Errorf("sign license: %w", err)
	}

	// Generate activation code
	activationCode, err := licensecrypto.GenerateActivationCode(payload, sig)
	if err != nil {
		return nil, fmt.Errorf("generate activation code: %w", err)
	}

	// Validate registration code
	var reg *domain.LicenseRegistration
	if params.RegistrationCode != "" {
		r, err := s.regRepo.FindByCodeInTx(tx, params.RegistrationCode)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if params.AutoCreateRegistration {
					r = &domain.LicenseRegistration{
						ProductID:  &params.ProductID,
						LicenseeID: &params.LicenseeID,
						Code:       params.RegistrationCode,
						Source:     "manual_input",
					}
					if err := s.regRepo.CreateInTx(tx, r); err != nil {
						return nil, err
					}
				} else {
					return nil, ErrRegistrationNotFound
				}
			} else {
				return nil, err
			}
		}
		if r.BoundLicenseID != nil {
			return nil, ErrRegistrationAlreadyBound
		}
		if r.ExpiresAt != nil && !r.ExpiresAt.After(now) {
			return nil, ErrRegistrationExpired
		}
		if r.ProductID != nil && *r.ProductID != params.ProductID {
			return nil, ErrRegistrationOwnership
		}
		if r.LicenseeID != nil && *r.LicenseeID != params.LicenseeID {
			return nil, ErrRegistrationOwnership
		}
		reg = r
	}

	// Create license record
	license := &domain.License{
		ProductID:        &params.ProductID,
		LicenseeID:       &params.LicenseeID,
		PlanID:           params.PlanID,
		PlanName:         params.PlanName,
		RegistrationCode: params.RegistrationCode,
		ConstraintValues: domain.JSONText(params.ConstraintValues),
		ValidFrom:        params.ValidFrom,
		ValidUntil:       params.ValidUntil,
		ActivationCode:   activationCode,
		KeyVersion:       key.Version,
		Signature:        sig,
		Status:           domain.LicenseStatusIssued,
		LifecycleStatus:  deriveLifecycleStatus(params.ValidFrom, params.ValidUntil, timeNow()),
		IssuedBy:         params.IssuedBy,
		Notes:            params.Notes,
	}

	if err := s.licenseRepo.CreateInTx(tx, license); err != nil {
		return nil, err
	}
	if reg != nil {
		if err := s.regRepo.UpdateBoundLicenseInTx(tx, reg.ID, license.ID); err != nil {
			return nil, err
		}
	}

	return license, nil
}

func (s *LicenseService) IssueLicense(params IssueLicenseParams) (*domain.License, error) {
	var license *domain.License
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		license, err = s.issueLicenseInTx(tx, params)
		return err
	})
	if err != nil {
		return nil, err
	}
	return license, nil
}

func (s *LicenseService) RevokeLicense(id uint, revokedBy uint) error {
	detail, err := s.licenseRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLicenseNotFound
		}
		return err
	}
	if isRevokedLicenseState(detail.Status, detail.LifecycleStatus) {
		return ErrLicenseAlreadyRevoked
	}

	now := timeNow()
	return s.licenseRepo.UpdateStatus(id, map[string]any{
		"status":           domain.LicenseStatusRevoked,
		"lifecycle_status": domain.LicenseLifecycleRevoked,
		"revoked_at":       now,
		"revoked_by":       revokedBy,
	})
}

func (s *LicenseService) RenewLicense(id uint, newValidUntil *time.Time, renewedBy uint) error {
	detail, err := s.licenseRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLicenseNotFound
		}
		return err
	}
	if isRevokedLicenseState(detail.Status, detail.LifecycleStatus) {
		return ErrLicenseAlreadyRevoked
	}
	if detail.ProductID == nil {
		return errors.New("license has no associated product")
	}
	if err := validateValidityPeriod(detail.ValidFrom, newValidUntil); err != nil {
		return err
	}

	key, err := s.keyRepo.FindCurrentByProductID(*detail.ProductID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProductKeyNotFound
		}
		return err
	}

	encKey, err := licensecrypto.GetEncryptionKeyWithFallback(s.licenseKeySecret, s.jwtSecret)
	if err != nil {
		return err
	}

	payload, err := buildLicensePayload(licensePayloadArgs{
		ProductCode:      detail.ProductCode,
		LicenseeCode:     detail.LicenseeCode,
		LicenseeName:     detail.LicenseeName,
		RegistrationCode: detail.RegistrationCode,
		ConstraintValues: detail.ConstraintValues,
		IssuedAt:         detail.CreatedAt,
		ValidFrom:        detail.ValidFrom,
		ValidUntil:       newValidUntil,
		KeyVersion:       key.Version,
	})
	if err != nil {
		return err
	}

	sig, err := licensecrypto.SignLicense(payload, key.EncryptedPrivateKey, encKey)
	if err != nil {
		return fmt.Errorf("sign license: %w", err)
	}

	activationCode, err := licensecrypto.GenerateActivationCode(payload, sig)
	if err != nil {
		return fmt.Errorf("generate activation code: %w", err)
	}

	updates := map[string]any{
		"valid_until":     newValidUntil,
		"key_version":     key.Version,
		"signature":       sig,
		"activation_code": activationCode,
	}
	if detail.LifecycleStatus == domain.LicenseLifecycleSuspended {
		updates["lifecycle_status"] = domain.LicenseLifecycleSuspended
	} else {
		updates["lifecycle_status"] = deriveLifecycleStatus(detail.ValidFrom, newValidUntil, timeNow())
	}
	return s.licenseRepo.UpdateStatus(id, updates)
}

func (s *LicenseService) UpgradeLicense(id uint, params IssueLicenseParams) (*domain.License, error) {
	original, err := s.licenseRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLicenseNotFound
		}
		return nil, err
	}
	if isRevokedLicenseState(original.Status, original.LifecycleStatus) {
		return nil, ErrLicenseAlreadyRevoked
	}
	if original.ProductID == nil || *original.ProductID != params.ProductID {
		return nil, ErrUpgradeScopeMismatch
	}
	if original.LicenseeID == nil || *original.LicenseeID != params.LicenseeID {
		return nil, ErrUpgradeScopeMismatch
	}
	if err := validateValidityPeriod(params.ValidFrom, params.ValidUntil); err != nil {
		return nil, err
	}

	var newLicense *domain.License
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Unbind registration from original license so it can be reused for upgrade
		if original.RegistrationCode != "" && original.RegistrationCode == params.RegistrationCode {
			if err := s.regRepo.UnbindLicenseInTx(tx, params.RegistrationCode); err != nil {
				return err
			}
		}

		// Issue new license in the same transaction
		var err error
		newLicense, err = s.issueLicenseInTx(tx, params)
		if err != nil {
			return err
		}

		now := timeNow()
		// Revoke original and link to new
		if err := s.licenseRepo.UpdateStatusInTx(tx, id, map[string]any{
			"status":           domain.LicenseStatusRevoked,
			"lifecycle_status": domain.LicenseLifecycleRevoked,
			"revoked_at":       now,
			"revoked_by":       params.IssuedBy,
		}); err != nil {
			return err
		}

		if err := s.licenseRepo.UpdateStatusInTx(tx, newLicense.ID, map[string]any{
			"original_license_id": id,
		}); err != nil {
			return err
		}
		newLicense.OriginalLicenseID = &id

		if original.LifecycleStatus == domain.LicenseLifecycleSuspended {
			if err := s.licenseRepo.UpdateStatusInTx(tx, newLicense.ID, map[string]any{
				"lifecycle_status": domain.LicenseLifecycleSuspended,
				"suspended_at":     original.SuspendedAt,
				"suspended_by":     original.SuspendedBy,
			}); err != nil {
				return err
			}
			newLicense.LifecycleStatus = domain.LicenseLifecycleSuspended
			newLicense.SuspendedAt = original.SuspendedAt
			newLicense.SuspendedBy = original.SuspendedBy
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return newLicense, nil
}

func (s *LicenseService) SuspendLicense(id uint, suspendedBy uint) error {
	detail, err := s.licenseRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLicenseNotFound
		}
		return err
	}
	if isRevokedLicenseState(detail.Status, detail.LifecycleStatus) {
		return ErrLicenseAlreadyRevoked
	}
	if detail.LifecycleStatus == domain.LicenseLifecycleSuspended {
		return ErrLicenseAlreadySuspended
	}

	now := timeNow()
	return s.licenseRepo.UpdateStatus(id, map[string]any{
		"lifecycle_status": domain.LicenseLifecycleSuspended,
		"suspended_at":     now,
		"suspended_by":     suspendedBy,
	})
}

func (s *LicenseService) ReactivateLicense(id uint) error {
	detail, err := s.licenseRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLicenseNotFound
		}
		return err
	}
	if isRevokedLicenseState(detail.Status, detail.LifecycleStatus) {
		return ErrLicenseAlreadyRevoked
	}
	if detail.LifecycleStatus != domain.LicenseLifecycleSuspended {
		return ErrLicenseNotSuspended
	}

	newStatus := deriveLifecycleStatus(detail.ValidFrom, detail.ValidUntil, timeNow())
	return s.licenseRepo.UpdateStatus(id, map[string]any{
		"lifecycle_status": newStatus,
		"suspended_at":     nil,
		"suspended_by":     nil,
	})
}

func (s *LicenseService) CheckExpiredLicenses() error {
	return s.licenseRepo.UpdateExpiredStatus(timeNow(), []string{domain.LicenseLifecyclePending, domain.LicenseLifecycleActive})
}

func (s *LicenseService) GetLicense(id uint) (*LicenseDetail, error) {
	detail, err := s.licenseRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLicenseNotFound
		}
		return nil, err
	}
	return detail, nil
}

func (s *LicenseService) ListLicenses(params LicenseListParams) ([]LicenseListItem, int64, error) {
	return s.licenseRepo.List(params)
}

type LicFile struct {
	ActivationCode string `json:"activationCode"`
	PublicKey      string `json:"publicKey"`
}

func (s *LicenseService) ExportLicFile(id uint, format string) (string, string, error) {
	detail, err := s.licenseRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", ErrLicenseNotFound
		}
		return "", "", err
	}
	if isRevokedLicenseState(detail.Status, detail.LifecycleStatus) {
		return "", "", ErrRevokedLicenseNoExport
	}

	// Get the key version used for signing
	if detail.ProductID == nil {
		return "", "", errors.New("license has no associated product")
	}
	key, err := s.keyRepo.FindByProductIDAndVersion(*detail.ProductID, detail.KeyVersion)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", ErrProductKeyNotFound
		}
		return "", "", err
	}

	// Get product's license key for v2 encryption
	product, err := s.productRepo.FindByID(*detail.ProductID)
	if err != nil {
		return "", "", err
	}

	licFile := &LicFile{
		ActivationCode: detail.ActivationCode,
		PublicKey:      key.PublicKey,
	}

	plainJSON, err := json.Marshal(licFile)
	if err != nil {
		return "", "", fmt.Errorf("marshal license file: %w", err)
	}

	productIdentity := detail.ProductName
	if productIdentity == "" {
		productIdentity = detail.ProductCode
	}

	var encryptedContent string
	if format == "v2" && product.LicenseKey != "" {
		encryptedContent, err = licensecrypto.EncryptLicenseFileV2(plainJSON, detail.RegistrationCode, productIdentity, product.LicenseKey)
	} else {
		encryptedContent, err = licensecrypto.EncryptLicenseFile(plainJSON, detail.RegistrationCode, productIdentity)
	}
	if err != nil {
		return "", "", err
	}

	filename := fmt.Sprintf("%s_%s.lic", detail.ProductCode, detail.CreatedAt.Format("20060102"))
	if detail.ProductCode == "" {
		filename = fmt.Sprintf("license_%s.lic", detail.CreatedAt.Format("20060102"))
	}

	return encryptedContent, filename, nil
}

// --- domain.LicenseRegistration services ---

func generateRegistrationCode() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	return domain.GenerateRandomCode(charset, 16, "RG-")
}

func (s *LicenseService) ensureRegistrationScopeExists(productID, licenseeID *uint) error {
	if productID != nil {
		if _, err := s.productRepo.FindByID(*productID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return productpkg.ErrProductNotFound
			}
			return err
		}
	}
	if licenseeID != nil {
		if _, err := s.licenseeRepo.FindByID(*licenseeID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return licenseepkg.ErrLicenseeNotFound
			}
			return err
		}
	}
	return nil
}

type CreateLicenseRegistrationParams struct {
	ProductID  *uint
	LicenseeID *uint
	Code       string
	Source     string
	ExpiresAt  *time.Time
}

func (s *LicenseService) CreateLicenseRegistration(params CreateLicenseRegistrationParams) (*domain.LicenseRegistration, error) {
	if err := s.ensureRegistrationScopeExists(params.ProductID, params.LicenseeID); err != nil {
		return nil, err
	}
	if params.ExpiresAt != nil && !params.ExpiresAt.After(timeNow()) {
		return nil, ErrRegistrationExpired
	}

	code := strings.TrimSpace(params.Code)
	if code == "" {
		c, err := generateRegistrationCode()
		if err != nil {
			return nil, err
		}
		code = c
	}

	exists, err := s.regRepo.CodeExists(code)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("error.license.registration_code_exists")
	}

	lr := &domain.LicenseRegistration{
		ProductID:  params.ProductID,
		LicenseeID: params.LicenseeID,
		Code:       code,
		Source:     params.Source,
		ExpiresAt:  params.ExpiresAt,
	}
	if lr.Source == "" {
		lr.Source = "pre_registered"
	}

	if err := s.regRepo.Create(lr); err != nil {
		return nil, err
	}
	return lr, nil
}

func (s *LicenseService) GenerateLicenseRegistration(productID, licenseeID *uint) (*domain.LicenseRegistration, error) {
	if err := s.ensureRegistrationScopeExists(productID, licenseeID); err != nil {
		return nil, err
	}

	var code string
	for i := 0; i < 3; i++ {
		c, err := generateRegistrationCode()
		if err != nil {
			return nil, err
		}
		exists, err := s.regRepo.CodeExists(c)
		if err != nil {
			return nil, err
		}
		if !exists {
			code = c
			break
		}
	}
	if code == "" {
		return nil, errors.New("error.license.registration_code_collision")
	}

	lr := &domain.LicenseRegistration{
		ProductID:  productID,
		LicenseeID: licenseeID,
		Code:       code,
		Source:     "auto_generated",
	}
	if err := s.regRepo.Create(lr); err != nil {
		return nil, err
	}
	return lr, nil
}

func (s *LicenseService) ListLicenseRegistrations(params registration.LicenseRegistrationListParams) ([]domain.LicenseRegistration, int64, error) {
	return s.regRepo.List(params)
}

func (s *LicenseService) CleanupExpiredRegistrations() error {
	return s.regRepo.DeleteExpired(timeNow())
}

// --- Key rotation impact assessment ---

func (s *LicenseService) AssessKeyRotationImpact(productID uint) (*domain.RotateKeyImpact, error) {
	key, err := s.keyRepo.FindCurrentByProductID(productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductKeyNotFound
		}
		return nil, err
	}

	count, err := s.licenseRepo.CountByProductAndKeyVersionLessThan(productID, key.Version)
	if err != nil {
		return nil, err
	}

	return &domain.RotateKeyImpact{
		AffectedCount:  count,
		CurrentVersion: key.Version,
	}, nil
}

func (s *LicenseService) BulkReissueLicenses(productID uint, ids []uint, issuedBy uint) (int, error) {
	key, err := s.keyRepo.FindCurrentByProductID(productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrProductKeyNotFound
		}
		return 0, err
	}

	if len(ids) == 0 {
		licenses, err := s.licenseRepo.FindReissueableByProductID(productID, key.Version)
		if err != nil {
			return 0, err
		}
		ids = make([]uint, 0, len(licenses))
		for _, l := range licenses {
			ids = append(ids, l.ID)
		}
	}

	if len(ids) > 0 {
		deduped := make([]uint, 0, len(ids))
		seen := make(map[uint]struct{}, len(ids))
		for _, id := range ids {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			deduped = append(deduped, id)
		}
		ids = deduped
	}

	if len(ids) > 100 {
		return 0, ErrBulkReissueTooMany
	}

	encKey, err := licensecrypto.GetEncryptionKeyWithFallback(s.licenseKeySecret, s.jwtSecret)
	if err != nil {
		return 0, err
	}

	reissued := 0
	for _, id := range ids {
		detail, err := s.licenseRepo.FindByID(id)
		if err != nil {
			continue
		}
		if detail.ProductID == nil || *detail.ProductID != productID {
			continue
		}
		if detail.Status == domain.LicenseStatusRevoked || detail.LifecycleStatus == domain.LicenseLifecycleRevoked {
			continue
		}
		if detail.KeyVersion >= key.Version {
			continue
		}

		payload, err := buildLicensePayload(licensePayloadArgs{
			ProductCode:      detail.ProductCode,
			LicenseeCode:     detail.LicenseeCode,
			LicenseeName:     detail.LicenseeName,
			RegistrationCode: detail.RegistrationCode,
			ConstraintValues: detail.ConstraintValues,
			IssuedAt:         detail.CreatedAt,
			ValidFrom:        detail.ValidFrom,
			ValidUntil:       detail.ValidUntil,
			KeyVersion:       key.Version,
		})
		if err != nil {
			continue
		}

		sig, err := licensecrypto.SignLicense(payload, key.EncryptedPrivateKey, encKey)
		if err != nil {
			continue
		}

		activationCode, err := licensecrypto.GenerateActivationCode(payload, sig)
		if err != nil {
			continue
		}

		if err := s.licenseRepo.UpdateStatus(id, map[string]any{
			"key_version":     key.Version,
			"signature":       sig,
			"activation_code": activationCode,
		}); err != nil {
			continue
		}
		reissued++
	}

	return reissued, nil
}
