# Capability: task-scheduler-engine

## Purpose
Provides the core task scheduling engine supporting cron-based scheduled tasks and async queue-based tasks, with GORM-backed persistence, bounded concurrency, timeout/retry handling, and a pluggable store interface.

## Requirements

### Requirement: Store interface for pluggable backends
The scheduler SHALL define a `Store` interface abstracting all persistence operations (task state CRUD, execution queue, execution history, statistics). The default implementation SHALL use GORM (compatible with SQLite and PostgreSQL).

#### Scenario: GormStore initialization
- **WHEN** the scheduler engine is created with a GORM database instance
- **THEN** it SHALL construct a GormStore that implements the Store interface

#### Scenario: Future driver extensibility
- **WHEN** a developer implements the Store interface with a different backend (e.g., Redis)
- **THEN** the engine SHALL accept it without code changes to the engine itself

### Requirement: Task definition and registration
The scheduler SHALL provide a `TaskDef` struct for defining tasks in code, with fields: Name (unique identifier), Type (scheduled/async), Description, CronExpr (for scheduled), Timeout, MaxRetries, and Handler function. Tasks SHALL be registered via `engine.Register(*TaskDef)`.

#### Scenario: Register a scheduled task
- **WHEN** a developer calls `engine.Register()` with a TaskDef of type `scheduled` including a valid cron expression
- **THEN** the task SHALL be added to the registry and available for scheduling

#### Scenario: Register an async task
- **WHEN** a developer calls `engine.Register()` with a TaskDef of type `async`
- **THEN** the task SHALL be added to the registry and available for enqueuing

#### Scenario: Duplicate task name
- **WHEN** a developer registers two tasks with the same name
- **THEN** the engine SHALL panic at startup with a clear error message

### Requirement: Cron-based scheduled task dispatch
The scheduler SHALL use `robfig/cron/v3` to dispatch scheduled tasks according to their cron expressions. Each cron tick SHALL create a TaskExecution record and submit it to the executor.

#### Scenario: Cron tick fires
- **WHEN** a scheduled task's cron expression matches the current time
- **THEN** the engine SHALL create a TaskExecution with trigger=cron and status=pending, then submit it to the executor

#### Scenario: Paused task skipped
- **WHEN** a scheduled task is paused and its cron expression matches
- **THEN** the engine SHALL NOT create an execution or invoke the handler

### Requirement: Async task enqueuing
The scheduler SHALL provide `engine.Enqueue(name string, payload any) error` to enqueue async tasks for background execution. The payload SHALL be serialized to JSON and stored in the TaskExecution record.

#### Scenario: Enqueue an async task
- **WHEN** code calls `engine.Enqueue("export-users", ExportRequest{Format: "csv"})`
- **THEN** a TaskExecution record SHALL be created with status=pending, and the queue poller SHALL be notified via channel for immediate pickup

#### Scenario: Enqueue unknown task
- **WHEN** code calls `engine.Enqueue("nonexistent", nil)`
- **THEN** the method SHALL return an error indicating the task is not registered

### Requirement: Queue poller for async tasks
The scheduler SHALL run a background goroutine that polls the Store for pending async task executions every 3 seconds. Additionally, the poller SHALL wake immediately when notified via an internal channel (triggered by Enqueue).

#### Scenario: Poll picks up pending tasks
- **WHEN** the poller runs and finds pending TaskExecution records
- **THEN** it SHALL submit them to the executor in FIFO order (by created_at)

#### Scenario: No pending tasks
- **WHEN** the poller runs and finds no pending records
- **THEN** it SHALL sleep until the next poll interval or channel notification

### Requirement: Task executor with goroutine pool
The executor SHALL run task handlers in a bounded goroutine pool (max 5 concurrent workers). Each execution SHALL respect its task's Timeout via context deadline and support retry on failure.

#### Scenario: Successful execution
- **WHEN** a task handler completes without error
- **THEN** the TaskExecution status SHALL be updated to `completed` with duration recorded

#### Scenario: Handler returns error
- **WHEN** a task handler returns an error and retry_count < max_retries
- **THEN** the TaskExecution SHALL be re-enqueued with retry_count incremented

#### Scenario: Max retries exhausted
- **WHEN** a task handler fails and retry_count >= max_retries
- **THEN** the TaskExecution status SHALL be updated to `failed` with the error message recorded

#### Scenario: Handler exceeds timeout
- **WHEN** a task handler does not complete within its Timeout duration
- **THEN** the context SHALL be cancelled and the TaskExecution status SHALL be updated to `timeout`

#### Scenario: Concurrent execution limit
- **WHEN** 5 tasks are already executing and a new task is dequeued
- **THEN** the new task SHALL wait until a worker slot becomes available

### Requirement: Engine lifecycle management
The engine SHALL provide `Start()` and `Stop()` methods. Start SHALL sync task definitions to DB, start the cron scheduler and queue poller, and recover stale executions. Stop SHALL cease accepting new tasks, wait for running tasks to complete (with a timeout), and shut down all goroutines.

#### Scenario: Engine start
- **WHEN** `engine.Start()` is called
- **THEN** it SHALL sync registered TaskDefs to the task_states table, start robfig/cron entries for active scheduled tasks, start the queue poller goroutine, and mark any previously `running` executions as `stale`

#### Scenario: Engine graceful stop
- **WHEN** `engine.Stop()` is called
- **THEN** it SHALL stop the cron scheduler, stop the queue poller, wait up to 30 seconds for running tasks to finish, then return

### Requirement: Built-in history cleanup task
The engine SHALL register an internal scheduled task `scheduler-history-cleanup` (cron: `0 4 * * *`) that reads `scheduler.history_retention_days` from SystemConfig and deletes TaskExecution records older than the configured days. If the config value is 0 or missing, no cleanup SHALL occur.

#### Scenario: Cleanup runs with retention configured
- **WHEN** the cleanup task fires and `scheduler.history_retention_days` is set to 30
- **THEN** all TaskExecution records with created_at older than 30 days SHALL be deleted

#### Scenario: Cleanup runs with no config
- **WHEN** the cleanup task fires and `scheduler.history_retention_days` is not set or is 0
- **THEN** no records SHALL be deleted

### Requirement: TaskState database model
The system SHALL provide a `TaskState` model with fields: Name (PK, string), Type (string), Description (string), CronExpr (string, nullable), TimeoutMs (int), MaxRetries (int), Status (string: active/paused), UpdatedAt (timestamp). This table tracks runtime state of registered tasks.

#### Scenario: Table auto-migration
- **WHEN** the database is initialized
- **THEN** the `task_states` table SHALL be created via GORM AutoMigrate

### Requirement: TaskExecution database model
The system SHALL provide a `TaskExecution` model with fields: ID (auto-increment PK), TaskName (string, indexed), Trigger (string: cron/manual/api), Status (string: pending/running/completed/failed/timeout/stale), Payload (text, JSON), Result (text, JSON), Error (text), RetryCount (int), StartedAt (timestamp, nullable), FinishedAt (timestamp, nullable), CreatedAt (timestamp).

#### Scenario: Table auto-migration
- **WHEN** the database is initialized
- **THEN** the `task_executions` table SHALL be created via GORM AutoMigrate with an index on `task_name` and `status`
