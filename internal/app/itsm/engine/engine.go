package engine

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
)

// WorkflowEngine defines the contract for workflow execution engines.
// ClassicEngine (BPMN graph traversal) implements this in Phase 2.
// SmartEngine (Agent-driven) will implement this in Phase 3.
type WorkflowEngine interface {
	// Start initialises the workflow for a ticket. It parses the workflow definition,
	// finds the start node, and creates the first Activity on the target node.
	Start(ctx context.Context, tx *gorm.DB, params StartParams) error

	// Progress advances the workflow. It completes the current Activity with the
	// given outcome and creates the next Activity based on outgoing edges.
	Progress(ctx context.Context, tx *gorm.DB, params ProgressParams) error

	// Cancel terminates all active Activities and marks the ticket as cancelled.
	Cancel(ctx context.Context, tx *gorm.DB, params CancelParams) error
}

type StartParams struct {
	TicketID        uint
	WorkflowJSON    json.RawMessage
	RequesterID     uint
	StartFormSchema string // form schema JSON for variable binding (optional)
	StartFormData   string // form data JSON for variable binding (optional)
}

type ProgressParams struct {
	TicketID   uint
	ActivityID uint
	Outcome    string
	Result     json.RawMessage // form data or processing result
	OperatorID uint
}

type CancelParams struct {
	TicketID   uint
	Reason     string
	OperatorID uint
}

// Errors
var (
	ErrNoStartNode       = errors.New("workflow: no start node found")
	ErrMultipleStartNodes = errors.New("workflow: multiple start nodes found")
	ErrNoEndNode         = errors.New("workflow: no end node found")
	ErrNoOutgoingEdge    = errors.New("workflow: no matching outgoing edge for outcome")
	ErrMaxDepthExceeded  = errors.New("workflow: automatic step depth exceeded maximum (50)")
	ErrInvalidNodeType   = errors.New("workflow: invalid node type")
	ErrActivityNotFound  = errors.New("workflow: activity not found")
	ErrActivityNotActive = errors.New("workflow: activity is not in an active state")
	ErrNodeNotFound      = errors.New("workflow: referenced node not found in workflow")
)

// Node types
const (
	NodeStart   = "start"
	NodeEnd     = "end"
	NodeForm    = "form"
	NodeApprove = "approve"
	NodeProcess = "process"
	NodeAction  = "action"
	NodeGateway = "gateway"
	NodeNotify  = "notify"
	NodeWait    = "wait"
)

var ValidNodeTypes = map[string]bool{
	NodeStart: true, NodeEnd: true, NodeForm: true,
	NodeApprove: true, NodeProcess: true, NodeAction: true,
	NodeGateway: true, NodeNotify: true, NodeWait: true,
}

// IsAutoNode returns true for node types that execute automatically without human intervention.
func IsAutoNode(nodeType string) bool {
	return nodeType == NodeGateway || nodeType == NodeAction || nodeType == NodeNotify
}

// IsHumanNode returns true for node types that require human interaction.
func IsHumanNode(nodeType string) bool {
	return nodeType == NodeForm || nodeType == NodeApprove || nodeType == NodeProcess || nodeType == NodeWait
}

// MaxAutoDepth limits recursive automatic node processing to prevent infinite loops.
const MaxAutoDepth = 50

// Activity status constants
const (
	ActivityPending         = "pending"
	ActivityPendingApproval = "pending_approval" // smart engine: low-confidence AI decision awaiting human confirmation
	ActivityInProgress      = "in_progress"
	ActivityCompleted       = "completed"
	ActivityCancelled       = "cancelled"
	ActivityRejected        = "rejected" // smart engine: AI decision rejected by human
)

// Smart engine errors
var (
	ErrSmartEngineUnavailable = errors.New("智能引擎不可用：AI 模块未安装")
	ErrAIDecisionFailed       = errors.New("AI 决策失败")
	ErrAIDecisionTimeout      = errors.New("AI 决策超时")
	ErrAIDisabled             = errors.New("AI 决策已停用（连续失败次数过多）")
	ErrInvalidDecisionPlan    = errors.New("AI 决策计划校验不通过")
)

// Smart engine defaults
const (
	DefaultConfidenceThreshold    = 0.8
	DefaultDecisionTimeoutSeconds = 30
	MaxAIFailureCount             = 3
)
