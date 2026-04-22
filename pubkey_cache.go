package common

import (
	"crypto/rsa"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

// PublicKeyCache: read-through cache da chave pública JWT.
// - Get() retorna chave cacheada se válida (dentro do TTL); senão re-fetcha
// - Loader chamado lazily na 1ª request e a cada expiração
// - Mutex coarse-grained — TTL alto (60s default) faz fetch raro
// - Erro de fetch propaga pro middleware → 500 (zero-trust fail-closed)
type PublicKeyCache struct {
	mu     sync.Mutex
	key    *rsa.PublicKey
	expiry time.Time
	ttl    time.Duration
	loader func() (*rsa.PublicKey, error)
}

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

// NewPublicKeyCacheFromEnv lê CPA_JWT_PUBLIC_KEY_URL (preferencial) ou
// CPA_JWT_PUBLIC_KEY (fallback PEM).
// TTL default 60s; override via CPA_JWT_PUBLIC_KEY_TTL_SECONDS.
// Prefixo CPA_ indica env var definida pelo sistema (não padrão externo).
func NewPublicKeyCacheFromEnv() *PublicKeyCache {
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
