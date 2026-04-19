package common

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	ctxDebugID = "debugID"
	ctxLogger  = "logger"
)

// ginZerologWriter redireciona stdout/stderr do GIN para zerolog (usado por
// logs internos do GIN — como registro de rotas no startup e panics).
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

// DebugIDMiddleware cria debugID por request, injeta logger no contexto,
// e loga uma linha no fim do request no formato nativo do GIN com debugID
// como campo JSON estruturado separado.
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

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		msg := fmt.Sprintf("[GIN] %s | %3d | %13v | %15s | %-7s %q",
			time.Now().Format("2006/01/02 - 15:04:05"),
			c.Writer.Status(),
			latency,
			c.ClientIP(),
			c.Request.Method,
			c.Request.URL.Path,
		)
		log.Info().Str("debugID", debugID).Msg(msg)
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
