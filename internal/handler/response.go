package handler

import (
	"net/http"

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
