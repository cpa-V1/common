package common

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ctxPool = "tenantPool"

// PoolResolver mantém 1 pool por tenant (prefeitura_uuid).
// Pool criado lazily no primeiro acesso + cacheado em memória.
type PoolResolver struct {
	mu    sync.RWMutex
	pools map[string]*pgxpool.Pool
	dsnFn func(prefeituraUUID string) string
}

func NewPoolResolver(dsnFn func(prefeituraUUID string) string) *PoolResolver {
	return &PoolResolver{
		pools: make(map[string]*pgxpool.Pool),
		dsnFn: dsnFn,
	}
}

// Tenant retorna pool do tenant; cria se não existir.
func (r *PoolResolver) Tenant(ctx context.Context, prefeituraUUID string) (*pgxpool.Pool, error) {
	r.mu.RLock()
	p := r.pools[prefeituraUUID]
	r.mu.RUnlock()
	if p != nil {
		return p, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if p := r.pools[prefeituraUUID]; p != nil {
		return p, nil
	}
	newPool, err := NewPool(ctx, r.dsnFn(prefeituraUUID))
	if err != nil {
		return nil, err
	}
	r.pools[prefeituraUUID] = newPool
	return newPool, nil
}

func (r *PoolResolver) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.pools {
		p.Close()
	}
	r.pools = make(map[string]*pgxpool.Pool)
}

// NewTenantDSN builda um dsnFn baseado em template com placeholder {TENANT}.
// prefeitura_uuid tem hyphens substituídos por underscores (Postgres identifier-friendly).
// Exemplo: "postgres://cpa:cpa@host:5432/prefeitura_{TENANT}_db?sslmode=disable"
func NewTenantDSN(template string) func(string) string {
	return func(prefeituraUUID string) string {
		normalized := strings.ReplaceAll(prefeituraUUID, "-", "_")
		return strings.ReplaceAll(template, "{TENANT}", normalized)
	}
}

// TenantDBName retorna o nome canônico do DB tenant (prefeitura_<uuid>_db, underscores).
func TenantDBName(prefeituraUUID string) string {
	return "prefeitura_" + strings.ReplaceAll(prefeituraUUID, "-", "_") + "_db"
}

// TenantPoolMiddleware injeta pool do tenant no ctx. Requer TenantMiddleware antes
// (pra prefeitura_uuid estar no ctx). Falha ao resolver pool → 500.
func TenantPoolMiddleware(r *PoolResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		prefUUID := PrefeituraUUIDFromCtx(c)
		if prefUUID == "" {
			RespondError(c, http.StatusInternalServerError, "CPA_ERROR_POOL_NO_TENANT",
				"prefeitura_uuid ausente no contexto")
			c.Abort()
			return
		}
		pool, err := r.Tenant(c.Request.Context(), prefUUID)
		if err != nil {
			LoggerFromCtx(c).Error().Err(err).Str("prefeitura_uuid", prefUUID).Msg("resolve tenant pool")
			RespondError(c, http.StatusInternalServerError, "CPA_ERROR_POOL_RESOLVE",
				"falha ao conectar no DB do tenant")
			c.Abort()
			return
		}
		c.Set(ctxPool, pool)
		c.Next()
	}
}

// PoolFromCtx retorna o pool do tenant injetado por TenantPoolMiddleware.
func PoolFromCtx(c *gin.Context) *pgxpool.Pool {
	if v, ok := c.Get(ctxPool); ok {
		if p, ok := v.(*pgxpool.Pool); ok {
			return p
		}
	}
	return nil
}
