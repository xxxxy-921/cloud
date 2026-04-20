package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// R is the unified API response structure.
type R struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, R{Code: 0, Message: "ok", Data: data})
}

func Fail(c *gin.Context, httpStatus int, msg string) {
	c.JSON(httpStatus, R{Code: -1, Message: msg})
}

// ParseUintParam extracts a uint path parameter from the gin context.
// If the parameter is missing or not a valid positive integer, it writes a
// 400 response and returns (0, false). Callers should return early on false.
func ParseUintParam(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || v == 0 {
		Fail(c, http.StatusBadRequest, fmt.Sprintf("invalid parameter %q: %s", name, raw))
		return 0, false
	}
	return uint(v), true
}
