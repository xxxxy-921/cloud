package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"metis/internal/database"
	"metis/internal/model"
	"metis/internal/scheduler"
)

func newTestDBForTask(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(&model.TaskState{}, &model.TaskExecution{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return gdb
}

func newTaskServiceForTest(t *testing.T, db *gorm.DB) *TaskService {
	t.Helper()
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	engine, err := scheduler.New(injector)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	engine.Register(&scheduler.TaskDef{
		Name:     "test-task",
		Type:     scheduler.TypeScheduled,
		CronExpr: "0 0 * * *",
		Handler:  func(ctx context.Context, payload json.RawMessage) error { return nil },
	})
	engine.Register(&scheduler.TaskDef{
		Name:    "test-async",
		Type:    scheduler.TypeAsync,
		Handler: func(ctx context.Context, payload json.RawMessage) error { return nil },
	})
	return &TaskService{store: engine.GetStore(), engine: engine}
}

func seedTaskState(t *testing.T, db *gorm.DB, state *model.TaskState) {
	t.Helper()
	if err := db.Save(state).Error; err != nil {
		t.Fatalf("seed task state: %v", err)
	}
}

func seedTaskExecution(t *testing.T, db *gorm.DB, exec *model.TaskExecution) {
	t.Helper()
	if err := db.Save(exec).Error; err != nil {
		t.Fatalf("seed task execution: %v", err)
	}
}

func TestNewTask_ConstructsService(t *testing.T) {
	db := newTestDBForTask(t)
	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	engine, err := scheduler.New(injector)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	do.ProvideValue(injector, engine)

	svc, err := NewTask(injector)
	if err != nil {
		t.Fatalf("NewTask returned error: %v", err)
	}
	if svc.engine == nil || svc.store == nil {
		t.Fatalf("expected task service dependencies to be wired, got %+v", svc)
	}
}

func TestTaskServiceListTasks_WithTypeFilter(t *testing.T) {
	db := newTestDBForTask(t)
	svc := newTaskServiceForTest(t, db)
	ctx := context.Background()

	seedTaskState(t, db, &model.TaskState{Name: "test-task", Type: scheduler.TypeScheduled, Status: scheduler.StatusActive, CronExpr: "0 0 * * *"})
	seedTaskState(t, db, &model.TaskState{Name: "test-async", Type: scheduler.TypeAsync, Status: scheduler.StatusActive})

	infos, err := svc.ListTasks(ctx, scheduler.TypeScheduled)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 scheduled task, got %d", len(infos))
	}
	if infos[0].Name != "test-task" {
		t.Fatalf("expected test-task, got %s", infos[0].Name)
	}
}

func TestTaskServiceListTasks_WithoutFilter(t *testing.T) {
	db := newTestDBForTask(t)
	svc := newTaskServiceForTest(t, db)
	ctx := context.Background()

	seedTaskState(t, db, &model.TaskState{Name: "test-task", Type: scheduler.TypeScheduled, Status: scheduler.StatusActive, CronExpr: "0 0 * * *"})
	seedTaskState(t, db, &model.TaskState{Name: "test-async", Type: scheduler.TypeAsync, Status: scheduler.StatusActive})

	infos, err := svc.ListTasks(ctx, "")
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(infos))
	}
}

func TestTaskServiceListTasks_AttachesLastExecution(t *testing.T) {
	db := newTestDBForTask(t)
	svc := newTaskServiceForTest(t, db)
	ctx := context.Background()

	seedTaskState(t, db, &model.TaskState{Name: "test-task", Type: scheduler.TypeScheduled, Status: scheduler.StatusActive, CronExpr: "0 0 * * *"})
	started := time.Now().Add(-5 * time.Minute)
	finished := time.Now().Add(-4 * time.Minute)
	seedTaskExecution(t, db, &model.TaskExecution{TaskName: "test-task", Status: scheduler.ExecCompleted, StartedAt: &started, FinishedAt: &finished, CreatedAt: time.Now()})

	infos, err := svc.ListTasks(ctx, "")
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 task, got %d", len(infos))
	}
	if infos[0].LastExecution == nil {
		t.Fatal("expected LastExecution to be attached")
	}
	if infos[0].LastExecution.Status != scheduler.ExecCompleted {
		t.Fatalf("expected status completed, got %s", infos[0].LastExecution.Status)
	}
	expectedDuration := finished.Sub(started).Milliseconds()
	if infos[0].LastExecution.Duration != expectedDuration {
		t.Fatalf("expected duration %d, got %d", expectedDuration, infos[0].LastExecution.Duration)
	}
}

func TestTaskServiceGetTask_WithRecentExecutions(t *testing.T) {
	db := newTestDBForTask(t)
	svc := newTaskServiceForTest(t, db)
	ctx := context.Background()

	seedTaskState(t, db, &model.TaskState{Name: "test-task", Type: scheduler.TypeScheduled, Status: scheduler.StatusActive, CronExpr: "0 0 * * *"})
	for i := 0; i < 3; i++ {
		seedTaskExecution(t, db, &model.TaskExecution{TaskName: "test-task", Status: scheduler.ExecCompleted, CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour)})
	}

	info, execs, err := svc.GetTask(ctx, "test-task")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if info == nil || info.Name != "test-task" {
		t.Fatalf("unexpected task info")
	}
	if len(execs) != 3 {
		t.Fatalf("expected 3 recent executions, got %d", len(execs))
	}
}

func TestTaskServiceGetTask_NotFound(t *testing.T) {
	db := newTestDBForTask(t)
	svc := newTaskServiceForTest(t, db)
	ctx := context.Background()

	info, execs, err := svc.GetTask(ctx, "missing-task")
	if err == nil {
		t.Fatal("expected missing task error")
	}
	if info != nil || execs != nil {
		t.Fatalf("expected nil task result on error, got info=%v execs=%v", info, execs)
	}
}

func TestTaskServiceListExecutions_Pagination(t *testing.T) {
	db := newTestDBForTask(t)
	svc := newTaskServiceForTest(t, db)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		seedTaskExecution(t, db, &model.TaskExecution{TaskName: "test-task", Status: scheduler.ExecCompleted, CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute)})
	}

	execs, total, err := svc.ListExecutions(ctx, "test-task", 1, 10)
	if err != nil {
		t.Fatalf("list executions: %v", err)
	}
	if total != 25 {
		t.Fatalf("expected total 25, got %d", total)
	}
	if len(execs) != 10 {
		t.Fatalf("expected 10 executions, got %d", len(execs))
	}
}

func TestTaskServiceGetStats_ReflectsQueue(t *testing.T) {
	db := newTestDBForTask(t)
	svc := newTaskServiceForTest(t, db)
	ctx := context.Background()

	seedTaskState(t, db, &model.TaskState{Name: "test-task", Type: scheduler.TypeScheduled, Status: scheduler.StatusActive, CronExpr: "0 0 * * *"})
	seedTaskState(t, db, &model.TaskState{Name: "test-async", Type: scheduler.TypeAsync, Status: scheduler.StatusActive})

	seedTaskExecution(t, db, &model.TaskExecution{TaskName: "test-task", Status: scheduler.ExecPending, CreatedAt: time.Now()})
	seedTaskExecution(t, db, &model.TaskExecution{TaskName: "test-task", Status: scheduler.ExecRunning, CreatedAt: time.Now()})
	now := time.Now()
	seedTaskExecution(t, db, &model.TaskExecution{TaskName: "test-task", Status: scheduler.ExecCompleted, CreatedAt: time.Now(), FinishedAt: &now})
	seedTaskExecution(t, db, &model.TaskExecution{TaskName: "test-task", Status: scheduler.ExecFailed, CreatedAt: time.Now()})

	stats, err := svc.GetStats(ctx)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if stats.TotalTasks != 2 {
		t.Fatalf("expected TotalTasks=2, got %d", stats.TotalTasks)
	}
	if stats.Pending != 1 {
		t.Fatalf("expected Pending=1, got %d", stats.Pending)
	}
	if stats.Running != 1 {
		t.Fatalf("expected Running=1, got %d", stats.Running)
	}
	if stats.CompletedToday != 1 {
		t.Fatalf("expected CompletedToday=1, got %d", stats.CompletedToday)
	}
	if stats.FailedToday != 1 {
		t.Fatalf("expected FailedToday=1, got %d", stats.FailedToday)
	}
}

func TestTaskServicePauseTask_UpdatesState(t *testing.T) {
	db := newTestDBForTask(t)
	svc := newTaskServiceForTest(t, db)
	ctx := context.Background()

	seedTaskState(t, db, &model.TaskState{Name: "test-task", Type: scheduler.TypeScheduled, Status: scheduler.StatusActive, CronExpr: "0 0 * * *"})

	if err := svc.PauseTask("test-task"); err != nil {
		t.Fatalf("pause task: %v", err)
	}

	state, err := svc.store.GetTaskState(ctx, "test-task")
	if err != nil {
		t.Fatalf("get task state: %v", err)
	}
	if state.Status != scheduler.StatusPaused {
		t.Fatalf("expected status paused, got %s", state.Status)
	}
}

func TestTaskServiceResumeTask_UpdatesState(t *testing.T) {
	db := newTestDBForTask(t)
	svc := newTaskServiceForTest(t, db)
	ctx := context.Background()

	seedTaskState(t, db, &model.TaskState{Name: "test-task", Type: scheduler.TypeScheduled, Status: scheduler.StatusPaused, CronExpr: "0 0 * * *"})

	if err := svc.ResumeTask("test-task"); err != nil {
		t.Fatalf("resume task: %v", err)
	}

	state, err := svc.store.GetTaskState(ctx, "test-task")
	if err != nil {
		t.Fatalf("get task state: %v", err)
	}
	if state.Status != scheduler.StatusActive {
		t.Fatalf("expected status active, got %s", state.Status)
	}
}

func TestTaskServiceTriggerTask_EnqueuesExecution(t *testing.T) {
	db := newTestDBForTask(t)
	svc := newTaskServiceForTest(t, db)
	ctx := context.Background()

	exec, err := svc.TriggerTask("test-task")
	if err != nil {
		t.Fatalf("trigger task: %v", err)
	}
	if exec == nil {
		t.Fatal("expected execution, got nil")
	}
	if exec.Status != scheduler.ExecPending {
		t.Fatalf("expected status pending, got %s", exec.Status)
	}
	if exec.Trigger != scheduler.TriggerManual {
		t.Fatalf("expected trigger manual, got %s", exec.Trigger)
	}

	stored, err := svc.store.GetExecution(ctx, exec.ID)
	if err != nil {
		t.Fatalf("get execution: %v", err)
	}
	if stored.Status != scheduler.ExecPending {
		t.Fatalf("expected stored status pending, got %s", stored.Status)
	}
}

func TestTaskServicePauseTask_PreventsDoublePause(t *testing.T) {
	db := newTestDBForTask(t)
	svc := newTaskServiceForTest(t, db)

	seedTaskState(t, db, &model.TaskState{Name: "test-task", Type: scheduler.TypeScheduled, Status: scheduler.StatusPaused, CronExpr: "0 0 * * *"})

	if err := svc.PauseTask("test-task"); err == nil {
		t.Fatal("expected error for double pause, got nil")
	}
}

func TestTaskServiceResumeTask_PreventsDoubleResume(t *testing.T) {
	db := newTestDBForTask(t)
	svc := newTaskServiceForTest(t, db)

	seedTaskState(t, db, &model.TaskState{Name: "test-task", Type: scheduler.TypeScheduled, Status: scheduler.StatusActive, CronExpr: "0 0 * * *"})

	if err := svc.ResumeTask("test-task"); err == nil {
		t.Fatal("expected error for double resume, got nil")
	}
}
