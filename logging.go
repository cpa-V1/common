package common

import (
	"fmt"
	"strconv"
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

// SetupLogging configura zerolog globalmente: timestamp unix, campo "caller"
// com path curto (2 componentes finais). Deve ser chamado no início de cada serviço.
func SetupLogging() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		short := file
		if idx := strings.LastIndex(file, "/"); idx > 0 {
			if idx2 := strings.LastIndex(file[:idx], "/"); idx2 >= 0 {
				short = file[idx2+1:]
			} else {
				short = file[idx+1:]
			}
		}
		return short + ":" + strconv.Itoa(line)
	}
	log.Logger = log.With().Caller().Logger()
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
		// Usa logger do contexto (já enriquecido por TenantMiddleware com prefeitura_uuid + sub)
		LoggerFromCtx(c).Info().Msg(msg)
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
