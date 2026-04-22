package common

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// requestBodyLogCap é o tamanho máximo (em bytes) do body incluído no log
// "incoming request". Body maior é truncado + flag truncated=true. Evita
// encher logs com uploads grandes.
const requestBodyLogCap = 8 * 1024

// loginPathsSkipped: paths cujo body contém credenciais e não deve ser logado
// (evita vazamento de senha em logs dev/prod).
var loginPathsSkipped = map[string]struct{}{
	"/cpa/v1/login": {},
}

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

// SetupLogging configura zerolog globalmente: timestamp "2006-01-02 15:04:05.000",
// campo "caller" com path curto (2 componentes finais), e campo "service"
// (lido do env var SERVICE_NAME, omitido se vazio). Deve ser chamado no início
// de cada serviço.
func SetupLogging() {
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05.000"
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
	ctx := log.With().Caller()
	if svc := os.Getenv("SERVICE_NAME"); svc != "" {
		ctx = ctx.Str("service", svc)
	}
	log.Logger = ctx.Logger()
}

// DebugIDMiddleware cria debugID por request, injeta logger no contexto,
// loga "incoming request" (com requestBody capado a 8KB, truncado se maior;
// redacted em paths de login) na entrada, e loga linha formato GIN com
// status+latency na saída.
func DebugIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Se veio de serviço upstream com X-Request-Id, herda (distributed tracing).
		// Senão gera UUID novo.
		debugID := c.GetHeader(HeaderRequestID)
		if debugID == "" {
			debugID = uuid.New().String()
		}
		logger := log.With().
			Str("debugID", debugID).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Logger()
		c.Set(ctxDebugID, debugID)
		c.Set(ctxLogger, &logger)

		// Lê body (cap 8KB), restaura pra handler. Evita logar senhas em paths
		// de login. Body maior que cap vira truncated=true.
		bodyStr, truncated := readRequestBodyForLog(c)
		entry := logger.Info().Str("requestBody", bodyStr)
		if truncated {
			entry = entry.Bool("requestBodyTruncated", true)
		}
		entry.Msg("incoming request")

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

// readRequestBodyForLog lê até requestBodyLogCap bytes do body e restaura
// c.Request.Body com o conteúdo completo lido (handlers funcionam normal).
// Retorna a string pra log + flag de truncamento. Em paths de login, não lê
// o body e retorna marker "[redacted:login]".
func readRequestBodyForLog(c *gin.Context) (string, bool) {
	if _, skip := loginPathsSkipped[c.Request.URL.Path]; skip {
		return "[redacted:login]", false
	}
	if c.Request.Body == nil {
		return "", false
	}
	// Lê cap+1 pra detectar truncamento.
	limited := io.LimitReader(c.Request.Body, requestBodyLogCap+1)
	b, err := io.ReadAll(limited)
	if err != nil {
		return "", false
	}
	// Restaura body: read cap+1 bytes já consumidos; concat com resto (se houver).
	rest, _ := io.ReadAll(c.Request.Body)
	full := append(b, rest...)
	c.Request.Body = io.NopCloser(bytes.NewReader(full))

	truncated := len(b) > requestBodyLogCap
	if truncated {
		b = b[:requestBodyLogCap]
	}
	return string(b), truncated
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
