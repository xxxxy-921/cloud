package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestJSONTextRoundTrip(t *testing.T) {
	original := JSONText(`{"features":["vpn"],"count":2}`)
	value, err := original.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if value != `{"features":["vpn"],"count":2}` {
		t.Fatalf("Value = %#v", value)
	}

	var scanned JSONText
	if err := scanned.Scan([]byte(`{"ok":true}`)); err != nil {
		t.Fatalf("Scan bytes: %v", err)
	}
	if string(scanned) != `{"ok":true}` {
		t.Fatalf("scanned bytes = %s", scanned)
	}
	if err := scanned.Scan(`{"from":"string"}`); err != nil {
		t.Fatalf("Scan string: %v", err)
	}
	if string(scanned) != `{"from":"string"}` {
		t.Fatalf("scanned string = %s", scanned)
	}
	if err := scanned.Scan(nil); err != nil {
		t.Fatalf("Scan nil: %v", err)
	}
	if string(scanned) != "null" {
		t.Fatalf("scanned nil = %s, want null", scanned)
	}

	data, err := original.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(data) != string(original) {
		t.Fatalf("MarshalJSON = %s, want %s", data, original)
	}

	var unmarshaled JSONText
	if err := unmarshaled.UnmarshalJSON([]byte(`{"items":[1,2]}`)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if string(unmarshaled.RawMessage()) != `{"items":[1,2]}` {
		t.Fatalf("RawMessage = %s", unmarshaled.RawMessage())
	}
}

func TestLicenseDomainResponsesAndHelpers(t *testing.T) {
	now := time.Unix(1710000000, 0)
	product := &Product{
		Name:             "Metis Enterprise",
		Code:             "metis-ent",
		Description:      "enterprise",
		Status:           StatusPublished,
		LicenseKey:       "abc",
		ConstraintSchema: JSONText(`[{"key":"vpn"}]`),
		Plans: []Plan{
			{Name: "Basic", ProductID: 1, IsDefault: true, SortOrder: 1, ConstraintValues: JSONText(`{"limit":10}`)},
		},
	}
	product.ID = 1
	product.CreatedAt = now
	product.UpdatedAt = now

	resp := product.ToResponse()
	if resp.PlanCount != 1 || len(resp.Plans) != 1 {
		t.Fatalf("unexpected product response: %+v", resp)
	}
	if resp.Plans[0].Name != "Basic" || resp.Plans[0].ConstraintValues.RawMessage() == nil {
		t.Fatalf("unexpected plan response: %+v", resp.Plans[0])
	}

	revokedAt := now.Add(2 * time.Hour)
	key := &ProductKey{
		ProductID: 1,
		Version:   3,
		PublicKey: "pub-key",
		IsCurrent: true,
		RevokedAt: &revokedAt,
	}
	key.ID = 7
	key.CreatedAt = now
	keyResp := key.ToResponse()
	if keyResp.Version != 3 || keyResp.PublicKey != "pub-key" || keyResp.RevokedAt != &revokedAt {
		t.Fatalf("unexpected product key response: %+v", keyResp)
	}

	license := &License{
		PlanName:         "Basic",
		RegistrationCode: "RG-1",
		ValidFrom:        now,
		ActivationCode:   "act",
		KeyVersion:       2,
		Signature:        "sig",
		Status:           LicenseStatusIssued,
		LifecycleStatus:  LicenseLifecycleActive,
		IssuedBy:         9,
	}
	license.ID = 2
	licenseResp := license.ToResponse()
	if string(licenseResp.ConstraintValues) != "{}" {
		t.Fatalf("empty constraint values = %s, want {}", licenseResp.ConstraintValues)
	}
	if licenseResp.PlanName != "Basic" || licenseResp.KeyVersion != 2 {
		t.Fatalf("unexpected license response: %+v", licenseResp)
	}

	licensee := &Licensee{Name: "Acme", Code: "LS-001", Notes: "important", Status: LicenseeStatusActive}
	licensee.ID = 3
	licenseeResp := licensee.ToResponse()
	if licenseeResp.Name != "Acme" || licenseeResp.Code != "LS-001" {
		t.Fatalf("unexpected licensee response: %+v", licenseeResp)
	}

	if (Product{}).TableName() != "license_products" ||
		(Plan{}).TableName() != "license_plans" ||
		(ProductKey{}).TableName() != "license_product_keys" ||
		(License{}).TableName() != "license_licenses" ||
		(LicenseRegistration{}).TableName() != "license_registrations" ||
		(Licensee{}).TableName() != "license_licensees" {
		t.Fatal("unexpected table names")
	}
}

func TestGenerateCodeHelpersAndParseID(t *testing.T) {
	code, err := GenerateRandomCode("ABC123", 8, "RG-")
	if err != nil {
		t.Fatalf("GenerateRandomCode: %v", err)
	}
	if !strings.HasPrefix(code, "RG-") || len(code) != 11 {
		t.Fatalf("generated code = %q", code)
	}
	for _, ch := range code[len("RG-"):] {
		if !strings.ContainsRune("ABC123", ch) {
			t.Fatalf("unexpected rune %q in code %q", ch, code)
		}
	}

	licenseeCode, err := GenerateLicenseeCode()
	if err != nil {
		t.Fatalf("GenerateLicenseeCode: %v", err)
	}
	if !strings.HasPrefix(licenseeCode, "LS-") || len(licenseeCode) != 15 {
		t.Fatalf("licensee code = %q", licenseeCode)
	}

	c, _ := gin.CreateTestContext(nil)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	id, err := ParseID(c)
	if err != nil || id != 42 {
		t.Fatalf("ParseID = %d, %v", id, err)
	}
	c.Params = gin.Params{{Key: "id", Value: "bad"}}
	if _, err := ParseID(c); err == nil {
		t.Fatal("expected ParseID to reject non-numeric id")
	}
}

func TestLicenseResponseConstraintValuesRemainJSON(t *testing.T) {
	license := &License{
		ConstraintValues: JSONText(`{"seatLimit":20}`),
	}
	resp := license.ToResponse()

	var decoded map[string]int
	if err := json.Unmarshal(resp.ConstraintValues, &decoded); err != nil {
		t.Fatalf("unmarshal response constraint values: %v", err)
	}
	if decoded["seatLimit"] != 20 {
		t.Fatalf("decoded constraint values = %+v", decoded)
	}
}
