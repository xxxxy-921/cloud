package channel

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net/smtp"
	"strings"
	"testing"
)

type fakeSMTPClient struct {
	authErr      error
	mailErr      error
	rcptErr      error
	dataErr      error
	writeErr     error
	closeErr     error
	quitErr      error
	extensions   map[string]bool
	startTLSErr  error

	authCalled      bool
	mailFrom        string
	rcptTo          []string
	writtenData     string
	startTLSCalled  bool
}

func (f *fakeSMTPClient) Auth(auth smtp.Auth) error {
	f.authCalled = true
	return f.authErr
}

func (f *fakeSMTPClient) Mail(from string) error {
	f.mailFrom = from
	return f.mailErr
}

func (f *fakeSMTPClient) Rcpt(to string) error {
	f.rcptTo = append(f.rcptTo, to)
	return f.rcptErr
}

func (f *fakeSMTPClient) Data() (io.WriteCloser, error) {
	if f.dataErr != nil {
		return nil, f.dataErr
	}
	return &fakeWriteCloser{client: f}, nil
}

func (f *fakeSMTPClient) Quit() error {
	return f.quitErr
}

func (f *fakeSMTPClient) Extension(ext string) (bool, string) {
	return f.extensions[ext], ""
}

func (f *fakeSMTPClient) StartTLS(config *tls.Config) error {
	f.startTLSCalled = true
	return f.startTLSErr
}

type fakeWriteCloser struct {
	client *fakeSMTPClient
	buf    bytes.Buffer
}

func (w *fakeWriteCloser) Write(p []byte) (int, error) {
	if w.client.writeErr != nil {
		return 0, w.client.writeErr
	}
	return w.buf.Write(p)
}

func (w *fakeWriteCloser) Close() error {
	w.client.writtenData = w.buf.String()
	return w.client.closeErr
}

func newFakeEmailDriver(client *fakeSMTPClient) *EmailDriver {
	return &EmailDriver{
		dial: func(addr, host string, secure bool) (smtpClient, error) {
			return client, nil
		},
	}
}

var baseConfig = map[string]any{
	"host":     "smtp.example.com",
	"port":     587,
	"username": "user@example.com",
	"password": "secret",
	"from":     "noreply@example.com",
}

func TestEmailDriverSend_PlainText(t *testing.T) {
	fake := &fakeSMTPClient{}
	d := newFakeEmailDriver(fake)

	err := d.Send(baseConfig, Payload{
		To:      []string{"to@example.com"},
		Subject: "Hello",
		Body:    "World",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	if !fake.authCalled {
		t.Fatal("expected Auth to be called")
	}
	if fake.mailFrom != "noreply@example.com" {
		t.Fatalf("expected Mail from noreply@example.com, got %s", fake.mailFrom)
	}
	if len(fake.rcptTo) != 1 || fake.rcptTo[0] != "to@example.com" {
		t.Fatalf("unexpected rcpt: %v", fake.rcptTo)
	}
	if !strings.Contains(fake.writtenData, "Content-Type: text/plain") {
		t.Fatal("expected plain text content type")
	}
	if !strings.Contains(fake.writtenData, "World") {
		t.Fatal("expected body in written data")
	}
}

func TestEmailDriverSend_HTML(t *testing.T) {
	fake := &fakeSMTPClient{}
	d := newFakeEmailDriver(fake)

	err := d.Send(baseConfig, Payload{
		To:      []string{"to@example.com"},
		Subject: "Hello",
		Body:    "Plain",
		HTML:    "<html>HTML</html>",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	if !strings.Contains(fake.writtenData, "Content-Type: multipart/alternative") {
		t.Fatal("expected multipart content type")
	}
	if !strings.Contains(fake.writtenData, "Plain") {
		t.Fatal("expected plain text part")
	}
	if !strings.Contains(fake.writtenData, "<html>HTML</html>") {
		t.Fatal("expected html part")
	}
}

func TestEmailDriverSend_TLS(t *testing.T) {
	fake := &fakeSMTPClient{}
	d := newFakeEmailDriver(fake)

	cfg := map[string]any{
		"host":     "smtp.example.com",
		"port":     465,
		"username": "user@example.com",
		"password": "secret",
		"from":     "noreply@example.com",
		"secure":   true,
	}
	err := d.Send(cfg, Payload{
		To:      []string{"to@example.com"},
		Subject: "Hello",
		Body:    "World",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if !fake.authCalled {
		t.Fatal("expected Auth to be called via TLS")
	}
}

func TestEmailDriverSend_MultipleRecipients(t *testing.T) {
	fake := &fakeSMTPClient{}
	d := newFakeEmailDriver(fake)

	err := d.Send(baseConfig, Payload{
		To:      []string{"a@example.com", "b@example.com"},
		Subject: "Hello",
		Body:    "World",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(fake.rcptTo) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(fake.rcptTo))
	}
}

func TestEmailDriverSend_AuthFailure(t *testing.T) {
	fake := &fakeSMTPClient{authErr: errors.New("bad creds")}
	d := newFakeEmailDriver(fake)

	err := d.Send(baseConfig, Payload{
		To:      []string{"to@example.com"},
		Subject: "Hello",
		Body:    "World",
	})
	if err == nil || err.Error() != "bad creds" {
		t.Fatalf("expected auth error, got %v", err)
	}
}

func TestEmailDriverTest_Secure(t *testing.T) {
	fake := &fakeSMTPClient{}
	d := newFakeEmailDriver(fake)

	cfg := map[string]any{
		"host":     "smtp.example.com",
		"port":     465,
		"username": "user@example.com",
		"password": "secret",
		"secure":   true,
	}
	err := d.Test(cfg)
	if err != nil {
		t.Fatalf("test: %v", err)
	}
	if !fake.authCalled {
		t.Fatal("expected Auth to be called")
	}
	if fake.startTLSCalled {
		t.Fatal("did not expect STARTTLS when secure=true")
	}
}

func TestEmailDriverTest_STARTTLS(t *testing.T) {
	fake := &fakeSMTPClient{extensions: map[string]bool{"STARTTLS": true}}
	d := newFakeEmailDriver(fake)

	err := d.Test(baseConfig)
	if err != nil {
		t.Fatalf("test: %v", err)
	}
	if !fake.startTLSCalled {
		t.Fatal("expected STARTTLS to be called")
	}
	if !fake.authCalled {
		t.Fatal("expected Auth after STARTTLS")
	}
}

func TestEmailDriverTest_PlainSMTP(t *testing.T) {
	fake := &fakeSMTPClient{extensions: map[string]bool{}}
	d := newFakeEmailDriver(fake)

	err := d.Test(baseConfig)
	if err != nil {
		t.Fatalf("test: %v", err)
	}
	if fake.startTLSCalled {
		t.Fatal("did not expect STARTTLS when not advertised")
	}
	if !fake.authCalled {
		t.Fatal("expected Auth to be called")
	}
}

func TestEmailDriverTest_AuthFailure(t *testing.T) {
	fake := &fakeSMTPClient{authErr: errors.New("invalid password")}
	d := newFakeEmailDriver(fake)

	err := d.Test(baseConfig)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "SMTP authentication failed") {
		t.Fatalf("expected wrapped auth error, got %v", err)
	}
}

func TestEmailDriverTest_STARTTLSFailure(t *testing.T) {
	fake := &fakeSMTPClient{
		extensions:  map[string]bool{"STARTTLS": true},
		startTLSErr: errors.New("tls handshake failed"),
	}
	d := newFakeEmailDriver(fake)

	err := d.Test(baseConfig)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "STARTTLS failed") {
		t.Fatalf("expected STARTTLS error, got %v", err)
	}
}

func TestEmailDriverSend_DialFailure(t *testing.T) {
	d := &EmailDriver{
		dial: func(addr, host string, secure bool) (smtpClient, error) {
			return nil, errors.New("connection refused")
		},
	}

	err := d.Send(baseConfig, Payload{
		To:      []string{"to@example.com"},
		Subject: "Hello",
		Body:    "World",
	})
	if err == nil || err.Error() != "connection refused" {
		t.Fatalf("expected dial error, got %v", err)
	}
}
