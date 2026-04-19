package common

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ginZerologWriter struct{}

func (ginZerologWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\n")
	if msg != "" {
		log.Info().Msg(msg)
	}
	return len(p), nil
}

// SetupGinLogger redirects gin's native logger output to zerolog so that
// GIN's line-based log messages appear inside the "message" field of our
// structured JSON logs.
func SetupGinLogger() {
	gin.DefaultWriter = ginZerologWriter{}
	gin.DefaultErrorWriter = ginZerologWriter{}
}

// GinLogFormatter é um formatter customizado para gin.LoggerWithFormatter
// que inclui o debugID na linha de log, lendo do contexto do GIN.
func GinLogFormatter(p gin.LogFormatterParams) string {
	debugID, _ := p.Keys[ctxDebugID].(string)
	return fmt.Sprintf("[GIN] debugID=%s | %3d | %v | %s | %s %q\n",
		debugID, p.StatusCode, p.Latency, p.ClientIP, p.Method, p.Path)
}

const (
	ctxDebugID = "debugID"
	ctxLogger  = "logger"
)

func DebugIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		debugID := uuid.New().String()
		logger := log.With().
			Str("debugID", debugID).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Logger()
		c.Set(ctxDebugID, debugID)
		c.Set(ctxLogger, &logger)

		c.Next()
	}
}

func LoggerFromCtx(c *gin.Context) *zerolog.Logger {
	if v, ok := c.Get(ctxLogger); ok {
		return v.(*zerolog.Logger)
	}
	l := log.Logger
	return &l
}

func DebugIDFromCtx(c *gin.Context) string {
	return c.GetString(ctxDebugID)
}
