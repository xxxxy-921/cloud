package service

import (
	"sync"
	"time"

	"github.com/mojocn/base64Captcha"
	"github.com/samber/do/v2"
)

// CaptchaService generates and verifies image captchas.
type CaptchaService struct {
	store   *captchaStore
	driver  *base64Captcha.DriverDigit
	captcha *base64Captcha.Captcha
}

func NewCaptcha(i do.Injector) (*CaptchaService, error) {
	store := newCaptchaStore(5*time.Minute, 1*time.Minute)
	driver := base64Captcha.NewDriverDigit(80, 240, 5, 0.7, 80)
	c := base64Captcha.NewCaptcha(driver, store)
	return &CaptchaService{store: store, driver: driver, captcha: c}, nil
}

// CaptchaResult holds a generated captcha's ID and base64-encoded image.
type CaptchaResult struct {
	ID    string `json:"id"`
	Image string `json:"image"`
}

// Generate creates a new captcha and returns its ID + base64 image.
func (s *CaptchaService) Generate() (*CaptchaResult, error) {
	id, b64s, _, err := s.captcha.Generate()
	if err != nil {
		return nil, err
	}
	return &CaptchaResult{ID: id, Image: b64s}, nil
}

// Verify checks the answer for the given captcha ID. The captcha is consumed on verify.
func (s *CaptchaService) Verify(id, answer string) bool {
	return s.store.Verify(id, answer, true)
}

// captchaStore implements base64Captcha.Store using sync.Map with TTL.
type captchaStore struct {
	data    sync.Map
	ttl     time.Duration
	stopped chan struct{}
}

type captchaEntry struct {
	answer    string
	expiresAt time.Time
}

func newCaptchaStore(ttl, cleanupInterval time.Duration) *captchaStore {
	s := &captchaStore{ttl: ttl, stopped: make(chan struct{})}
	go s.cleanup(cleanupInterval)
	return s
}

func (s *captchaStore) Set(id string, value string) error {
	s.data.Store(id, captchaEntry{answer: value, expiresAt: time.Now().Add(s.ttl)})
	return nil
}

func (s *captchaStore) Get(id string, clear bool) string {
	v, ok := s.data.Load(id)
	if !ok {
		return ""
	}
	entry := v.(captchaEntry)
	if time.Now().After(entry.expiresAt) {
		s.data.Delete(id)
		return ""
	}
	if clear {
		s.data.Delete(id)
	}
	return entry.answer
}

func (s *captchaStore) Verify(id, answer string, clear bool) bool {
	stored := s.Get(id, clear)
	return stored != "" && stored == answer
}

func (s *captchaStore) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			s.data.Range(func(key, value any) bool {
				if entry := value.(captchaEntry); now.After(entry.expiresAt) {
					s.data.Delete(key)
				}
				return true
			})
		case <-s.stopped:
			return
		}
	}
}
