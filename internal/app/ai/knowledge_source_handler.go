package ai

import (
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"metis/internal/handler"
)

type KnowledgeSourceHandler struct {
	svc        *KnowledgeSourceService
	repo       *KnowledgeSourceRepo
	kbSvc      *KnowledgeBaseService
	extractSvc *KnowledgeExtractService
}

func NewKnowledgeSourceHandler(i do.Injector) (*KnowledgeSourceHandler, error) {
	return &KnowledgeSourceHandler{
		svc:        do.MustInvoke[*KnowledgeSourceService](i),
		repo:       do.MustInvoke[*KnowledgeSourceRepo](i),
		kbSvc:      do.MustInvoke[*KnowledgeBaseService](i),
		extractSvc: do.MustInvoke[*KnowledgeExtractService](i),
	}, nil
}

type createSourceReq struct {
	Title         string `json:"title"`
	SourceURL     string `json:"sourceUrl"`
	CrawlDepth    int    `json:"crawlDepth"`
	URLPattern    string `json:"urlPattern"`
	CrawlEnabled  bool   `json:"crawlEnabled"`
	CrawlSchedule string `json:"crawlSchedule"`
	Content       string `json:"content"`
	Format        string `json:"format"`
}

func (h *KnowledgeSourceHandler) Create(c *gin.Context) {
	kbID, _ := strconv.Atoi(c.Param("id"))
	if _, err := h.kbSvc.Get(uint(kbID)); err != nil {
		handler.Fail(c, http.StatusNotFound, "knowledge base not found")
		return
	}

	// Check if this is a file upload
	file, fileHeader, fileErr := c.Request.FormFile("file")
	if fileErr == nil {
		defer file.Close()
		// File upload
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileHeader.Filename), "."))
		format := mapExtToFormat(ext)
		if format == "" {
			handler.Fail(c, http.StatusBadRequest, "unsupported file format: "+ext)
			return
		}

		title := strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename))

		// Read file content for text-based formats
		var content string
		if format == SourceFormatMarkdown || format == SourceFormatText {
			buf := make([]byte, fileHeader.Size)
			if _, err := file.Read(buf); err != nil {
				handler.Fail(c, http.StatusInternalServerError, "failed to read file")
				return
			}
			content = string(buf)
		}

		extractStatus := ExtractStatusCompleted
		if format != SourceFormatMarkdown && format != SourceFormatText {
			extractStatus = ExtractStatusPending
		}

		src := &KnowledgeSource{
			KbID:          uint(kbID),
			Title:         title,
			Content:       content,
			Format:        format,
			FileName:      fileHeader.Filename,
			ByteSize:      fileHeader.Size,
			ExtractStatus: extractStatus,
		}

		if err := h.svc.Create(src); err != nil {
			handler.Fail(c, http.StatusInternalServerError, err.Error())
			return
		}

		if src.ExtractStatus == ExtractStatusPending {
			h.extractSvc.EnqueueExtract(src.ID)
		}

		c.Set("audit_action", "create")
		c.Set("audit_resource", "ai_knowledge_source")
		c.Set("audit_resource_id", strconv.Itoa(int(src.ID)))
		c.Set("audit_summary", "Uploaded source: "+src.FileName)

		handler.OK(c, src.ToResponse())
		return
	}

	// JSON body (URL or text input)
	var req createSourceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	src := &KnowledgeSource{
		KbID:          uint(kbID),
		Title:         req.Title,
		CrawlDepth:    req.CrawlDepth,
		URLPattern:    req.URLPattern,
		ExtractStatus: ExtractStatusPending,
	}

	if req.SourceURL != "" {
		// URL source
		src.Format = SourceFormatURL
		src.SourceURL = req.SourceURL
		src.CrawlEnabled = req.CrawlEnabled
		src.CrawlSchedule = req.CrawlSchedule
		if src.Title == "" {
			src.Title = req.SourceURL
		}
	} else if req.Content != "" {
		// Direct text/markdown input
		src.Format = SourceFormatMarkdown
		if req.Format != "" {
			src.Format = req.Format
		}
		src.Content = req.Content
		src.ExtractStatus = ExtractStatusCompleted
		if src.Title == "" {
			src.Title = "Text input"
		}
	} else {
		handler.Fail(c, http.StatusBadRequest, "sourceUrl or content is required")
		return
	}

	if err := h.svc.Create(src); err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	if src.ExtractStatus == ExtractStatusPending {
		h.extractSvc.EnqueueExtract(src.ID)
	}

	c.Set("audit_action", "create")
	c.Set("audit_resource", "ai_knowledge_source")
	c.Set("audit_resource_id", strconv.Itoa(int(src.ID)))

	handler.OK(c, src.ToResponse())
}

func (h *KnowledgeSourceHandler) List(c *gin.Context) {
	kbID, _ := strconv.Atoi(c.Param("id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	items, total, err := h.repo.List(SourceListParams{
		KbID:     uint(kbID),
		Keyword:  c.Query("keyword"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]KnowledgeSourceResponse, len(items))
	for i, s := range items {
		resp[i] = s.ToResponse()
	}
	handler.OK(c, gin.H{"items": resp, "total": total})
}

func (h *KnowledgeSourceHandler) Get(c *gin.Context) {
	sid, _ := strconv.Atoi(c.Param("sid"))
	src, err := h.svc.Get(uint(sid))
	if err != nil {
		if errors.Is(err, ErrSourceNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	handler.OK(c, src.ToResponse())
}

func (h *KnowledgeSourceHandler) Delete(c *gin.Context) {
	sid, _ := strconv.Atoi(c.Param("sid"))
	if err := h.svc.Delete(uint(sid)); err != nil {
		if errors.Is(err, ErrSourceNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_action", "delete")
	c.Set("audit_resource", "ai_knowledge_source")
	c.Set("audit_resource_id", c.Param("sid"))

	handler.OK(c, nil)
}

func mapExtToFormat(ext string) string {
	switch ext {
	case "md", "markdown":
		return SourceFormatMarkdown
	case "txt":
		return SourceFormatText
	case "pdf":
		return SourceFormatPDF
	case "docx":
		return SourceFormatDocx
	case "xlsx":
		return SourceFormatXlsx
	case "pptx":
		return SourceFormatPptx
	default:
		return ""
	}
}
