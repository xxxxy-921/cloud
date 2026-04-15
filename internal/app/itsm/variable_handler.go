package itsm

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"metis/internal/handler"
)

// VariableHandler exposes process variable APIs.
type VariableHandler struct {
	svc *VariableService
}

func NewVariableHandler(i do.Injector) (*VariableHandler, error) {
	svc := do.MustInvoke[*VariableService](i)
	return &VariableHandler{svc: svc}, nil
}

// List returns all process variables for a ticket.
// GET /api/v1/itsm/tickets/:id/variables
func (h *VariableHandler) List(c *gin.Context) {
	ticketID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		handler.Fail(c, http.StatusBadRequest, "invalid ticket id")
		return
	}

	vars, err := h.svc.ListByTicket(uint(ticketID))
	if err != nil {
		handler.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]ProcessVariableResponse, len(vars))
	for i, v := range vars {
		result[i] = v.ToResponse()
	}
	handler.OK(c, result)
}
