package service

import (
	"strconv"

	"github.com/samber/do/v2"

	"metis/internal/model"
	"metis/internal/pkg/token"
	"metis/internal/repository"
)

type SettingsService struct {
	sysConfigRepo *repository.SysConfigRepo
}

func NewSettings(i do.Injector) (*SettingsService, error) {
	return &SettingsService{
		sysConfigRepo: do.MustInvoke[*repository.SysConfigRepo](i),
	}, nil
}

// SecuritySettings represents the security configuration group.
type SecuritySettings struct {
	// Session
	MaxConcurrentSessions int `json:"maxConcurrentSessions"`
	SessionTimeoutMinutes int `json:"sessionTimeoutMinutes"`

	// Password policy
	PasswordMinLength     int  `json:"passwordMinLength"`
	PasswordRequireUpper  bool `json:"passwordRequireUpper"`
	PasswordRequireLower  bool `json:"passwordRequireLower"`
	PasswordRequireNumber bool `json:"passwordRequireNumber"`
	PasswordRequireSpecial bool `json:"passwordRequireSpecial"`
	PasswordExpiryDays    int  `json:"passwordExpiryDays"`

	// Login security
	LoginMaxAttempts    int    `json:"loginMaxAttempts"`
	LoginLockoutMinutes int    `json:"loginLockoutMinutes"`
	CaptchaProvider     string `json:"captchaProvider"`

	// Two-factor
	RequireTwoFactor bool `json:"requireTwoFactor"`

	// Registration
	RegistrationOpen bool   `json:"registrationOpen"`
	DefaultRoleCode  string `json:"defaultRoleCode"`
}

// SchedulerSettings represents the scheduler/cleanup configuration group.
type SchedulerSettings struct {
	HistoryRetentionDays        int `json:"historyRetentionDays"`
	AuditRetentionDaysAuth      int `json:"auditRetentionDaysAuth"`
	AuditRetentionDaysOperation int `json:"auditRetentionDaysOperation"`
}

func (s *SettingsService) GetSecuritySettings() SecuritySettings {
	return SecuritySettings{
		MaxConcurrentSessions:       s.getInt("security.max_concurrent_sessions", 5),
		SessionTimeoutMinutes:       s.getInt("security.session_timeout_minutes", 10080),
		PasswordMinLength:           s.getInt("security.password_min_length", 8),
		PasswordRequireUpper:        s.getBool("security.password_require_upper", false),
		PasswordRequireLower:        s.getBool("security.password_require_lower", false),
		PasswordRequireNumber:       s.getBool("security.password_require_number", false),
		PasswordRequireSpecial:      s.getBool("security.password_require_special", false),
		PasswordExpiryDays:          s.getInt("security.password_expiry_days", 0),
		LoginMaxAttempts:            s.getInt("security.login_max_attempts", 5),
		LoginLockoutMinutes:         s.getInt("security.login_lockout_minutes", 30),
		CaptchaProvider:             s.getString("security.captcha_provider", "none"),
		RequireTwoFactor:            s.getBool("security.require_two_factor", false),
		RegistrationOpen:            s.getBool("security.registration_open", false),
		DefaultRoleCode:             s.getString("security.default_role_code", ""),
	}
}

func (s *SettingsService) UpdateSecuritySettings(settings SecuritySettings) error {
	// Validation
	if settings.PasswordMinLength < 1 {
		settings.PasswordMinLength = 1
	}
	if settings.LoginMaxAttempts < 0 {
		settings.LoginMaxAttempts = 0
	}
	if settings.LoginLockoutMinutes < 0 {
		settings.LoginLockoutMinutes = 0
	}
	if settings.SessionTimeoutMinutes < 1 {
		settings.SessionTimeoutMinutes = 10080
	}
	if settings.CaptchaProvider != "none" && settings.CaptchaProvider != "image" {
		settings.CaptchaProvider = "none"
	}

	boolStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}

	configs := []model.SystemConfig{
		{Key: "security.max_concurrent_sessions", Value: strconv.Itoa(settings.MaxConcurrentSessions), Remark: "每用户最大并发会话数，0 表示不限制"},
		{Key: "security.session_timeout_minutes", Value: strconv.Itoa(settings.SessionTimeoutMinutes), Remark: "会话超时时间（分钟）"},
		{Key: "security.password_min_length", Value: strconv.Itoa(settings.PasswordMinLength), Remark: "密码最小长度"},
		{Key: "security.password_require_upper", Value: boolStr(settings.PasswordRequireUpper), Remark: "密码需要大写字母"},
		{Key: "security.password_require_lower", Value: boolStr(settings.PasswordRequireLower), Remark: "密码需要小写字母"},
		{Key: "security.password_require_number", Value: boolStr(settings.PasswordRequireNumber), Remark: "密码需要数字"},
		{Key: "security.password_require_special", Value: boolStr(settings.PasswordRequireSpecial), Remark: "密码需要特殊字符"},
		{Key: "security.password_expiry_days", Value: strconv.Itoa(settings.PasswordExpiryDays), Remark: "密码过期天数，0 表示永不过期"},
		{Key: "security.login_max_attempts", Value: strconv.Itoa(settings.LoginMaxAttempts), Remark: "最大登录失败次数，0 表示不限制"},
		{Key: "security.login_lockout_minutes", Value: strconv.Itoa(settings.LoginLockoutMinutes), Remark: "账户锁定时长（分钟）"},
		{Key: "security.captcha_provider", Value: settings.CaptchaProvider, Remark: "验证码提供商：none/image"},
		{Key: "security.require_two_factor", Value: boolStr(settings.RequireTwoFactor), Remark: "强制所有用户启用两步验证"},
		{Key: "security.registration_open", Value: boolStr(settings.RegistrationOpen), Remark: "开放用户注册"},
		{Key: "security.default_role_code", Value: settings.DefaultRoleCode, Remark: "新注册用户默认角色代码"},
	}
	for _, cfg := range configs {
		if err := s.sysConfigRepo.Set(&cfg); err != nil {
			return err
		}
	}
	return nil
}

func (s *SettingsService) GetSchedulerSettings() SchedulerSettings {
	return SchedulerSettings{
		HistoryRetentionDays:        s.getInt("scheduler.history_retention_days", 30),
		AuditRetentionDaysAuth:      s.getInt("audit.retention_days_auth", 90),
		AuditRetentionDaysOperation: s.getInt("audit.retention_days_operation", 365),
	}
}

func (s *SettingsService) UpdateSchedulerSettings(settings SchedulerSettings) error {
	configs := []model.SystemConfig{
		{Key: "scheduler.history_retention_days", Value: strconv.Itoa(settings.HistoryRetentionDays), Remark: "任务执行历史保留天数，0 表示永不清理"},
		{Key: "audit.retention_days_auth", Value: strconv.Itoa(settings.AuditRetentionDaysAuth), Remark: "登录活动日志保留天数，0 表示永不清理"},
		{Key: "audit.retention_days_operation", Value: strconv.Itoa(settings.AuditRetentionDaysOperation), Remark: "操作记录日志保留天数，0 表示永不清理"},
	}
	for _, cfg := range configs {
		if err := s.sysConfigRepo.Set(&cfg); err != nil {
			return err
		}
	}
	return nil
}

func (s *SettingsService) getInt(key string, defaultVal int) int {
	cfg, err := s.sysConfigRepo.Get(key)
	if err != nil {
		return defaultVal
	}
	v, err := strconv.Atoi(cfg.Value)
	if err != nil {
		return defaultVal
	}
	return v
}

func (s *SettingsService) getBool(key string, defaultVal bool) bool {
	cfg, err := s.sysConfigRepo.Get(key)
	if err != nil {
		return defaultVal
	}
	return cfg.Value == "true"
}

func (s *SettingsService) getString(key string, defaultVal string) string {
	cfg, err := s.sysConfigRepo.Get(key)
	if err != nil {
		return defaultVal
	}
	return cfg.Value
}

// GetPasswordPolicy reads the password policy from SystemConfig.
func (s *SettingsService) GetPasswordPolicy() token.PasswordPolicy {
	return token.PasswordPolicy{
		MinLength:      s.getInt("security.password_min_length", 8),
		RequireUpper:   s.getBool("security.password_require_upper", false),
		RequireLower:   s.getBool("security.password_require_lower", false),
		RequireNumber:  s.getBool("security.password_require_number", false),
		RequireSpecial: s.getBool("security.password_require_special", false),
	}
}

// GetPasswordExpiryDays returns the password expiry in days (0 = never).
func (s *SettingsService) GetPasswordExpiryDays() int {
	return s.getInt("security.password_expiry_days", 0)
}

// GetSessionTimeoutMinutes returns the session timeout in minutes.
func (s *SettingsService) GetSessionTimeoutMinutes() int {
	v := s.getInt("security.session_timeout_minutes", 10080)
	if v <= 0 {
		return 10080
	}
	return v
}

// GetLoginLockoutSettings returns max attempts and lockout duration.
func (s *SettingsService) GetLoginLockoutSettings() (maxAttempts int, lockoutMinutes int) {
	return s.getInt("security.login_max_attempts", 5), s.getInt("security.login_lockout_minutes", 30)
}

// GetCaptchaProvider returns the configured captcha provider ("none" or "image").
func (s *SettingsService) GetCaptchaProvider() string {
	return s.getString("security.captcha_provider", "none")
}

// IsRegistrationOpen returns whether self-registration is enabled.
func (s *SettingsService) IsRegistrationOpen() bool {
	return s.getBool("security.registration_open", false)
}

// GetDefaultRoleCode returns the default role code for new registrations.
func (s *SettingsService) GetDefaultRoleCode() string {
	return s.getString("security.default_role_code", "")
}

// IsTwoFactorRequired returns whether 2FA is mandatory for all users.
func (s *SettingsService) IsTwoFactorRequired() bool {
	return s.getBool("security.require_two_factor", false)
}
