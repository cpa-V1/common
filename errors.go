package common

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ErrorResponse struct {
	Message        string `json:"message"`
	DebugID        string `json:"debugID"`
	ErrorCode      string `json:"errorCode"`
	HTTPStatusCode int    `json:"httpStatusCode"`
}

func RespondError(c *gin.Context, httpStatus int, errorCode, message string) {
	debugID := DebugIDFromCtx(c)
	if debugID == "" {
		debugID = uuid.New().String()
	}
	// Auto-log 5xx pra facilitar debug (sem depender de cada call site lembrar)
	if httpStatus >= 500 {
		LoggerFromCtx(c).Error().
			Int("status", httpStatus).
			Str("errorCode", errorCode).
			Str("path", c.Request.URL.Path).
			Str("method", c.Request.Method).
			Msg(message)
	}
	c.JSON(httpStatus, ErrorResponse{
		Message:        message,
		DebugID:        debugID,
		ErrorCode:      errorCode,
		HTTPStatusCode: httpStatus,
	})
}
