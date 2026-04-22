package common

import (
	"crypto/rsa"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

// PublicKeyCache guarda a chave pública RSA usada pra verificar JWTs.
// Tipicamente obtida do endpoint JWKS do svc-login.
//
// É um **ponteiro pro cache**, não um snapshot one-shot. Uma instância viva
// pelo lifetime do processo (criada no main, reusada em todas requests).
//
// # Fluxo (ADR-005)
//
//  1. main() chama NewJWKSCache() — APENAS configura loader + TTL. Sem rede.
//     Boot não falha se svc-login estiver offline.
//  2. 1ª request → middleware JWT chama keyCache.Get():
//     - estado vazio → loader() roda → HTTP GET /jwks.json → chave salva
//     - seta expiry = now + TTL (60s default)
//  3. Requests dentro do TTL → retorna chave cacheada, zero rede
//  4. Request após TTL → loader() re-fetcha, renova cache
//
// Thread-safe via mutex. TTL alto (60s) → contenção baixa.
// Fail-closed: erro no fetch propaga pro middleware → 500 (nunca libera).
type PublicKeyCache struct {
	mu     sync.Mutex
	key    *rsa.PublicKey
	expiry time.Time
	ttl    time.Duration
	loader func() (*rsa.PublicKey, error)
}

// Get retorna a chave cacheada (se dentro do TTL) ou re-fetcha via loader.
func (c *PublicKeyCache) Get() (*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.key != nil && time.Now().Before(c.expiry) {
		return c.key, nil
	}
	k, err := c.loader()
	if err != nil {
		return nil, err
	}
	c.key = k
	c.expiry = time.Now().Add(c.ttl)
	return c.key, nil
}

// NewJWKSCache cria cache lazy da chave pública JWT. Lê config de env:
//
//   - CPA_JWT_PUBLIC_KEY_URL — preferencial, ex: http://svc-login:8088/.well-known/jwks.json
//   - CPA_JWT_PUBLIC_KEY     — fallback, chave RSA PEM inline (dev)
//   - CPA_JWT_PUBLIC_KEY_TTL_SECONDS — override do TTL default 60s
//
// Antes chamado NewPublicKeyCacheFromEnv. Renomeado porque "FromEnv" não
// explicava o que se cacheava (resposta: JWKS).
func NewJWKSCache() *PublicKeyCache {
	ttl := 60 * time.Second
	if s := os.Getenv("CPA_JWT_PUBLIC_KEY_TTL_SECONDS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			ttl = time.Duration(n) * time.Second
		}
	}
	return &PublicKeyCache{
		ttl: ttl,
		loader: func() (*rsa.PublicKey, error) {
			if url := os.Getenv("CPA_JWT_PUBLIC_KEY_URL"); url != "" {
				return FetchJWKS(url)
			}
			if pem := os.Getenv("CPA_JWT_PUBLIC_KEY"); pem != "" {
				return ParseRSAPublicKeyPEM(pem)
			}
			return nil, fmt.Errorf("defina CPA_JWT_PUBLIC_KEY_URL ou CPA_JWT_PUBLIC_KEY")
		},
	}
}

// NewPublicKeyCacheStatic retorna cache que sempre entrega a mesma chave.
// Uso: testes (key efêmero do par RSA in-process).
func NewPublicKeyCacheStatic(key *rsa.PublicKey) *PublicKeyCache {
	return &PublicKeyCache{
		ttl: 24 * time.Hour,
		loader: func() (*rsa.PublicKey, error) {
			return key, nil
		},
	}
}
