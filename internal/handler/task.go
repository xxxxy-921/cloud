package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"metis/internal/scheduler"
)

type TaskHandler struct {
	engine *scheduler.Engine
}

func (h *TaskHandler) ListTasks(c *gin.Context) {
	taskType := c.Query("type")
	ctx := c.Request.Context()

	states, err := h.engine.GetStore().ListTaskStates(ctx, taskType)
	if err != nil {
		Fail(c, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	registry := h.engine.GetRegistry()
	var infos []scheduler.TaskInfo
	for _, state := range states {
		info := scheduler.TaskInfo{TaskState: *state}
		// Attach last execution summary
		if last, err := h.engine.GetStore().GetLastExecution(ctx, state.Name); err == nil {
			var duration int64
			if last.StartedAt != nil && last.FinishedAt != nil {
				duration = last.FinishedAt.Sub(*last.StartedAt).Milliseconds()
			}
			info.LastExecution = &scheduler.LastExecution{
				Timestamp: last.CreatedAt,
				Status:    last.Status,
				Duration:  duration,
			}
		}
		// Only include tasks that are still in the registry
		if _, ok := registry[state.Name]; ok {
			infos = append(infos, info)
		}
	}

	OK(c, infos)
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	name := c.Param("name")
	ctx := c.Request.Context()

	state, err := h.engine.GetStore().GetTaskState(ctx, name)
	if err != nil {
		Fail(c, http.StatusNotFound, "task not found")
		return
	}

	info := scheduler.TaskInfo{TaskState: *state}
	if last, err := h.engine.GetStore().GetLastExecution(ctx, name); err == nil {
		var duration int64
		if last.StartedAt != nil && last.FinishedAt != nil {
			duration = last.FinishedAt.Sub(*last.StartedAt).Milliseconds()
		}
		info.LastExecution = &scheduler.LastExecution{
			Timestamp: last.CreatedAt,
			Status:    last.Status,
			Duration:  duration,
		}
	}

	// Get recent executions
	execs, _, _ := h.engine.GetStore().ListExecutions(ctx, scheduler.ExecutionFilter{
		TaskName: name, Page: 1, PageSize: 20,
	})

	OK(c, gin.H{
		"task":             info,
		"recentExecutions": execs,
	})
}

func (h *TaskHandler) ListExecutions(c *gin.Context) {
	name := c.Param("name")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	execs, total, err := h.engine.GetStore().ListExecutions(c.Request.Context(), scheduler.ExecutionFilter{
		TaskName: name,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		Fail(c, http.StatusInternalServerError, "failed to list executions")
		return
	}

	OK(c, gin.H{
		"list":     execs,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *TaskHandler) GetStats(c *gin.Context) {
	stats, err := h.engine.GetStore().Stats(c.Request.Context())
	if err != nil {
		Fail(c, http.StatusInternalServerError, "failed to get stats")
		return
	}
	OK(c, stats)
}

func (h *TaskHandler) PauseTask(c *gin.Context) {
	name := c.Param("name")
	if err := h.engine.PauseTask(name); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OK(c, nil)
}

func (h *TaskHandler) ResumeTask(c *gin.Context) {
	name := c.Param("name")
	if err := h.engine.ResumeTask(name); err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OK(c, nil)
}

func (h *TaskHandler) TriggerTask(c *gin.Context) {
	name := c.Param("name")
	exec, err := h.engine.TriggerTask(name)
	if err != nil {
		Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	OK(c, gin.H{"executionId": exec.ID})
}
