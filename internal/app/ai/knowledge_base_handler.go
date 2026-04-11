package ai

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"metis/internal/handler"
)

type KnowledgeBaseHandler struct {
	svc        *KnowledgeBaseService
	repo       *KnowledgeBaseRepo
	compileSvc *KnowledgeCompileService
}

func NewKnowledgeBaseHandler(i do.Injector) (*KnowledgeBaseHandler, error) {
	return &KnowledgeBaseHandler{
		svc:        do.MustInvoke[*KnowledgeBaseService](i),
		repo:       do.MustInvoke[*KnowledgeBaseRepo](i),
		compileSvc: do.MustInvoke[*KnowledgeCompileService](i),
	}, nil
}

type createKBReq struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	CompileMethod  string `json:"compileMethod"`
	CompileModelID *uint  `json:"compileModelId"`
	AutoCompile    bool   `json:"autoCompile"`
}

func (h *KnowledgeBaseHandler) Create(c *gin.Context) {
	var req createKBReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	compileMethod := req.CompileMethod
	if compileMethod == "" {
		compileMethod = CompileMethodKnowledgeGraph
	}

	kb := &KnowledgeBase{
		Name:           req.Name,
		Description:    req.Description,
		CompileMethod:  compileMethod,
		CompileModelID: req.CompileModelID,
		AutoCompile:    req.AutoCompile,
	}

	if err := h.svc.Create(kb); err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_action", "create")
	c.Set("audit_resource", "ai_knowledge_base")
	c.Set("audit_resource_id", strconv.Itoa(int(kb.ID)))
	c.Set("audit_summary", "Created knowledge base: "+kb.Name)

	handler.OK(c, kb.ToResponse())
}

func (h *KnowledgeBaseHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	items, total, err := h.repo.List(KBListParams{
		Keyword:  c.Query("keyword"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]KnowledgeBaseResponse, len(items))
	for i, kb := range items {
		resp[i] = kb.ToResponse()
	}
	handler.OK(c, gin.H{"items": resp, "total": total})
}

func (h *KnowledgeBaseHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	kb, err := h.svc.Get(uint(id))
	if err != nil {
		if errors.Is(err, ErrKnowledgeBaseNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	handler.OK(c, kb.ToResponse())
}

type updateKBReq struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	CompileMethod  string `json:"compileMethod"`
	CompileModelID *uint  `json:"compileModelId"`
	AutoCompile    bool   `json:"autoCompile"`
}

func (h *KnowledgeBaseHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req updateKBReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	kb, err := h.svc.Get(uint(id))
	if err != nil {
		handler.Fail(c, http.StatusNotFound, err.Error())
		return
	}

	kb.Name = req.Name
	kb.Description = req.Description
	kb.CompileMethod = req.CompileMethod
	if kb.CompileMethod == "" {
		kb.CompileMethod = CompileMethodKnowledgeGraph
	}
	kb.CompileModelID = req.CompileModelID
	kb.AutoCompile = req.AutoCompile

	if err := h.svc.Update(kb); err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_action", "update")
	c.Set("audit_resource", "ai_knowledge_base")
	c.Set("audit_resource_id", strconv.Itoa(int(kb.ID)))
	c.Set("audit_summary", "Updated knowledge base: "+kb.Name)

	handler.OK(c, kb.ToResponse())
}

func (h *KnowledgeBaseHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.svc.Delete(uint(id)); err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Set("audit_action", "delete")
	c.Set("audit_resource", "ai_knowledge_base")
	c.Set("audit_resource_id", c.Param("id"))

	handler.OK(c, nil)
}

func (h *KnowledgeBaseHandler) Compile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	kb, err := h.svc.Get(uint(id))
	if err != nil {
		if errors.Is(err, ErrKnowledgeBaseNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	if kb.CompileStatus == CompileStatusCompiling {
		handler.Fail(c, http.StatusConflict, "compilation already in progress")
		return
	}

	kb.CompileStatus = CompileStatusCompiling
	if err := h.svc.Update(kb); err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.compileSvc.EnqueueCompile(kb.ID, false)

	c.Set("audit_action", "compile")
	c.Set("audit_resource", "ai_knowledge_base")
	c.Set("audit_resource_id", strconv.Itoa(int(kb.ID)))
	c.Set("audit_summary", "Triggered compilation: "+kb.Name)

	handler.OK(c, kb.ToResponse())
}

func (h *KnowledgeBaseHandler) Recompile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	kb, err := h.svc.Get(uint(id))
	if err != nil {
		if errors.Is(err, ErrKnowledgeBaseNotFound) {
			handler.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	if kb.CompileStatus == CompileStatusCompiling {
		handler.Fail(c, http.StatusConflict, "compilation already in progress")
		return
	}

	kb.CompileStatus = CompileStatusCompiling
	if err := h.svc.Update(kb); err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.compileSvc.EnqueueCompile(kb.ID, true)

	c.Set("audit_action", "recompile")
	c.Set("audit_resource", "ai_knowledge_base")
	c.Set("audit_resource_id", strconv.Itoa(int(kb.ID)))
	c.Set("audit_summary", "Triggered recompilation: "+kb.Name)

	handler.OK(c, kb.ToResponse())
}
