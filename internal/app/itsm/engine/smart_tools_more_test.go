package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type smartToolConfigProvider struct {
	critical int
	warning  int
	limit    int
}

func (m smartToolConfigProvider) FallbackAssigneeID() uint                  { return 0 }
func (m smartToolConfigProvider) DecisionMode() string                      { return "ai_only" }
func (m smartToolConfigProvider) DecisionAgentID() uint                     { return 0 }
func (m smartToolConfigProvider) AuditLevel() string                        { return "full" }
func (m smartToolConfigProvider) SLACriticalThresholdSeconds() int          { return m.critical }
func (m smartToolConfigProvider) SLAWarningThresholdSeconds() int           { return m.warning }
func (m smartToolConfigProvider) SimilarHistoryLimit() int                  { return m.limit }
func (m smartToolConfigProvider) ParallelConvergenceTimeout() time.Duration { return time.Hour }

func TestDecisionToolResolveParticipantFiltersInactiveAndMissingUsers(t *testing.T) {
	def := toolResolveParticipant()

	raw, err := def.Handler(&decisionToolContext{
		ticketID: 42,
		resolver: &ParticipantResolver{},
		data: fakeDecisionDataProvider{
			resolveUserIDs: []uint{1, 2, 3},
			users: map[uint]*UserBasicInfo{
				1: {ID: 1, Username: "alice", IsActive: true},
				2: {ID: 2, Username: "bob", IsActive: false},
			},
			userErrors: map[uint]error{3: errors.New("missing")},
		},
	}, json.RawMessage(`{"type":"department","value":"ops"}`))
	if err != nil {
		t.Fatalf("resolve participant: %v", err)
	}

	var resp struct {
		OK         bool `json:"ok"`
		Status     string
		Count      int `json:"count"`
		Candidates []struct {
			UserID uint   `json:"user_id"`
			Name   string `json:"name"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal resolve participant: %v", err)
	}
	if !resp.OK || resp.Status != "resolved" || resp.Count != 1 {
		t.Fatalf("expected one resolved active candidate, got %+v", resp)
	}
	if len(resp.Candidates) != 1 || resp.Candidates[0].UserID != 1 || resp.Candidates[0].Name != "alice" {
		t.Fatalf("expected only active candidate alice, got %+v", resp.Candidates)
	}
}

func TestDecisionToolUserWorkloadAndSimilarHistoryExposeDecisionStats(t *testing.T) {
	workloadRaw, err := toolUserWorkload().Handler(&decisionToolContext{
		data: fakeDecisionDataProvider{
			users:         map[uint]*UserBasicInfo{7: {ID: 7, Username: "ops-admin", IsActive: true}},
			pendingCounts: map[uint]int64{7: 3},
		},
	}, json.RawMessage(`{"user_id":7}`))
	if err != nil {
		t.Fatalf("user workload: %v", err)
	}

	var workload struct {
		UserID            uint   `json:"user_id"`
		Name              string `json:"name"`
		IsActive          bool   `json:"is_active"`
		PendingActivities int64  `json:"pending_activities"`
	}
	if err := json.Unmarshal(workloadRaw, &workload); err != nil {
		t.Fatalf("unmarshal workload: %v", err)
	}
	if workload.UserID != 7 || workload.Name != "ops-admin" || !workload.IsActive || workload.PendingActivities != 3 {
		t.Fatalf("unexpected workload payload: %+v", workload)
	}

	now := time.Now()
	historyRaw, err := toolSimilarHistory().Handler(&decisionToolContext{
		ticketID:  99,
		serviceID: 5,
		configProvider: smartToolConfigProvider{
			limit: 2,
		},
		data: fakeDecisionDataProvider{
			similarHistory: []TicketHistoryRow{
				{ID: 11, Code: "T-11", Title: "VPN 变更", Status: TicketStatusCompleted, CreatedAt: now.Add(-5 * time.Hour), FinishedAt: ptrTime(now.Add(-2 * time.Hour))},
				{ID: 12, Code: "T-12", Title: "VPN 放通", Status: TicketStatusCompleted, CreatedAt: now.Add(-10 * time.Hour), FinishedAt: ptrTime(now.Add(-6 * time.Hour))},
			},
			ticketActivityCounts: map[uint]int64{11: 4, 12: 2},
			completedTicketCount: 9,
		},
	}, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("similar history: %v", err)
	}

	var history struct {
		Tickets []struct {
			Code                    string  `json:"code"`
			ResolutionDurationHours float64 `json:"resolution_duration_hours"`
			ActivityCount           int64   `json:"activity_count"`
		} `json:"tickets"`
		Stats struct {
			AvgResolutionHours float64 `json:"avg_resolution_hours"`
			TotalCount         int64   `json:"total_count"`
		} `json:"stats"`
	}
	if err := json.Unmarshal(historyRaw, &history); err != nil {
		t.Fatalf("unmarshal history: %v", err)
	}
	if len(history.Tickets) != 2 {
		t.Fatalf("expected 2 history tickets, got %+v", history.Tickets)
	}
	if history.Tickets[0].Code != "T-11" || history.Tickets[0].ResolutionDurationHours != 3.0 || history.Tickets[0].ActivityCount != 4 {
		t.Fatalf("unexpected first history ticket: %+v", history.Tickets[0])
	}
	if history.Tickets[1].Code != "T-12" || history.Tickets[1].ResolutionDurationHours != 4.0 || history.Tickets[1].ActivityCount != 2 {
		t.Fatalf("unexpected second history ticket: %+v", history.Tickets[1])
	}
	if history.Stats.AvgResolutionHours != 3.5 || history.Stats.TotalCount != 9 {
		t.Fatalf("unexpected history stats: %+v", history.Stats)
	}
}

func TestDecisionToolSLAStatusAndActionsExposeRuntimeState(t *testing.T) {
	now := time.Now()
	slaRaw, err := toolSLAStatus().Handler(&decisionToolContext{
		ticketID: 42,
		configProvider: smartToolConfigProvider{
			critical: 600,
			warning:  1800,
		},
		data: fakeDecisionDataProvider{
			slaData: &SLATicketData{
				SLAStatus:             "active",
				SLAResponseDeadline:   ptrTime(now.Add(20 * time.Minute)),
				SLAResolutionDeadline: ptrTime(now.Add(-10 * time.Minute)),
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("sla status: %v", err)
	}

	var sla struct {
		HasSLA    bool   `json:"has_sla"`
		SLAStatus string `json:"sla_status"`
		Urgency   string `json:"urgency"`
	}
	if err := json.Unmarshal(slaRaw, &sla); err != nil {
		t.Fatalf("unmarshal sla: %v", err)
	}
	if !sla.HasSLA || sla.SLAStatus != "active" || sla.Urgency != "breached" {
		t.Fatalf("expected breached sla payload, got %+v", sla)
	}

	actionListRaw, err := toolListActions().Handler(&decisionToolContext{
		ticketID: 42,
		serviceID: 5,
		data: fakeDecisionDataProvider{
			serviceActions: []ServiceActionRow{
				{ID: 3, Code: "precheck", Name: "预检", Description: "检查参数"},
				{ID: 4, Code: "apply", Name: "放通", Description: "执行放通"},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("list actions: %v", err)
	}

	var actionList struct {
		Count   int `json:"count"`
		Actions []struct {
			ID          uint   `json:"id"`
			Code        string `json:"code"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"actions"`
	}
	if err := json.Unmarshal(actionListRaw, &actionList); err != nil {
		t.Fatalf("unmarshal actions: %v", err)
	}
	if actionList.Count != 2 || len(actionList.Actions) != 2 || actionList.Actions[1].Code != "apply" {
		t.Fatalf("unexpected action list: %+v", actionList)
	}
}

func TestDecisionToolExecuteActionHonorsCachedAndGuardPaths(t *testing.T) {
	def := toolExecuteAction()

	t.Run("executor missing", func(t *testing.T) {
		raw, err := def.Handler(&decisionToolContext{}, json.RawMessage(`{"action_id":3}`))
		if err != nil {
			t.Fatalf("execute action without executor: %v", err)
		}
		assertToolErrorMessage(t, raw, "动作执行器不可用")
	})

	t.Run("inactive action rejected", func(t *testing.T) {
		raw, err := def.Handler(&decisionToolContext{
			ticketID:       42,
			serviceID:      5,
			ctx:            t.Context(),
			actionExecutor: &ActionExecutor{},
			data: fakeDecisionDataProvider{
				serviceActionByID: map[uint]*ServiceActionRow{
					3: {ID: 3, Code: "precheck", Name: "预检", IsActive: false},
				},
			},
		}, json.RawMessage(`{"action_id":3}`))
		if err != nil {
			t.Fatalf("execute inactive action: %v", err)
		}
		assertToolErrorMessage(t, raw, "动作已停用")
	})

	t.Run("missing action rejected", func(t *testing.T) {
		raw, err := def.Handler(&decisionToolContext{
			ticketID:       42,
			serviceID:      5,
			ctx:            t.Context(),
			actionExecutor: &ActionExecutor{},
			data:           fakeDecisionDataProvider{},
		}, json.RawMessage(`{"action_id":404}`))
		if err != nil {
			t.Fatalf("execute missing action: %v", err)
		}
		assertToolErrorMessage(t, raw, "动作不存在")
	})

	t.Run("db whitelist action rejects vague form window before execution", func(t *testing.T) {
		raw, err := def.Handler(&decisionToolContext{
			ticketID:           42,
			serviceID:          5,
			ctx:                t.Context(),
			collaborationSpec:  "数据库备份白名单申请必须先预检再放行，并由数据库管理员审核后执行放行",
			actionExecutor:     &ActionExecutor{},
			data: fakeDecisionDataProvider{
				ticket: &DecisionTicketData{
					FormData: `{"database_name":"prod-orders","source_ip":"10.0.0.8","whitelist_window":"今晚维护窗口","access_reason":"应急备份"}`,
				},
				serviceActionByID: map[uint]*ServiceActionRow{
					3: {ID: 3, Code: "db_backup_whitelist_precheck", Name: "预检", IsActive: true},
				},
			},
		}, json.RawMessage(`{"action_id":3}`))
		if err != nil {
			t.Fatalf("execute guarded whitelist action: %v", err)
		}
		assertToolErrorMessageContains(t, raw, "时间窗不明确")
	})

	t.Run("successful execution is served from cache", func(t *testing.T) {
		raw, err := def.Handler(&decisionToolContext{
			ticketID:       42,
			serviceID:      5,
			ctx:            t.Context(),
			actionExecutor: &ActionExecutor{},
			data: fakeDecisionDataProvider{
				serviceActionByID: map[uint]*ServiceActionRow{
					3: {ID: 3, Code: "precheck", Name: "预检", IsActive: true},
				},
				executed: []ExecutedActionInfo{
					{ActionName: "预检", ActionCode: "precheck", Status: "success"},
				},
			},
		}, json.RawMessage(`{"action_id":3}`))
		if err != nil {
			t.Fatalf("execute cached action: %v", err)
		}

		var resp struct {
			Success    bool   `json:"success"`
			ActionCode string `json:"action_code"`
			Cached     bool   `json:"cached"`
			Message    string `json:"message"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			t.Fatalf("unmarshal cached action: %v", err)
		}
		if !resp.Success || !resp.Cached || resp.ActionCode != "precheck" {
			t.Fatalf("unexpected cached response: %+v", resp)
		}
	})

	t.Run("executor failure returns business json payload", func(t *testing.T) {
		db := setupActionExecutorDB(t)
		if err := db.Create(&ticketModel{
			ID:          42,
			Code:        "TICK-EXEC-FAIL-TOOL",
			Status:      TicketStatusDecisioning,
			RequesterID: 9,
			PriorityID:  2,
		}).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
		}))
		defer server.Close()

		action := &ServiceActionRow{
			ID:         9,
			Code:       "notify",
			Name:       "通知",
			IsActive:   true,
			ActionType: "http",
			ConfigJSON: fmt.Sprintf(`{"url":%q,"method":"POST","body":"retry body","timeout":5,"retries":0}`, server.URL),
		}
		raw, err := def.Handler(&decisionToolContext{
			ctx:            context.Background(),
			ticketID:       42,
			serviceID:      5,
			actionExecutor: NewActionExecutor(db),
			data: fakeDecisionDataProvider{
				serviceActionByID: map[uint]*ServiceActionRow{9: action},
			},
		}, json.RawMessage(`{"action_id":9}`))
		if err != nil {
			t.Fatalf("execute action with executor failure: %v", err)
		}

		var resp struct {
			Success    bool   `json:"success"`
			ActionName string `json:"action_name"`
			ActionCode string `json:"action_code"`
			Error      string `json:"error"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			t.Fatalf("unmarshal failure response: %v", err)
		}
		if resp.Success || resp.ActionName != "通知" || resp.ActionCode != "notify" || !strings.Contains(resp.Error, "HTTP 500") {
			t.Fatalf("unexpected failure payload: %+v", resp)
		}
	})

	t.Run("live execution success persists and returns success payload", func(t *testing.T) {
		db := setupActionExecutorDB(t)
		if err := db.Create(&ticketModel{
			ID:          43,
			Code:        "TICK-EXEC-OK-TOOL",
			Status:      TicketStatusDecisioning,
			RequesterID: 7,
			PriorityID:  3,
			FormData:    `{"env":"prod"}`,
		}).Error; err != nil {
			t.Fatalf("create ticket: %v", err)
		}

		requests := make(chan string, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			requests <- string(body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		defer server.Close()

		action := &ServiceActionRow{
			ID:         10,
			Code:       "notify",
			Name:       "通知",
			IsActive:   true,
			ActionType: "http",
			ConfigJSON: fmt.Sprintf(`{
				"url": %q,
				"method": "POST",
				"body": {"ticket":"{{ticket.code}}","env":"{{ticket.form_data.env}}"},
				"timeout": 5,
				"retries": 0
			}`, server.URL),
		}
		raw, err := def.Handler(&decisionToolContext{
			ctx:            context.Background(),
			ticketID:       43,
			serviceID:      5,
			actionExecutor: NewActionExecutor(db),
			data: fakeDecisionDataProvider{
				serviceActionByID: map[uint]*ServiceActionRow{10: action},
			},
		}, json.RawMessage(`{"action_id":10}`))
		if err != nil {
			t.Fatalf("execute action success path: %v", err)
		}

		body := <-requests
		if !strings.Contains(body, `"ticket":"TICK-EXEC-OK-TOOL"`) || !strings.Contains(body, `"env":"prod"`) {
			t.Fatalf("unexpected request body: %s", body)
		}

		var resp struct {
			Success    bool   `json:"success"`
			ActionName string `json:"action_name"`
			ActionCode string `json:"action_code"`
			Message    string `json:"message"`
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			t.Fatalf("unmarshal success response: %v", err)
		}
		if !resp.Success || resp.ActionName != "通知" || resp.ActionCode != "notify" || !strings.Contains(resp.Message, "执行成功") {
			t.Fatalf("unexpected success payload: %+v", resp)
		}

		var rows []actionExecutionModel
		if err := db.Order("id asc").Find(&rows).Error; err != nil {
			t.Fatalf("list execution rows: %v", err)
		}
		if len(rows) != 1 || rows[0].Status != "success" {
			t.Fatalf("unexpected execution rows: %+v", rows)
		}
	})
}

func assertToolErrorMessage(t *testing.T, raw []byte, want string) {
	t.Helper()
	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal tool error: %v", err)
	}
	if !resp.Error || resp.Message != want {
		t.Fatalf("tool error = %+v, want %q", resp, want)
	}
}

func assertToolErrorMessageContains(t *testing.T, raw []byte, want string) {
	t.Helper()
	var resp struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal tool error: %v", err)
	}
	if !resp.Error || !strings.Contains(resp.Message, want) {
		t.Fatalf("tool error = %+v, want substring %q", resp, want)
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
