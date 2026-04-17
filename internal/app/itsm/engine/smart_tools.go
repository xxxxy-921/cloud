package engine

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"

	"metis/internal/llm"
)

// decisionToolDef defines a decision domain tool with its LLM definition and handler.
type decisionToolDef struct {
	Def     llm.ToolDef
	Handler func(ctx *decisionToolContext, args json.RawMessage) (json.RawMessage, error)
}

// decisionToolContext holds the shared context for all decision tool executions.
type decisionToolContext struct {
	tx                *gorm.DB
	ticketID          uint
	serviceID         uint
	knowledgeSearcher KnowledgeSearcher
	resolver          *ParticipantResolver
	knowledgeBaseIDs  []uint
}

// allDecisionTools returns the complete set of decision domain tools.
func allDecisionTools() []decisionToolDef {
	return []decisionToolDef{
		toolTicketContext(),
		toolKnowledgeSearch(),
		toolResolveParticipant(),
		toolUserWorkload(),
		toolSimilarHistory(),
		toolSLAStatus(),
		toolListActions(),
	}
}

// buildDecisionToolDefs extracts llm.ToolDef list from all decision tools.
func buildDecisionToolDefs() []llm.ToolDef {
	tools := allDecisionTools()
	defs := make([]llm.ToolDef, len(tools))
	for i, t := range tools {
		defs[i] = t.Def
	}
	return defs
}

// --- Tool: decision.ticket_context ---

func toolTicketContext() decisionToolDef {
	return decisionToolDef{
		Def: llm.ToolDef{
			Name:        "decision.ticket_context",
			Description: "查询工单的完整上下文信息，包括表单数据、SLA 状态、活动历史和当前指派",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		Handler: func(ctx *decisionToolContext, _ json.RawMessage) (json.RawMessage, error) {
			var fullTicket struct {
				Code                  string
				Title                 string
				Description           string
				Status                string
				Source                string
				FormData              string
				SLAResponseDeadline   *time.Time
				SLAResolutionDeadline *time.Time
			}
			if err := ctx.tx.Table("itsm_tickets").Where("id = ?", ctx.ticketID).
				Select("code, title, description, status, source, form_data, sla_response_deadline, sla_resolution_deadline").
				First(&fullTicket).Error; err != nil {
				return toolError("工单不存在")
			}

			result := map[string]any{
				"code":        fullTicket.Code,
				"title":       fullTicket.Title,
				"description": fullTicket.Description,
				"status":      fullTicket.Status,
				"source":      fullTicket.Source,
			}

			// Form data
			if fullTicket.FormData != "" {
				result["form_data"] = json.RawMessage(fullTicket.FormData)
			}

			// SLA status
			now := time.Now()
			if fullTicket.SLAResponseDeadline != nil || fullTicket.SLAResolutionDeadline != nil {
				sla := map[string]any{}
				if fullTicket.SLAResponseDeadline != nil {
					sla["response_remaining_seconds"] = int64(fullTicket.SLAResponseDeadline.Sub(now).Seconds())
				}
				if fullTicket.SLAResolutionDeadline != nil {
					sla["resolution_remaining_seconds"] = int64(fullTicket.SLAResolutionDeadline.Sub(now).Seconds())
				}
				result["sla_status"] = sla
			}

			// Activity history
			var activities []activityModel
			ctx.tx.Where("ticket_id = ? AND status = ?", ctx.ticketID, ActivityCompleted).
				Order("id ASC").Find(&activities)

			var history []map[string]any
			for _, a := range activities {
				entry := map[string]any{
					"type":    a.ActivityType,
					"name":    a.Name,
					"outcome": a.TransitionOutcome,
				}
				if a.FinishedAt != nil {
					entry["completed_at"] = a.FinishedAt.Format(time.RFC3339)
				}
				if a.AIReasoning != "" {
					entry["ai_reasoning"] = a.AIReasoning
				}
				history = append(history, entry)
			}
			result["activity_history"] = history

			// Current assignment
			var assignment struct {
				AssigneeID *uint
			}
			if err := ctx.tx.Table("itsm_ticket_assignments").
				Where("ticket_id = ? AND is_current = ?", ctx.ticketID, true).
				Select("assignee_id").First(&assignment).Error; err == nil && assignment.AssigneeID != nil {
				var user struct {
					Username string
				}
				ctx.tx.Table("users").Where("id = ?", *assignment.AssigneeID).Select("username").First(&user)
				result["current_assignment"] = map[string]any{
					"assignee_id":   *assignment.AssigneeID,
					"assignee_name": user.Username,
				}
			}

			return json.Marshal(result)
		},
	}
}

// --- Tool: decision.knowledge_search ---

func toolKnowledgeSearch() decisionToolDef {
	return decisionToolDef{
		Def: llm.ToolDef{
			Name:        "decision.knowledge_search",
			Description: "搜索服务关联的知识库，返回与查询相关的知识片段",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "搜索查询文本"},
					"limit": map[string]any{"type": "integer", "description": "返回结果数量上限（默认 3）"},
				},
				"required": []string{"query"},
			},
		},
		Handler: func(ctx *decisionToolContext, args json.RawMessage) (json.RawMessage, error) {
			var params struct {
				Query string `json:"query"`
				Limit int    `json:"limit"`
			}
			json.Unmarshal(args, &params)
			if params.Limit <= 0 {
				params.Limit = 3
			}

			if ctx.knowledgeSearcher == nil {
				return json.Marshal(map[string]any{
					"results": []any{},
					"count":   0,
					"message": "知识搜索不可用",
				})
			}

			if len(ctx.knowledgeBaseIDs) == 0 {
				return json.Marshal(map[string]any{
					"results": []any{},
					"count":   0,
				})
			}

			results, err := ctx.knowledgeSearcher.Search(ctx.knowledgeBaseIDs, params.Query, params.Limit)
			if err != nil {
				return toolError(fmt.Sprintf("知识搜索失败: %v", err))
			}

			items := make([]map[string]any, len(results))
			for i, r := range results {
				items[i] = map[string]any{
					"title":   r.Title,
					"content": r.Content,
					"score":   r.Score,
				}
			}
			return json.Marshal(map[string]any{
				"results": items,
				"count":   len(items),
			})
		},
	}
}

// --- Tool: decision.resolve_participant ---

func toolResolveParticipant() decisionToolDef {
	return decisionToolDef{
		Def: llm.ToolDef{
			Name:        "decision.resolve_participant",
			Description: "按参与人类型解析出具体用户。支持 user/position/department/position_department/requester_manager",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type":            map[string]any{"type": "string", "description": "参与人类型: user|position|department|position_department|requester_manager"},
					"value":           map[string]any{"type": "string", "description": "类型相关值（user类型为user_id, position类型为position_code等）"},
					"position_code":   map[string]any{"type": "string", "description": "岗位代码（position_department类型时必填）"},
					"department_code": map[string]any{"type": "string", "description": "部门代码（position_department类型时必填）"},
				},
				"required": []string{"type"},
			},
		},
		Handler: func(ctx *decisionToolContext, args json.RawMessage) (json.RawMessage, error) {
			if ctx.resolver == nil {
				return toolError("参与人解析器不可用")
			}

			userIDs, err := ctx.resolver.ResolveForTool(ctx.tx, ctx.ticketID, args)
			if err != nil {
				return toolError(fmt.Sprintf("参与人解析失败: %v", err))
			}

			// Enrich with user details
			var candidates []map[string]any
			for _, uid := range userIDs {
				var user struct {
					ID       uint
					Username string
					IsActive bool
				}
				if err := ctx.tx.Table("users").Where("id = ?", uid).
					Select("id, username, is_active").First(&user).Error; err != nil {
					continue
				}
				if !user.IsActive {
					continue
				}
				candidates = append(candidates, map[string]any{
					"user_id": user.ID,
					"name":    user.Username,
				})
			}

			return json.Marshal(map[string]any{
				"candidates": candidates,
				"count":      len(candidates),
			})
		},
	}
}

// --- Tool: decision.user_workload ---

func toolUserWorkload() decisionToolDef {
	return decisionToolDef{
		Def: llm.ToolDef{
			Name:        "decision.user_workload",
			Description: "查询指定用户当前的工单负载信息（待处理活动数），帮助做出负载均衡的指派决策",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"user_id": map[string]any{"type": "integer", "description": "用户 ID"},
				},
				"required": []string{"user_id"},
			},
		},
		Handler: func(ctx *decisionToolContext, args json.RawMessage) (json.RawMessage, error) {
			var params struct {
				UserID uint `json:"user_id"`
			}
			json.Unmarshal(args, &params)

			var user struct {
				ID       uint
				Username string
				IsActive bool
			}
			if err := ctx.tx.Table("users").Where("id = ?", params.UserID).
				Select("id, username, is_active").First(&user).Error; err != nil {
				return toolError("用户不存在")
			}

			// Count pending activities assigned to this user
			var pendingCount int64
			ctx.tx.Table("itsm_ticket_assignments").
				Joins("JOIN itsm_ticket_activities ON itsm_ticket_activities.id = itsm_ticket_assignments.activity_id").
				Where("itsm_ticket_assignments.assignee_id = ? AND itsm_ticket_activities.status IN ?",
					params.UserID, []string{ActivityPending, ActivityInProgress}).
				Count(&pendingCount)

			return json.Marshal(map[string]any{
				"user_id":            user.ID,
				"name":              user.Username,
				"is_active":         user.IsActive,
				"pending_activities": pendingCount,
			})
		},
	}
}

// --- Tool: decision.similar_history ---

func toolSimilarHistory() decisionToolDef {
	return decisionToolDef{
		Def: llm.ToolDef{
			Name:        "decision.similar_history",
			Description: "查询同一服务下已完成工单的处理模式，提供历史参考（平均耗时、常见审批人等）",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit": map[string]any{"type": "integer", "description": "返回工单数量上限（默认 5）"},
				},
			},
		},
		Handler: func(ctx *decisionToolContext, args json.RawMessage) (json.RawMessage, error) {
			var params struct {
				Limit int `json:"limit"`
			}
			json.Unmarshal(args, &params)
			if params.Limit <= 0 {
				params.Limit = 5
			}

			type historyRow struct {
				ID         uint
				Code       string
				Title      string
				Status     string
				CreatedAt  time.Time
				FinishedAt *time.Time
			}
			var rows []historyRow
			ctx.tx.Table("itsm_tickets").
				Where("service_id = ? AND status = ? AND id != ? AND deleted_at IS NULL",
					ctx.serviceID, "completed", ctx.ticketID).
				Select("id, code, title, status, created_at, finished_at").
				Order("finished_at DESC").
				Limit(params.Limit).
				Find(&rows)

			var tickets []map[string]any
			var totalHours float64
			for _, r := range rows {
				entry := map[string]any{
					"code":   r.Code,
					"title":  r.Title,
					"status": r.Status,
				}

				if r.FinishedAt != nil {
					hours := r.FinishedAt.Sub(r.CreatedAt).Hours()
					entry["resolution_duration_hours"] = math.Round(hours*10) / 10
					totalHours += hours
				}

				// Count activities
				var actCount int64
				ctx.tx.Table("itsm_ticket_activities").Where("ticket_id = ?", r.ID).Count(&actCount)
				entry["activity_count"] = actCount

				tickets = append(tickets, entry)
			}

			// Aggregate stats
			var totalCount int64
			ctx.tx.Table("itsm_tickets").
				Where("service_id = ? AND status = ? AND deleted_at IS NULL", ctx.serviceID, "completed").
				Count(&totalCount)

			avgHours := 0.0
			if len(rows) > 0 {
				avgHours = math.Round(totalHours/float64(len(rows))*10) / 10
			}

			return json.Marshal(map[string]any{
				"tickets": tickets,
				"stats": map[string]any{
					"avg_resolution_hours": avgHours,
					"total_count":          totalCount,
				},
			})
		},
	}
}

// --- Tool: decision.sla_status ---

func toolSLAStatus() decisionToolDef {
	return decisionToolDef{
		Def: llm.ToolDef{
			Name:        "decision.sla_status",
			Description: "查询工单的 SLA 状态和紧急程度评估",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		Handler: func(ctx *decisionToolContext, _ json.RawMessage) (json.RawMessage, error) {
			var ticket struct {
				SLAStatus             string
				SLAResponseDeadline   *time.Time
				SLAResolutionDeadline *time.Time
			}
			if err := ctx.tx.Table("itsm_tickets").Where("id = ?", ctx.ticketID).
				Select("sla_status, sla_response_deadline, sla_resolution_deadline").
				First(&ticket).Error; err != nil {
				return toolError("工单不存在")
			}

			if ticket.SLAResponseDeadline == nil && ticket.SLAResolutionDeadline == nil {
				return json.Marshal(map[string]any{
					"has_sla": false,
					"urgency": "normal",
				})
			}

			now := time.Now()
			result := map[string]any{
				"has_sla":    true,
				"sla_status": ticket.SLAStatus,
			}

			urgency := "normal"
			if ticket.SLAResponseDeadline != nil {
				remaining := int64(ticket.SLAResponseDeadline.Sub(now).Seconds())
				result["response_remaining_seconds"] = remaining
				if remaining < 0 {
					urgency = "breached"
				} else if remaining < 1800 { // 30 minutes
					urgency = "critical"
				} else if remaining < 3600 { // 1 hour
					urgency = "warning"
				}
			}
			if ticket.SLAResolutionDeadline != nil {
				remaining := int64(ticket.SLAResolutionDeadline.Sub(now).Seconds())
				result["resolution_remaining_seconds"] = remaining
				if remaining < 0 && urgency != "breached" {
					urgency = "breached"
				}
			}
			result["urgency"] = urgency

			return json.Marshal(result)
		},
	}
}

// --- Tool: decision.list_actions ---

func toolListActions() decisionToolDef {
	return decisionToolDef{
		Def: llm.ToolDef{
			Name:        "decision.list_actions",
			Description: "列出当前服务可用的自动化动作（ServiceAction）",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		Handler: func(ctx *decisionToolContext, _ json.RawMessage) (json.RawMessage, error) {
			type actionRow struct {
				ID          uint
				Code        string
				Name        string
				Description string
			}
			var actions []actionRow
			ctx.tx.Table("itsm_service_actions").
				Where("service_id = ? AND is_active = ? AND deleted_at IS NULL", ctx.serviceID, true).
				Select("id, code, name, description").
				Order("id ASC").
				Find(&actions)

			items := make([]map[string]any, len(actions))
			for i, a := range actions {
				items[i] = map[string]any{
					"id":          a.ID,
					"code":        a.Code,
					"name":        a.Name,
					"description": a.Description,
				}
			}
			return json.Marshal(map[string]any{
				"actions": items,
				"count":   len(items),
			})
		},
	}
}

// --- Helpers ---

func toolError(msg string) (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"error":   true,
		"message": msg,
	})
}
