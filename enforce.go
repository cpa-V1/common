package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// enforceRequest espelha svc-authz EnforceRequest (sem importar svc-authz — evita dep circular).
type enforceRequest struct {
	Subject    string `json:"subject"`
	Permission string `json:"permission"`
	Scope      string `json:"scope"`
}

type enforceResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// EnforceMiddleware chama svc-authz /enforce pra cada request.
// Scope vem do prefeitura_uuid do token (tenant-scoped svcs: svc-trator etc).
// Fail-closed: qualquer erro na chamada vira 500. NUNCA libera em caso de falha.
// authzURL: URL base de svc-authz (ex: http://localhost:8083).
// permission: nome da permission exigida (ex: "cpa.tratores.create").
func EnforceMiddleware(authzURL, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		prefUUID := PrefeituraUUIDFromCtx(c)
		scope := "grn:cpa/prefeitura/" + prefUUID
		enforceCheck(c, authzURL, permission, scope)
	}
}

// EnforceScopeFromParam é variante pra svcs não-tenant-scoped (ex: svc-prefeitura).
// Extrai prefeitura_uuid do gin.Param(paramName) pra montar o scope.
// Use quando o :uuid da URL identifica a prefeitura alvo (não o token).
func EnforceScopeFromParam(authzURL, permission, paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		prefUUID := c.Param(paramName)
		if prefUUID == "" {
			RespondError(c, http.StatusInternalServerError, "CPA_ERROR_AUTHZ_INTERNAL",
				fmt.Sprintf("param %q vazio ao montar scope", paramName))
			c.Abort()
			return
		}
		scope := "grn:cpa/prefeitura/" + prefUUID
		enforceCheck(c, authzURL, permission, scope)
	}
}

func enforceCheck(c *gin.Context, authzURL, permission, scope string) {
	logger := LoggerFromCtx(c)
	sub := SubFromCtx(c)

	body, err := json.Marshal(enforceRequest{
		Subject:    "user:" + sub,
		Permission: permission,
		Scope:      scope,
	})
	if err != nil {
		logger.Error().Err(err).Msg("marshal enforce request")
		RespondError(c, http.StatusInternalServerError, "CPA_ERROR_AUTHZ_INTERNAL", "Erro interno ao autorizar.")
		c.Abort()
		return
	}

	req, err := http.NewRequest("POST", authzURL+"/authorization/v1/enforce", bytes.NewReader(body))
	if err != nil {
		logger.Error().Err(err).Msg("criar request enforce")
		RespondError(c, http.StatusInternalServerError, "CPA_ERROR_AUTHZ_INTERNAL", "Erro interno ao autorizar.")
		c.Abort()
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.GetHeader("Authorization"))

	resp, err := DoRequest(c, req)
	if err != nil {
		logger.Error().Err(err).Str("permission", permission).Msg("svc-authz unreachable")
		RespondError(c, http.StatusInternalServerError, "CPA_ERROR_AUTHZ_UNAVAILABLE",
			"Serviço de autorização indisponível. Tente novamente.")
		c.Abort()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error().Int("status", resp.StatusCode).Str("permission", permission).Msg("svc-authz erro")
		RespondError(c, http.StatusInternalServerError, "CPA_ERROR_AUTHZ_UNAVAILABLE",
			fmt.Sprintf("Serviço de autorização retornou %d.", resp.StatusCode))
		c.Abort()
		return
	}

	var er enforceResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		logger.Error().Err(err).Msg("decode enforce response")
		RespondError(c, http.StatusInternalServerError, "CPA_ERROR_AUTHZ_INTERNAL", "Erro interno ao autorizar.")
		c.Abort()
		return
	}

	if !er.Allowed {
		logger.Warn().Str("permission", permission).Str("scope", scope).Str("reason", er.Reason).Msg("enforce denied")
		RespondError(c, http.StatusForbidden, "CPA_ERROR_FORBIDDEN",
			"Acesso negado: "+er.Reason)
		c.Abort()
		return
	}

	c.Next()
}
