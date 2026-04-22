package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Identidades e escopo do **control plane** (cross-tenant ops).
//
// Root user = super-admin do sistema CPA. Não vive em nenhum tenant DB —
// autenticado por svc-login contra env CPA_ROOT_PASSWORD_HASH (bcrypt).
// Token emitido carrega cpa_prefeitura_id=PlatformTenantUUID como escopo
// especial — PlatformMiddleware valida.
//
// Control plane endpoints (POST /prefeituras, listar all, billing, etc)
// exigem PlatformMiddleware + EnforceMiddleware("cpa.platform.<op>").
const (
	RootUserUUID       = "00000000-0000-0000-0000-000000000000"
	RootUserEmail      = "root@cpa.local" // `.local` reservado (RFC 6762) — clarifica que não é email externo
	PlatformTenantUUID = "00000000-0000-0000-0000-000000000000"
)

// PlatformMiddleware garante que o JWT foi emitido pro scope platform
// (cpa_prefeitura_id == PlatformTenantUUID). Usado em endpoints control plane
// que operam cross-tenant ou sobre o próprio registry de tenants.
//
// Ordem de uso: AuthMiddleware → PlatformMiddleware → EnforceMiddleware.
func PlatformMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		prefUUID := PrefeituraUUIDFromCtx(c)
		if prefUUID != PlatformTenantUUID {
			RespondError(c, http.StatusForbidden, "CPA_ERROR_NOT_PLATFORM_SCOPE",
				"Endpoint exige token de escopo platform (root user).")
			c.Abort()
			return
		}
		c.Next()
	}
}
