package common

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ctxPrefeituraUUID = "prefeituraUUID"
	ctxSubject        = "subject"
)

// AuthMiddleware valida o JWT RS256 Bearer, extrai `sub` (obrigatório) e
// `cpa_prefeitura_id` (opcional — pode não existir em tokens bootstrap/admin).
// Enriquece logger com ambos quando presentes.
// Use em serviços que NÃO são tenant-scoped (ex: svc-prefeitura).
//
// keyCache: read-through cache de chave pública. Erro ao fetch → 500 (fail-closed).
func AuthMiddleware(keyCache *PublicKeyCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := LoggerFromCtx(c)
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			RespondError(c, http.StatusUnauthorized, "CPA_ERROR_UNAUTHORIZED", "Token não fornecido.")
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		if tokenStr == "" {
			RespondError(c, http.StatusUnauthorized, "CPA_ERROR_UNAUTHORIZED", "Token inválido.")
			c.Abort()
			return
		}

		pubKey, err := keyCache.Get()
		if err != nil {
			logger.Error().Err(err).Msg("fetch JWT public key")
			RespondError(c, http.StatusInternalServerError, "CPA_ERROR_TOKEN_UNAVAILABLE",
				"Serviço de token indisponível.")
			c.Abort()
			return
		}

		claims, err := ParseJWT(tokenStr, pubKey)
		if err != nil {
			RespondError(c, http.StatusUnauthorized, "CPA_ERROR_UNAUTHORIZED", "Token inválido.")
			c.Abort()
			return
		}
		sub := claims.RegisteredClaims.Subject
		c.Set(ctxSubject, sub)
		c.Set(ctxPrefeituraUUID, claims.CpaPrefeituraID)

		logCtx := LoggerFromCtx(c).With().Str("sub", sub)
		if claims.CpaPrefeituraID != "" {
			logCtx = logCtx.Str("prefeitura_uuid", claims.CpaPrefeituraID)
		}
		enriched := logCtx.Logger()
		c.Set(ctxLogger, &enriched)
		c.Next()
	}
}

// TenantMiddleware = AuthMiddleware + exige `cpa_prefeitura_id` não-vazio.
// Use em serviços tenant-scoped (ex: svc-trator).
func TenantMiddleware(keyCache *PublicKeyCache) gin.HandlerFunc {
	auth := AuthMiddleware(keyCache)
	return func(c *gin.Context) {
		auth(c)
		if c.IsAborted() {
			return
		}
		if PrefeituraUUIDFromCtx(c) == "" {
			RespondError(c, http.StatusUnauthorized, "CPA_ERROR_UNAUTHORIZED", "Token sem cpa_prefeitura_id.")
			c.Abort()
			return
		}
	}
}

func PrefeituraUUIDFromCtx(c *gin.Context) string { return c.GetString(ctxPrefeituraUUID) }
func SubFromCtx(c *gin.Context) string            { return c.GetString(ctxSubject) }
