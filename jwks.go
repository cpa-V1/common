package common

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"
)

// jwksResponse é o formato JSON de JWKS (RFC 7517).
type jwksResponse struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	N   string `json:"n"` // base64url modulus
	E   string `json:"e"` // base64url exponent
}

// FetchJWKS faz GET na URL e parseia a primeira chave RSA RS256.
// Usado pelo PublicKeyCache (lazy, ver pubkey_cache.go). Single retry com
// backoff 1s entre tentativas — cobre flakiness passageira (svc-login
// reiniciando, rede instável). Total: 2 tentativas.
func FetchJWKS(url string) (*rsa.PublicKey, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Second)
		}
		key, err := fetchJWKSOnce(url)
		if err == nil {
			return key, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("fetch JWKS após 2 tentativas: %w", lastErr)
}

func fetchJWKSOnce(url string) (*rsa.PublicKey, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("parse JWKS: %w", err)
	}

	for _, k := range jwks.Keys {
		if k.Kty == "RSA" && (k.Alg == "RS256" || k.Alg == "") {
			return jwkToRSAPublicKey(k)
		}
	}
	return nil, fmt.Errorf("nenhuma chave RSA RS256 encontrada em %s", url)
}

func jwkToRSAPublicKey(k jwk) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode N: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode E: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes).Int64()
	return &rsa.PublicKey{N: n, E: int(e)}, nil
}

// RSAPublicKeyToJWK converte uma chave pública RSA em JWK (usado por svc-token).
func RSAPublicKeyToJWK(pub *rsa.PublicKey, kid string) map[string]any {
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	eBytes := big.NewInt(int64(pub.E)).Bytes()
	e := base64.RawURLEncoding.EncodeToString(eBytes)
	return map[string]any{
		"kty": "RSA",
		"alg": "RS256",
		"use": "sig",
		"kid": kid,
		"n":   n,
		"e":   e,
	}
}

// LoadPublicKey removido — dead code. Startup eager foi substituído por
// PublicKeyCache lazy (ver pubkey_cache.go, ADR-005). Retry agora mora
// direto em FetchJWKS.
