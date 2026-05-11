package service

import (
	"testing"
	"time"

	"github.com/samber/do/v2"
)

func TestCaptchaServiceGenerateAndVerify(t *testing.T) {
	injector := do.New()
	svc, err := NewCaptcha(injector)
	if err != nil {
		t.Fatalf("NewCaptcha: %v", err)
	}

	result, err := svc.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.ID == "" || result.Image == "" {
		t.Fatalf("expected captcha payload, got %+v", result)
	}

	if svc.Verify("missing", "12345") {
		t.Fatal("expected missing captcha to fail")
	}
}

func TestCaptchaStoreSetGetVerifyAndExpiry(t *testing.T) {
	store := newCaptchaStore(20*time.Millisecond, 5*time.Millisecond)
	t.Cleanup(func() { close(store.stopped) })

	if err := store.Set("id", "12345"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := store.Get("id", false); got != "12345" {
		t.Fatalf("expected stored answer, got %q", got)
	}
	if ok := store.Verify("id", "12345", true); !ok {
		t.Fatal("expected verify success")
	}
	if got := store.Get("id", false); got != "" {
		t.Fatalf("expected cleared value after verify, got %q", got)
	}

	if err := store.Set("expire", "54321"); err != nil {
		t.Fatalf("Set expire: %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	if got := store.Get("expire", false); got != "" {
		t.Fatalf("expected expired value to be removed, got %q", got)
	}
}
