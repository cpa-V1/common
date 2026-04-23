package common

// Quotas — limites máximos por recurso por prefeitura. Valores são hardcoded
// (sem override per-tenant, sem persistência em DB) — source of truth único.
// Valor -1 significa ilimitado.
//
// Enforcement: cada svc importa DefaultQuotas e valida `COUNT(*)+1 > quota`
// antes de INSERT. Exceder → HTTP 422 + errorCode CPA_ERROR_QUOTA_EXCEEDED.
//
// Consumo UI: svc-prefeitura expõe `GET /cpa/v1/prefeituras/me/quotas`
// retornando este struct serializado. UI usa pra badges "X/Y" em cada página.
// Usage atual é derivado localmente (items.length dos GETs de listagem).
type Quotas struct {
	Tratores      int `json:"tratores"`
	Ferramentas   int `json:"ferramentas"`
	Motoristas    int `json:"motoristas"`
	Funcionarios  int `json:"funcionarios"`
	Agricultores  int `json:"agricultores"`
	Terras        int `json:"terras"`
	Users         int `json:"users"`
	PedidosAtivos int `json:"pedidosAtivos"` // status NOT IN (concluido, cancelado)
}

// DefaultQuotas — valores default aplicados a toda prefeitura. Ajustar
// aqui = mudança global (todos tenants). Rebuild de todos svcs que consomem.
var DefaultQuotas = Quotas{
	Tratores:      20,
	Ferramentas:   30,
	Motoristas:    30,
	Funcionarios:  50,
	Agricultores:  2000,
	Terras:        6000,
	Users:         100,
	PedidosAtivos: 500,
}

// ErrCodeQuotaExceeded — errorCode padronizado retornado em 422 quando
// POST excederia quota. UI reconhece e exibe mensagem amigável.
const ErrCodeQuotaExceeded = "CPA_ERROR_QUOTA_EXCEEDED"
