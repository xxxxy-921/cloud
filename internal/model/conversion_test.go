package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestJSONTextRoundTrip(t *testing.T) {
	raw := JSONText(`{"name":"metis"}`)

	val, err := raw.Value()
	if err != nil {
		t.Fatalf("Value returned error: %v", err)
	}
	if val != `{"name":"metis"}` {
		t.Fatalf("expected stored string, got %#v", val)
	}

	var scanned JSONText
	if err := scanned.Scan(val); err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if string(scanned) != string(raw) {
		t.Fatalf("expected %s, got %s", raw, scanned)
	}

	out, err := scanned.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON returned error: %v", err)
	}
	if string(out) != string(raw) {
		t.Fatalf("expected %s, got %s", raw, out)
	}
}

func TestJSONTextHandlesNilAndUnsupportedTypes(t *testing.T) {
	var raw JSONText
	val, err := raw.Value()
	if err != nil {
		t.Fatalf("Value returned error: %v", err)
	}
	if val != nil {
		t.Fatalf("expected nil value, got %#v", val)
	}

	if err := raw.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) returned error: %v", err)
	}
	if raw != nil {
		t.Fatalf("expected nil JSONText after Scan(nil), got %q", raw)
	}

	if err := raw.Scan(123); err == nil {
		t.Fatal("expected Scan to reject unsupported type")
	}

	var nilPtr *JSONText
	if err := nilPtr.UnmarshalJSON([]byte(`{}`)); err == nil {
		t.Fatal("expected nil receiver UnmarshalJSON to fail")
	}
}

func TestJSONTextUnmarshalAndNullMarshal(t *testing.T) {
	var raw JSONText
	if err := raw.UnmarshalJSON([]byte(`{"ok":true}`)); err != nil {
		t.Fatalf("UnmarshalJSON returned error: %v", err)
	}
	if string(raw) != `{"ok":true}` {
		t.Fatalf("unexpected raw content: %s", raw)
	}

	var empty JSONText
	out, err := empty.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON returned error: %v", err)
	}
	if string(out) != "null" {
		t.Fatalf("expected null, got %s", out)
	}
}

func TestAuthProviderConversions(t *testing.T) {
	now := time.Now()
	provider := &AuthProvider{
		BaseModel: BaseModel{ID: 7, CreatedAt: now, UpdatedAt: now},
		ProviderKey:  "github",
		DisplayName:  "GitHub",
		Enabled:      true,
		ClientID:     "client-id",
		ClientSecret: "secret",
		Scopes:       "read:user",
		CallbackURL:  "https://example.com/callback",
		SortOrder:    2,
	}

	resp := provider.ToResponse()
	if resp.ClientSecret != "••••••" {
		t.Fatalf("expected masked secret, got %q", resp.ClientSecret)
	}
	publicInfo := provider.ToPublicInfo()
	if publicInfo.ProviderKey != "github" || publicInfo.DisplayName != "GitHub" || publicInfo.SortOrder != 2 {
		t.Fatalf("unexpected public info: %+v", publicInfo)
	}

	provider.ClientSecret = ""
	resp = provider.ToResponse()
	if resp.ClientSecret != "" {
		t.Fatalf("expected empty secret when provider has no secret, got %q", resp.ClientSecret)
	}
}

func TestUserToResponseIncludesManagerAndPasswordMetadata(t *testing.T) {
	now := time.Now()
	managerID := uint(8)
	lockedUntil := now.Add(time.Hour)
	changedAt := now.Add(-time.Hour)
	user := &User{
		BaseModel:           BaseModel{ID: 9, CreatedAt: now, UpdatedAt: now},
		Username:            "alice",
		Password:            "hashed",
		Email:               "alice@example.com",
		Phone:               "123",
		Avatar:              "avatar.png",
		Locale:              "zh-CN",
		Timezone:            "Asia/Shanghai",
		RoleID:              3,
		Role:                Role{BaseModel: BaseModel{ID: 3}, Name: "管理员", Code: RoleAdmin},
		ManagerID:           &managerID,
		Manager:             &User{BaseModel: BaseModel{ID: managerID}, Username: "boss", Avatar: "boss.png"},
		IsActive:            true,
		PasswordChangedAt:   &changedAt,
		ForcePasswordReset:  true,
		FailedLoginAttempts: 2,
		LockedUntil:         &lockedUntil,
		TwoFactorEnabled:    true,
	}

	resp := user.ToResponse()
	if !resp.HasPassword || resp.Manager == nil {
		t.Fatalf("expected password and manager info, got %+v", resp)
	}
	if resp.Manager.Username != "boss" || resp.Role.Code != RoleAdmin {
		t.Fatalf("unexpected response payload: %+v", resp)
	}
	if !user.HasPassword() {
		t.Fatal("expected HasPassword to be true")
	}
	if !user.IsLocked() {
		t.Fatal("expected IsLocked to be true")
	}

	user.Password = ""
	user.Manager = nil
	expired := now.Add(-time.Hour).UTC()
	user.LockedUntil = &expired
	resp = user.ToResponse()
	if resp.HasPassword {
		t.Fatal("expected HasPassword to be false")
	}
	if resp.Manager != nil {
		t.Fatalf("expected nil manager info, got %+v", resp.Manager)
	}
	if user.IsLocked() {
		t.Fatal("expected IsLocked to be false after expiration")
	}
}

func TestUserConnectionAndMessageChannelResponses(t *testing.T) {
	conn := (&UserConnection{
		BaseModel:     BaseModel{ID: 10},
		UserID:        1,
		Provider:      "github",
		ExternalID:    "42",
		ExternalName:  "alice",
		ExternalEmail: "alice@example.com",
		AvatarURL:     "https://example.com/avatar.png",
	}).ToResponse()
	if conn.Provider != "github" || conn.ExternalName != "alice" {
		t.Fatalf("unexpected connection response: %+v", conn)
	}

	ch := (&MessageChannel{
		BaseModel: BaseModel{ID: 3},
		Name:      "SMTP",
		Type:      "email",
		Config:    `{"password":"secret","host":"smtp.example.com"}`,
		Enabled:   true,
	}).ToResponse()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(ch.Config), &parsed); err != nil {
		t.Fatalf("channel config should stay valid JSON: %v", err)
	}
	if parsed["password"] != "secret" || parsed["host"] != "smtp.example.com" {
		t.Fatalf("unexpected channel config payload: %+v", parsed)
	}
}
