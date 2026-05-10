package identity

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	ber "github.com/go-asn1-ber/asn1-ber"
	"github.com/go-ldap/ldap/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"

	"metis/internal/model"
)

func newOIDCDiscoveryServer() *httptest.Server {
	mux := http.NewServeMux()
	var issuer string
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/authorize",
			"token_endpoint":         issuer + "/token",
			"jwks_uri":               issuer + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{}})
	})
	srv := httptest.NewServer(mux)
	issuer = srv.URL
	return srv
}

func TestGetOIDCProviderCacheAndOAuth2Config(t *testing.T) {
	srv := newOIDCDiscoveryServer()
	defer srv.Close()
	t.Cleanup(func() {
		ClearOIDCProviderCache(1)
	})

	cfg := &model.OIDCConfig{
		IssuerURL:   srv.URL,
		ClientID:    "client-id",
		ClientSecret:"secret",
		CallbackURL: "https://app.example.com/callback",
	}
	ctx := context.Background()

	first, err := GetOIDCProvider(ctx, 1, cfg)
	if err != nil {
		t.Fatalf("GetOIDCProvider returned error: %v", err)
	}
	second, err := GetOIDCProvider(ctx, 1, cfg)
	if err != nil {
		t.Fatalf("second GetOIDCProvider returned error: %v", err)
	}
	if first != second {
		t.Fatal("expected cached provider instance on second lookup")
	}

	oauthCfg := first.OAuth2Config()
	if oauthCfg.ClientID != "client-id" || oauthCfg.ClientSecret != "secret" || oauthCfg.RedirectURL != "https://app.example.com/callback" {
		t.Fatalf("unexpected OAuth config: %+v", oauthCfg)
	}
	if len(oauthCfg.Scopes) != 3 || oauthCfg.Scopes[0] != gooidc.ScopeOpenID {
		t.Fatalf("expected default OIDC scopes, got %+v", oauthCfg.Scopes)
	}

	pkce, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE returned error: %v", err)
	}
	if pkce.Verifier == "" || pkce.Challenge == "" {
		t.Fatalf("expected PKCE verifier and challenge, got %+v", pkce)
	}
	if url := first.AuthURL("state-1", pkce); !strings.Contains(url, "code_challenge=") || !strings.Contains(url, "state=state-1") {
		t.Fatalf("expected PKCE auth URL, got %s", url)
	}
	if url := first.AuthURL("state-2", nil); !strings.Contains(url, "state=state-2") {
		t.Fatalf("expected auth URL with state, got %s", url)
	}

	ClearOIDCProviderCache(1)
	third, err := GetOIDCProvider(ctx, 1, cfg)
	if err != nil {
		t.Fatalf("GetOIDCProvider after clear returned error: %v", err)
	}
	if third == first {
		t.Fatal("expected cache clear to force a new provider instance")
	}
}

func TestOIDCProviderFailurePaths(t *testing.T) {
	ctx := context.Background()
	if _, err := GetOIDCProvider(ctx, 99, &model.OIDCConfig{IssuerURL: "http://127.0.0.1:1"}); err == nil {
		t.Fatal("expected OIDC discovery to fail for invalid issuer")
	}
	if err := TestOIDCDiscovery(ctx, "http://127.0.0.1:1"); err == nil {
		t.Fatal("expected TestOIDCDiscovery to fail for invalid issuer")
	}
}

func newSignedOIDCServer(t *testing.T) *httptest.Server {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	mux := http.NewServeMux()
	var issuer string
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"token_endpoint":                        issuer + "/token",
			"jwks_uri":                              issuer + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		pub := privateKey.PublicKey
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{{
				"kty": "RSA",
				"kid": "test-key",
				"use": "sig",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
			}},
		})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if got := r.Form.Get("code"); got == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		claims := jwt.MapClaims{
			"iss": issuer,
			"sub": "user-1",
			"aud": []string{"client-id"},
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
			"email": "user@example.com",
			"name": "OIDC User",
			"picture": "https://example.com/avatar.png",
		}
		tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(privateKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     tokenStr,
		})
	})
	srv := httptest.NewServer(mux)
	issuer = srv.URL
	return srv
}

func TestOIDCProviderExchangeVerifyAndClaims(t *testing.T) {
	srv := newSignedOIDCServer(t)
	defer srv.Close()
	t.Cleanup(func() {
		ClearOIDCProviderCache(2)
	})

	cfg := &model.OIDCConfig{
		IssuerURL:   srv.URL,
		ClientID:    "client-id",
		ClientSecret: "secret",
		CallbackURL: "https://app.example.com/callback",
	}
	op, err := GetOIDCProvider(context.Background(), 2, cfg)
	if err != nil {
		t.Fatalf("GetOIDCProvider: %v", err)
	}
	if op.Verifier() == nil {
		t.Fatal("expected non-nil verifier")
	}

	tokenResp, err := op.ExchangeCode(context.Background(), "auth-code", "verifier-1")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if tokenResp.AccessToken != "access-token" {
		t.Fatalf("expected access token, got %q", tokenResp.AccessToken)
	}

	idToken, err := op.VerifyIDToken(context.Background(), tokenResp)
	if err != nil {
		t.Fatalf("VerifyIDToken: %v", err)
	}
	claims, err := ExtractClaims(idToken)
	if err != nil {
		t.Fatalf("ExtractClaims: %v", err)
	}
	if claims.Sub != "user-1" || claims.Email != "user@example.com" || claims.Name != "OIDC User" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestOIDCProviderVerifyIDTokenRequiresIDToken(t *testing.T) {
	srv := newSignedOIDCServer(t)
	defer srv.Close()
	t.Cleanup(func() {
		ClearOIDCProviderCache(3)
	})

	cfg := &model.OIDCConfig{IssuerURL: srv.URL, ClientID: "client-id", ClientSecret: "secret"}
	op, err := GetOIDCProvider(context.Background(), 3, cfg)
	if err != nil {
		t.Fatalf("GetOIDCProvider: %v", err)
	}
	if _, err := op.VerifyIDToken(context.Background(), &oauth2.Token{AccessToken: "access"}); err == nil {
		t.Fatal("expected missing id_token error")
	}
}

func TestExtractClaims_Failure(t *testing.T) {
	if _, err := ExtractClaims(&gooidc.IDToken{}); err == nil {
		t.Fatal("expected claims extraction error")
	}
}

func TestOIDCProviderExchangeCodeFailure(t *testing.T) {
	mux := http.NewServeMux()
	var issuer string
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/authorize",
			"token_endpoint":         issuer + "/token",
			"jwks_uri":               issuer + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad token exchange", http.StatusBadRequest)
	})
	srv := httptest.NewServer(mux)
	issuer = srv.URL
	defer srv.Close()
	t.Cleanup(func() {
		ClearOIDCProviderCache(4)
	})

	cfg := &model.OIDCConfig{IssuerURL: srv.URL, ClientID: "client-id", ClientSecret: "secret", CallbackURL: "https://app.example.com/callback"}
	op, err := GetOIDCProvider(context.Background(), 4, cfg)
	if err != nil {
		t.Fatalf("GetOIDCProvider: %v", err)
	}
	if _, err := op.ExchangeCode(context.Background(), "bad-code", "verifier"); err == nil {
		t.Fatal("expected token exchange error")
	}
}

func TestLDAPHelpersAndFailurePaths(t *testing.T) {
	cfg := &model.LDAPConfig{
		ServerURL:    "ldap://127.0.0.1:1",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "secret",
		SearchBase:   "dc=example,dc=com",
		UserFilter:   "(uid={{username}})",
	}
	if _, err := LDAPAuthenticate(cfg, "alice", "password"); err == nil {
		t.Fatal("expected LDAPAuthenticate to fail for unreachable LDAP server")
	}
	if err := TestLDAPConnection(cfg); err == nil {
		t.Fatal("expected TestLDAPConnection to fail for unreachable LDAP server")
	}

	attrs := ldapAttributes(map[string]string{
		"username":     "uid",
		"email":        "mail",
		"display_name": "cn",
		"avatar":       "mail",
	})
	if len(attrs) != 3 {
		t.Fatalf("expected deduplicated LDAP attributes, got %+v", attrs)
	}
	defaultAttrs := ldapAttributes(nil)
	if len(defaultAttrs) == 0 {
		t.Fatal("expected default LDAP attributes when mapping is nil")
	}

	entry := &ldap.Entry{
		DN: "uid=alice,dc=example,dc=com",
		Attributes: []*ldap.EntryAttribute{
			{Name: "uid", Values: []string{"alice"}},
			{Name: "mail", Values: []string{"alice@example.com"}},
		},
	}
	if got := getAttr(entry, "uid"); got != "alice" {
		t.Fatalf("expected uid attribute, got %q", got)
	}
	if got := getAttr(entry, ""); got != "" {
		t.Fatalf("expected empty attribute lookup to return empty string, got %q", got)
	}
}

func TestLDAPMalformedURLAndStartTLSFailure(t *testing.T) {
	if _, err := ldapConnect(&model.LDAPConfig{ServerURL: "://bad-url"}); err == nil {
		t.Fatal("expected malformed LDAP URL to fail")
	}
	if err := TestLDAPConnection(&model.LDAPConfig{ServerURL: "://bad-url"}); err == nil {
		t.Fatal("expected TestLDAPConnection to fail for malformed URL")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()

	cfg := &model.LDAPConfig{
		ServerURL:   "ldap://" + ln.Addr().String(),
		UseTLS:      true,
		SkipVerify:  true,
		BindDN:      "cn=admin,dc=example,dc=com",
		BindPassword:"secret",
	}
	if _, err := ldapConnect(cfg); err == nil {
		t.Fatal("expected StartTLS handshake failure")
	}
	<-done
}

func newFakeLDAPServer(t *testing.T, hold time.Duration) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen fake ldap: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				_, _ = c.Read(buf)
				if hold > 0 {
					time.Sleep(hold)
				}
				_, _ = io.WriteString(c, "\x30\x0c\x02\x01\x01\x61\x07\x0a\x01\x02\x04\x00\x04\x00")
			}(conn)
		}
	}()
	return "ldap://" + ln.Addr().String()
}

func TestLDAPAuthenticateAndConnection_BindFailurePaths(t *testing.T) {
	cfg := &model.LDAPConfig{
		ServerURL:    newFakeLDAPServer(t, 0),
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "secret",
		SearchBase:   "dc=example,dc=com",
		UserFilter:   "(uid={{username}})",
	}

	if _, err := LDAPAuthenticate(cfg, "alice", "password"); err == nil || !strings.Contains(err.Error(), "LDAP admin bind") {
		t.Fatalf("expected LDAP admin bind failure, got %v", err)
	}
	if err := TestLDAPConnection(cfg); err == nil || !strings.Contains(err.Error(), "LDAP bind") {
		t.Fatalf("expected TestLDAPConnection bind failure, got %v", err)
	}
}

func TestSSOStateManagerCleanupStopsOnDone(t *testing.T) {
	sm := &SSOStateManager{
		done:  make(chan struct{}),
		nowFn: time.Now,
	}
	close(sm.done)

	finished := make(chan struct{})
	go func() {
		sm.cleanup()
		close(finished)
	}()

	select {
	case <-finished:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected cleanup to exit after done is closed")
	}
}

func encodeLDAPResponse(t *testing.T, packet *ber.Packet) []byte {
	t.Helper()
	out := packet.Bytes()
	if len(out) == 0 {
		t.Fatal("expected BER packet bytes")
	}
	return out
}

func ldapMessage(messageID int64, child *ber.Packet) *ber.Packet {
	packet := ber.NewSequence("LDAPMessage")
	packet.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, messageID, "messageID"))
	packet.AppendChild(child)
	return packet
}

func ldapResultPacket(appTag ber.Tag, resultCode int64, matchedDN, diagnostic string) *ber.Packet {
	resp := ber.Encode(ber.ClassApplication, ber.TypeConstructed, appTag, nil, "LDAP Response")
	resp.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, resultCode, "resultCode"))
	resp.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, matchedDN, "matchedDN"))
	resp.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, diagnostic, "diagnosticMessage"))
	return resp
}

func ldapSearchEntryPacket(dn string, attrs map[string][]string) *ber.Packet {
	entry := ber.Encode(ber.ClassApplication, ber.TypeConstructed, ldap.ApplicationSearchResultEntry, nil, "Search Result Entry")
	entry.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, dn, "objectName"))
	attrList := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "attributes")
	for name, values := range attrs {
		attr := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "attribute")
		attr.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, name, "type"))
		valueSet := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSet, nil, "vals")
		for _, value := range values {
			valueSet.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, value, "value"))
		}
		attr.AppendChild(valueSet)
		attrList.AppendChild(attr)
	}
	entry.AppendChild(attrList)
	return entry
}

func readLDAPPacket(t *testing.T, conn net.Conn) *ber.Packet {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	packet, err := ber.ReadPacket(conn)
	if err != nil {
		t.Fatalf("read ldap packet: %v", err)
	}
	return packet
}

func startScriptedLDAPServer(t *testing.T, handler func(net.Conn)) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen ldap: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				handler(c)
			}(conn)
		}
	}()
	return "ldap://" + ln.Addr().String()
}

func TestLDAPAuthenticate_SuccessAndFailurePaths(t *testing.T) {
	successURL := startScriptedLDAPServer(t, func(conn net.Conn) {
		adminBind := readLDAPPacket(t, conn)
		if _, err := conn.Write(encodeLDAPResponse(t, ldapMessage(adminBind.Children[0].Value.(int64), ldapResultPacket(ldap.ApplicationBindResponse, int64(ldap.LDAPResultSuccess), "", "")))); err != nil {
			t.Errorf("write admin bind response: %v", err)
			return
		}

		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Errorf("set read deadline: %v", err)
			return
		}
		searchReq, err := ber.ReadPacket(conn)
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Errorf("read search request: %v", err)
			return
		}
		msgID := searchReq.Children[0].Value.(int64)
		entry := ldapSearchEntryPacket("uid=alice,dc=example,dc=com", map[string][]string{
			"uid":  {"alice"},
			"mail": {"alice@example.com"},
			"cn":   {"Alice"},
		})
		if _, err := conn.Write(encodeLDAPResponse(t, ldapMessage(msgID, entry))); err != nil {
			t.Errorf("write search entry: %v", err)
			return
		}
		if _, err := conn.Write(encodeLDAPResponse(t, ldapMessage(msgID, ldapResultPacket(ldap.ApplicationSearchResultDone, int64(ldap.LDAPResultSuccess), "", "")))); err != nil {
			t.Errorf("write search done: %v", err)
			return
		}

		userBind := readLDAPPacket(t, conn)
		if _, err := conn.Write(encodeLDAPResponse(t, ldapMessage(userBind.Children[0].Value.(int64), ldapResultPacket(ldap.ApplicationBindResponse, int64(ldap.LDAPResultSuccess), "", "")))); err != nil {
			t.Errorf("write user bind response: %v", err)
		}
	})

	cfg := &model.LDAPConfig{
		ServerURL:  successURL,
		BindDN:     "cn=admin,dc=example,dc=com",
		BindPassword:"secret",
		SearchBase: "dc=example,dc=com",
		UserFilter: "(uid={{username}})",
		AttributeMapping: map[string]string{
			"username":     "uid",
			"email":        "mail",
			"display_name": "cn",
		},
	}
	result, err := LDAPAuthenticate(cfg, "alice", "password")
	if err != nil {
		t.Fatalf("expected LDAPAuthenticate success, got %v", err)
	}
	if result.DN != "uid=alice,dc=example,dc=com" || result.Username != "alice" || result.Email != "alice@example.com" || result.DisplayName != "Alice" {
		t.Fatalf("unexpected ldap auth result: %+v", result)
	}
	if err := TestLDAPConnection(cfg); err != nil {
		t.Fatalf("expected TestLDAPConnection success, got %v", err)
	}

	missingUserURL := startScriptedLDAPServer(t, func(conn net.Conn) {
		adminBind := readLDAPPacket(t, conn)
		_, _ = conn.Write(encodeLDAPResponse(t, ldapMessage(adminBind.Children[0].Value.(int64), ldapResultPacket(ldap.ApplicationBindResponse, int64(ldap.LDAPResultSuccess), "", ""))))
		searchReq := readLDAPPacket(t, conn)
		_, _ = conn.Write(encodeLDAPResponse(t, ldapMessage(searchReq.Children[0].Value.(int64), ldapResultPacket(ldap.ApplicationSearchResultDone, int64(ldap.LDAPResultSuccess), "", ""))))
	})
	cfg.ServerURL = missingUserURL
	if _, err := LDAPAuthenticate(cfg, "alice", "password"); err == nil || !strings.Contains(err.Error(), "user not found in LDAP") {
		t.Fatalf("expected missing user error, got %v", err)
	}

	userBindFailURL := startScriptedLDAPServer(t, func(conn net.Conn) {
		adminBind := readLDAPPacket(t, conn)
		_, _ = conn.Write(encodeLDAPResponse(t, ldapMessage(adminBind.Children[0].Value.(int64), ldapResultPacket(ldap.ApplicationBindResponse, int64(ldap.LDAPResultSuccess), "", ""))))
		searchReq := readLDAPPacket(t, conn)
		msgID := searchReq.Children[0].Value.(int64)
		_, _ = conn.Write(encodeLDAPResponse(t, ldapMessage(msgID, ldapSearchEntryPacket("uid=alice,dc=example,dc=com", map[string][]string{"uid": {"alice"}}))))
		_, _ = conn.Write(encodeLDAPResponse(t, ldapMessage(msgID, ldapResultPacket(ldap.ApplicationSearchResultDone, int64(ldap.LDAPResultSuccess), "", ""))))
		userBind := readLDAPPacket(t, conn)
		_, _ = conn.Write(encodeLDAPResponse(t, ldapMessage(userBind.Children[0].Value.(int64), ldapResultPacket(ldap.ApplicationBindResponse, int64(ldap.LDAPResultInvalidCredentials), "", "bad password"))))
	})
	cfg.ServerURL = userBindFailURL
	if _, err := LDAPAuthenticate(cfg, "alice", "bad-password"); err == nil || !strings.Contains(err.Error(), "LDAP user bind") {
		t.Fatalf("expected user bind failure, got %v", err)
	}
}
