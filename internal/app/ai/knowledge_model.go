package ai

import (
	"encoding/json"
	"time"

	"metis/internal/model"
)

// Knowledge base compile statuses
const (
	CompileStatusIdle      = "idle"
	CompileStatusCompiling = "compiling"
	CompileStatusCompleted = "completed"
	CompileStatusError     = "error"
)

// Source formats
const (
	SourceFormatMarkdown = "markdown"
	SourceFormatText     = "text"
	SourceFormatPDF      = "pdf"
	SourceFormatDocx     = "docx"
	SourceFormatXlsx     = "xlsx"
	SourceFormatPptx     = "pptx"
	SourceFormatURL      = "url"
)

// Source extract statuses
const (
	ExtractStatusPending   = "pending"
	ExtractStatusCompleted = "completed"
	ExtractStatusError     = "error"
)

// Knowledge node types
const (
	NodeTypeIndex   = "index"
	NodeTypeConcept = "concept"
)

// Knowledge edge relation types
const (
	EdgeRelationRelated     = "related"
	EdgeRelationContradicts = "contradicts"
	EdgeRelationExtends     = "extends"
	EdgeRelationPartOf      = "part_of"
)

// Knowledge log actions
const (
	KnowledgeLogCompile   = "compile"
	KnowledgeLogRecompile = "recompile"
	KnowledgeLogCrawl     = "crawl"
	KnowledgeLogLint      = "lint"
)

// Compile methods
const (
	CompileMethodKnowledgeGraph = "knowledge_graph"
)

// --- KnowledgeBase ---

type KnowledgeBase struct {
	model.BaseModel
	Name           string     `json:"name" gorm:"size:128;not null"`
	Description    string     `json:"description" gorm:"type:text"`
	CompileStatus  string     `json:"compileStatus" gorm:"size:16;not null;default:idle"`
	CompileMethod  string     `json:"compileMethod" gorm:"size:64;not null;default:knowledge_graph"`
	CompileModelID *uint      `json:"compileModelId" gorm:"index"`
	CompiledAt     *time.Time `json:"compiledAt"`
	AutoCompile    bool       `json:"autoCompile" gorm:"not null;default:false"`
	SourceCount    int        `json:"sourceCount" gorm:"not null;default:0"`
	NodeCount      int        `json:"nodeCount" gorm:"not null;default:0"`
}

func (KnowledgeBase) TableName() string { return "ai_knowledge_bases" }

type KnowledgeBaseResponse struct {
	ID             uint       `json:"id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	CompileStatus  string     `json:"compileStatus"`
	CompileMethod  string     `json:"compileMethod"`
	CompileModelID *uint      `json:"compileModelId"`
	CompiledAt     *time.Time `json:"compiledAt"`
	AutoCompile    bool       `json:"autoCompile"`
	SourceCount    int        `json:"sourceCount"`
	NodeCount      int        `json:"nodeCount"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

func (kb *KnowledgeBase) ToResponse() KnowledgeBaseResponse {
	return KnowledgeBaseResponse{
		ID:             kb.ID,
		Name:           kb.Name,
		Description:    kb.Description,
		CompileStatus:  kb.CompileStatus,
		CompileMethod:  kb.CompileMethod,
		CompileModelID: kb.CompileModelID,
		CompiledAt:     kb.CompiledAt,
		AutoCompile:    kb.AutoCompile,
		SourceCount:    kb.SourceCount,
		NodeCount:      kb.NodeCount,
		CreatedAt:      kb.CreatedAt,
		UpdatedAt:      kb.UpdatedAt,
	}
}

// --- KnowledgeSource ---

type KnowledgeSource struct {
	model.BaseModel
	KbID          uint    `json:"kbId" gorm:"not null;index"`
	ParentID      *uint   `json:"parentId" gorm:"index"`
	Title         string  `json:"title" gorm:"size:256;not null"`
	Content       string  `json:"content" gorm:"type:text"`
	Format        string  `json:"format" gorm:"size:16;not null"`
	SourceURL     string  `json:"sourceUrl" gorm:"size:1024"`
	CrawlDepth    int     `json:"crawlDepth" gorm:"not null;default:0"`
	URLPattern    string     `json:"urlPattern" gorm:"size:512"`
	CrawlEnabled  bool       `json:"crawlEnabled" gorm:"not null;default:false"`
	CrawlSchedule string     `json:"crawlSchedule" gorm:"size:64"`
	LastCrawledAt *time.Time `json:"lastCrawledAt"`
	FileName      string     `json:"fileName" gorm:"size:256"`
	ByteSize      int64   `json:"byteSize"`
	ExtractStatus string  `json:"extractStatus" gorm:"size:16;not null;default:pending"`
	ContentHash   string  `json:"contentHash" gorm:"size:64"`
	ErrorMessage  string  `json:"errorMessage" gorm:"type:text"`
}

func (KnowledgeSource) TableName() string { return "ai_knowledge_sources" }

type KnowledgeSourceResponse struct {
	ID            uint       `json:"id"`
	KbID          uint       `json:"kbId"`
	ParentID      *uint      `json:"parentId"`
	Title         string     `json:"title"`
	Format        string     `json:"format"`
	SourceURL     string     `json:"sourceUrl,omitempty"`
	CrawlDepth    int        `json:"crawlDepth"`
	CrawlEnabled  bool       `json:"crawlEnabled"`
	CrawlSchedule string     `json:"crawlSchedule,omitempty"`
	LastCrawledAt *time.Time `json:"lastCrawledAt,omitempty"`
	FileName      string     `json:"fileName,omitempty"`
	ByteSize      int64      `json:"byteSize"`
	ExtractStatus string     `json:"extractStatus"`
	ErrorMessage  string     `json:"errorMessage,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func (s *KnowledgeSource) ToResponse() KnowledgeSourceResponse {
	return KnowledgeSourceResponse{
		ID:            s.ID,
		KbID:          s.KbID,
		ParentID:      s.ParentID,
		Title:         s.Title,
		Format:        s.Format,
		SourceURL:     s.SourceURL,
		CrawlDepth:    s.CrawlDepth,
		CrawlEnabled:  s.CrawlEnabled,
		CrawlSchedule: s.CrawlSchedule,
		LastCrawledAt: s.LastCrawledAt,
		FileName:      s.FileName,
		ByteSize:      s.ByteSize,
		ExtractStatus: s.ExtractStatus,
		ErrorMessage:  s.ErrorMessage,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

// --- KnowledgeNode ---

type KnowledgeNode struct {
	model.BaseModel
	KbID       uint            `json:"kbId" gorm:"not null;index"`
	Title      string          `json:"title" gorm:"size:256;not null"`
	Summary    string          `json:"summary" gorm:"type:text"`
	Content    *string         `json:"content" gorm:"type:text"`
	NodeType   string          `json:"nodeType" gorm:"size:16;not null;default:concept"`
	SourceIDs  json.RawMessage `json:"sourceIds" gorm:"type:text"`
	CompiledAt *time.Time      `json:"compiledAt"`
}

func (KnowledgeNode) TableName() string { return "ai_knowledge_nodes" }

type KnowledgeNodeResponse struct {
	ID         uint            `json:"id"`
	KbID       uint            `json:"kbId"`
	Title      string          `json:"title"`
	Summary    string          `json:"summary"`
	Content    *string         `json:"content,omitempty"`
	HasContent bool            `json:"hasContent"`
	NodeType   string          `json:"nodeType"`
	SourceIDs  json.RawMessage `json:"sourceIds"`
	EdgeCount  int             `json:"edgeCount,omitempty"`
	CompiledAt *time.Time      `json:"compiledAt"`
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

func (n *KnowledgeNode) ToResponse() KnowledgeNodeResponse {
	sourceIDs := n.SourceIDs
	if len(sourceIDs) == 0 {
		sourceIDs = json.RawMessage("[]")
	}
	return KnowledgeNodeResponse{
		ID:         n.ID,
		KbID:       n.KbID,
		Title:      n.Title,
		Summary:    n.Summary,
		Content:    n.Content,
		HasContent: n.Content != nil && *n.Content != "",
		NodeType:   n.NodeType,
		SourceIDs:  sourceIDs,
		CompiledAt: n.CompiledAt,
		CreatedAt:  n.CreatedAt,
		UpdatedAt:  n.UpdatedAt,
	}
}

// --- KnowledgeEdge ---

type KnowledgeEdge struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	KbID        uint   `json:"kbId" gorm:"not null;index"`
	FromNodeID  uint   `json:"fromNodeId" gorm:"not null;index"`
	ToNodeID    uint   `json:"toNodeId" gorm:"not null;index"`
	Relation    string `json:"relation" gorm:"size:32;not null"`
	Description string `json:"description" gorm:"size:512"`
}

func (KnowledgeEdge) TableName() string { return "ai_knowledge_edges" }

type KnowledgeEdgeResponse struct {
	ID          uint   `json:"id"`
	FromNodeID  uint   `json:"fromNodeId"`
	ToNodeID    uint   `json:"toNodeId"`
	Relation    string `json:"relation"`
	Description string `json:"description,omitempty"`
}

func (e *KnowledgeEdge) ToResponse() KnowledgeEdgeResponse {
	return KnowledgeEdgeResponse{
		ID:          e.ID,
		FromNodeID:  e.FromNodeID,
		ToNodeID:    e.ToNodeID,
		Relation:    e.Relation,
		Description: e.Description,
	}
}

// --- KnowledgeLog ---

type KnowledgeLog struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	KbID         uint      `json:"kbId" gorm:"not null;index"`
	Action       string    `json:"action" gorm:"size:32;not null"`
	ModelID      string    `json:"modelId" gorm:"size:128"`
	NodesCreated int       `json:"nodesCreated"`
	NodesUpdated int       `json:"nodesUpdated"`
	EdgesCreated int       `json:"edgesCreated"`
	LintIssues   int       `json:"lintIssues"`
	Details      string    `json:"details" gorm:"type:text"`
	ErrorMessage string    `json:"errorMessage" gorm:"type:text"`
	CreatedAt    time.Time `json:"createdAt" gorm:"index"`
}

func (KnowledgeLog) TableName() string { return "ai_knowledge_logs" }
