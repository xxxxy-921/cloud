package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/channel"
	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/repository"
)

func newTestDBForMessageChannel(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(&model.MessageChannel{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return gdb
}

func seedMessageChannel(t *testing.T, db *gorm.DB, name, channelType, config string, enabled bool) *model.MessageChannel {
	t.Helper()
	ch := &model.MessageChannel{
		Name:    name,
		Type:    channelType,
		Config:  config,
		Enabled: enabled,
	}
	if err := db.Create(ch).Error; err != nil {
		t.Fatalf("seed message channel: %v", err)
	}
	return ch
}

type stubDriver struct {
	sendErr error
	testErr error
	sent    *channel.Payload
}

func (d *stubDriver) Send(config map[string]any, payload channel.Payload) error {
	d.sent = &payload
	return d.sendErr
}

func (d *stubDriver) Test(config map[string]any) error {
	return d.testErr
}

func newMessageChannelServiceForTest(t *testing.T, db *gorm.DB, stub *stubDriver) *MessageChannelService {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.Provide(injector, repository.NewMessageChannel)
	do.Provide(injector, NewMessageChannel)
	svc := do.MustInvoke[*MessageChannelService](injector)
	if stub != nil {
		svc.DriverResolver = func(string) (channel.Driver, error) {
			return stub, nil
		}
	} else {
		svc.DriverResolver = func(string) (channel.Driver, error) {
			return nil, errors.New("unsupported channel type")
		}
	}
	return svc
}

func TestMessageChannelServiceCreate_Success(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	svc := newMessageChannelServiceForTest(t, db, &stubDriver{})

	ch, err := svc.Create("SMTP", "email", `{"host":"localhost"}`)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ch.ID == 0 {
		t.Fatal("expected ID to be generated")
	}
	if ch.Type != "email" {
		t.Fatalf("expected type email, got %s", ch.Type)
	}
	if !ch.Enabled {
		t.Fatal("expected enabled to be true")
	}
}

func TestMessageChannelServiceCreate_InvalidType(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	svc := newMessageChannelServiceForTest(t, db, nil)

	_, err := svc.Create("SMTP", "unknown", `{}`)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}

	var count int64
	db.Model(&model.MessageChannel{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected no records created, got %d", count)
	}
}

func TestMessageChannelServiceGet_Success(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	svc := newMessageChannelServiceForTest(t, db, &stubDriver{})
	seeded := seedMessageChannel(t, db, "SMTP", "email", `{"password":"secret"}`, true)

	resp, err := svc.Get(seeded.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if resp.Name != "SMTP" {
		t.Fatalf("expected name SMTP, got %s", resp.Name)
	}
	if resp.Config != `{"password":"******"}` {
		t.Fatalf("expected masked password, got %s", resp.Config)
	}
}

func TestMessageChannelServiceGet_NotFound(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	svc := newMessageChannelServiceForTest(t, db, &stubDriver{})

	_, err := svc.Get(9999)
	if !errors.Is(err, ErrChannelNotFound) {
		t.Fatalf("expected ErrChannelNotFound, got %v", err)
	}
}

func TestMessageChannelServiceList(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	svc := newMessageChannelServiceForTest(t, db, &stubDriver{})
	seedMessageChannel(t, db, "A", "email", `{"password":"secret"}`, true)
	seedMessageChannel(t, db, "B", "email", `{}`, true)

	items, total, err := svc.List(repository.ListParams{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	configs := map[string]bool{items[0].Config: true, items[1].Config: true}
	if !configs[`{"password":"******"}`] || !configs[`{}`] {
		t.Fatalf("expected masked config in list, got %+v", items)
	}
}

func TestMessageChannelServiceUpdate_Success(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	svc := newMessageChannelServiceForTest(t, db, &stubDriver{})
	seeded := seedMessageChannel(t, db, "Old", "email", `{"host":"old","password":"secret"}`, true)

	resp, err := svc.Update(seeded.ID, "New", `{"host":"new","password":"changed"}`)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if resp.Name != "New" {
		t.Fatalf("expected name New, got %s", resp.Name)
	}

	stored, _ := svc.repo.FindByID(seeded.ID)
	if stored.Config != `{"host":"new","password":"changed"}` {
		t.Fatalf("expected config updated, got %s", stored.Config)
	}
}

func TestMessageChannelServiceUpdate_PreservesMaskedPassword(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	svc := newMessageChannelServiceForTest(t, db, &stubDriver{})
	seeded := seedMessageChannel(t, db, "Old", "email", `{"host":"old","password":"secret"}`, true)

	resp, err := svc.Update(seeded.ID, "New", `{"host":"new","password":"******"}`)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	stored, _ := svc.repo.FindByID(seeded.ID)
	if stored.Config != `{"host":"new","password":"secret"}` {
		t.Fatalf("expected original password preserved, got %s", stored.Config)
	}
	if resp.Config != `{"host":"new","password":"******"}` {
		t.Fatalf("expected masked password in response, got %s", resp.Config)
	}
}

func TestMessageChannelServiceUpdate_InvalidJSON(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	svc := newMessageChannelServiceForTest(t, db, &stubDriver{})
	seeded := seedMessageChannel(t, db, "Old", "email", `{"password":"secret"}`, true)

	_, err := svc.Update(seeded.ID, "New", `not-json`)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	stored, _ := svc.repo.FindByID(seeded.ID)
	if stored.Name != "Old" {
		t.Fatal("expected record to remain unchanged after failed update")
	}
}

func TestMessageChannelServiceDelete_Success(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	svc := newMessageChannelServiceForTest(t, db, &stubDriver{})
	seeded := seedMessageChannel(t, db, "ToDelete", "email", `{}`, true)

	if err := svc.Delete(seeded.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := svc.repo.FindByID(seeded.ID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected record not found after delete, got %v", err)
	}
}

func TestMessageChannelServiceDelete_NotFound(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	svc := newMessageChannelServiceForTest(t, db, &stubDriver{})

	err := svc.Delete(9999)
	if !errors.Is(err, ErrChannelNotFound) {
		t.Fatalf("expected ErrChannelNotFound, got %v", err)
	}
}

func TestMessageChannelServiceToggleEnabled(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	svc := newMessageChannelServiceForTest(t, db, &stubDriver{})
	seeded := seedMessageChannel(t, db, "Toggle", "email", `{}`, true)

	resp, err := svc.ToggleEnabled(seeded.ID)
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if resp.Enabled {
		t.Fatal("expected disabled after toggle")
	}
}

func TestMessageChannelServiceToggleEnabled_NotFound(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	svc := newMessageChannelServiceForTest(t, db, &stubDriver{})

	_, err := svc.ToggleEnabled(9999)
	if !errors.Is(err, ErrChannelNotFound) {
		t.Fatalf("expected ErrChannelNotFound, got %v", err)
	}
}

func TestMessageChannelServiceTestChannel_Success(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	stub := &stubDriver{}
	svc := newMessageChannelServiceForTest(t, db, stub)
	seeded := seedMessageChannel(t, db, "SMTP", "email", `{"host":"localhost"}`, true)

	if err := svc.TestChannel(seeded.ID); err != nil {
		t.Fatalf("test channel: %v", err)
	}
}

func TestMessageChannelServiceTestChannel_Failure(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	stub := &stubDriver{testErr: errors.New("auth failed")}
	svc := newMessageChannelServiceForTest(t, db, stub)
	seeded := seedMessageChannel(t, db, "SMTP", "email", `{"host":"localhost"}`, true)

	err := svc.TestChannel(seeded.ID)
	if err == nil || err.Error() != "auth failed" {
		t.Fatalf("expected auth failed error, got %v", err)
	}
}

func TestMessageChannelServiceTestChannel_InvalidConfig(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	stub := &stubDriver{}
	svc := newMessageChannelServiceForTest(t, db, stub)
	seeded := seedMessageChannel(t, db, "SMTP", "email", `not-json`, true)

	err := svc.TestChannel(seeded.ID)
	if err == nil {
		t.Fatal("expected invalid config error")
	}
}

func TestMessageChannelServiceSendTest_Success(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	stub := &stubDriver{}
	svc := newMessageChannelServiceForTest(t, db, stub)
	seeded := seedMessageChannel(t, db, "SMTP", "email", `{"host":"localhost"}`, true)

	if err := svc.SendTest(seeded.ID, []string{"to@example.com"}, "Subject", "Body"); err != nil {
		t.Fatalf("send test: %v", err)
	}
	if stub.sent == nil {
		t.Fatal("expected payload to be sent")
	}
	if stub.sent.Subject != "Subject" || stub.sent.Body != "Body" {
		t.Fatalf("unexpected payload: %+v", stub.sent)
	}
}

func TestMessageChannelServiceSendTest_NotFound(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	stub := &stubDriver{}
	svc := newMessageChannelServiceForTest(t, db, stub)

	err := svc.SendTest(9999, []string{"to@example.com"}, "Subject", "Body")
	if !errors.Is(err, ErrChannelNotFound) {
		t.Fatalf("expected ErrChannelNotFound, got %v", err)
	}
}

func TestMessageChannelServiceSend_SuccessAndDisabled(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	stub := &stubDriver{}
	svc := newMessageChannelServiceForTest(t, db, stub)
	enabled := seedMessageChannel(t, db, "SMTP", "email", `{"host":"localhost"}`, true)
	disabled := seedMessageChannel(t, db, "Disabled", "email", `{"host":"localhost"}`, false)
	if err := db.Model(&model.MessageChannel{}).Where("id = ?", disabled.ID).Update("enabled", false).Error; err != nil {
		t.Fatalf("disable channel: %v", err)
	}

	if err := svc.Send(enabled.ID, []string{"to@example.com"}, "Subject", "Body"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if stub.sent == nil || stub.sent.Subject != "Subject" {
		t.Fatalf("expected sent payload, got %+v", stub.sent)
	}

	err := svc.Send(disabled.ID, []string{"to@example.com"}, "Subject", "Body")
	if !errors.Is(err, ErrChannelDisabled) {
		t.Fatalf("expected ErrChannelDisabled, got %v", err)
	}
}

func TestMessageChannelServiceSend_InvalidConfig(t *testing.T) {
	db := newTestDBForMessageChannel(t)
	stub := &stubDriver{}
	svc := newMessageChannelServiceForTest(t, db, stub)
	seeded := seedMessageChannel(t, db, "SMTP", "email", `not-json`, true)

	err := svc.Send(seeded.ID, []string{"to@example.com"}, "Subject", "Body")
	if err == nil {
		t.Fatal("expected invalid config error")
	}
}
