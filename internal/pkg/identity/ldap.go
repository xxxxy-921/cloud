package identity

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"

	"metis/internal/model"
)

// LDAPAuthResult holds the result of a successful LDAP authentication.
type LDAPAuthResult struct {
	DN          string
	Username    string
	Email       string
	DisplayName string
	Avatar      string
}

// LDAPAuthenticate performs LDAP bind authentication:
// 1. Connect to server (TLS or StartTLS as configured)
// 2. Admin bind with BindDN/BindPassword
// 3. Search for user with UserFilter
// 4. Re-bind with found user DN + provided password
// 5. Return mapped attributes
func LDAPAuthenticate(cfg *model.LDAPConfig, username, password string) (*LDAPAuthResult, error) {
	conn, err := ldapConnect(cfg)
	if err != nil {
		return nil, fmt.Errorf("LDAP connect: %w", err)
	}
	defer conn.Close()

	// Admin bind
	if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		return nil, fmt.Errorf("LDAP admin bind: %w", err)
	}

	// Search for user
	filter := strings.ReplaceAll(cfg.UserFilter, "{{username}}", ldap.EscapeFilter(username))
	searchReq := ldap.NewSearchRequest(
		cfg.SearchBase,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 10, false,
		filter,
		ldapAttributes(cfg.AttributeMapping),
		nil,
	)

	result, err := conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("LDAP search: %w", err)
	}
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("user not found in LDAP")
	}

	entry := result.Entries[0]
	userDN := entry.DN

	// Re-bind with user credentials
	if err := conn.Bind(userDN, password); err != nil {
		return nil, fmt.Errorf("LDAP user bind: %w", err)
	}

	// Map attributes
	mapping := cfg.AttributeMapping
	if mapping == nil {
		mapping = model.DefaultLDAPAttributeMapping()
	}

	return &LDAPAuthResult{
		DN:          userDN,
		Username:    getAttr(entry, mapping["username"]),
		Email:       getAttr(entry, mapping["email"]),
		DisplayName: getAttr(entry, mapping["display_name"]),
		Avatar:      getAttr(entry, mapping["avatar"]),
	}, nil
}

// TestLDAPConnection tests that LDAP admin bind succeeds.
func TestLDAPConnection(cfg *model.LDAPConfig) error {
	conn, err := ldapConnect(cfg)
	if err != nil {
		return fmt.Errorf("LDAP connect: %w", err)
	}
	defer conn.Close()

	if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		return fmt.Errorf("LDAP bind: %w", err)
	}
	return nil
}

func ldapConnect(cfg *model.LDAPConfig) (*ldap.Conn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.SkipVerify,
	}

	if strings.HasPrefix(cfg.ServerURL, "ldaps://") {
		return ldap.DialURL(cfg.ServerURL, ldap.DialWithTLSConfig(tlsConfig))
	}

	conn, err := ldap.DialURL(cfg.ServerURL)
	if err != nil {
		return nil, err
	}

	if cfg.UseTLS {
		if err := conn.StartTLS(tlsConfig); err != nil {
			conn.Close()
			return nil, fmt.Errorf("StartTLS: %w", err)
		}
	}

	return conn, nil
}

func ldapAttributes(mapping map[string]string) []string {
	if mapping == nil {
		mapping = model.DefaultLDAPAttributeMapping()
	}
	seen := make(map[string]bool)
	var attrs []string
	for _, v := range mapping {
		if v != "" && !seen[v] {
			attrs = append(attrs, v)
			seen[v] = true
		}
	}
	return attrs
}

func getAttr(entry *ldap.Entry, attrName string) string {
	if attrName == "" {
		return ""
	}
	return entry.GetAttributeValue(attrName)
}
