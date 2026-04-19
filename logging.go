package common

import (
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
