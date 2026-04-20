package common

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
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
	// Auto-log: 5xx como Error (inesperado, humano investiga);
	// 4xx como Info (comportamento esperado, útil pra debug client).
	logger := LoggerFromCtx(c)
	var evt *zerolog.Event
	switch {
	case httpStatus >= 500:
		evt = logger.Error()
	case httpStatus >= 400:
		evt = logger.Info()
	}
	if evt != nil {
		evt.
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
