package token

import (
	"fmt"
	"strings"
	"unicode"
)

// PasswordPolicy defines the password validation rules.
type PasswordPolicy struct {
	MinLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireNumber  bool
	RequireSpecial bool
}

// ValidatePassword checks a password against the given policy.
// Returns a list of violation messages (empty means valid).
func ValidatePassword(plain string, policy PasswordPolicy) []string {
	var violations []string

	if len(plain) < policy.MinLength {
		violations = append(violations, fmt.Sprintf("密码长度至少 %d 位", policy.MinLength))
	}

	if policy.RequireUpper && !strings.ContainsFunc(plain, unicode.IsUpper) {
		violations = append(violations, "密码需要包含大写字母")
	}

	if policy.RequireLower && !strings.ContainsFunc(plain, unicode.IsLower) {
		violations = append(violations, "密码需要包含小写字母")
	}

	if policy.RequireNumber && !strings.ContainsFunc(plain, unicode.IsDigit) {
		violations = append(violations, "密码需要包含数字")
	}

	if policy.RequireSpecial && !containsSpecial(plain) {
		violations = append(violations, "密码需要包含特殊字符")
	}

	return violations
}

func containsSpecial(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			return true
		}
	}
	return false
}
