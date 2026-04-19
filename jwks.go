package common

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
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
// Para uso em startup; não implementa cache nem refresh (deferred).
func FetchJWKS(url string) (*rsa.PublicKey, error) {
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

// LoadPublicKey carrega a chave pública RSA do ambiente.
// Prioridade:
//   1. JWT_PUBLIC_KEY_URL — fetch JWKS (retry 3x: 1s, 2s, 4s)
//   2. JWT_PUBLIC_KEY — PEM direto
// Retorna erro se nenhuma var definida.
func LoadPublicKey() (*rsa.PublicKey, error) {
	if url := os.Getenv("JWT_PUBLIC_KEY_URL"); url != "" {
		delays := []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second}
		var lastErr error
		for i, d := range delays {
			if d > 0 {
				time.Sleep(d)
			}
			key, err := FetchJWKS(url)
			if err == nil {
				return key, nil
			}
			lastErr = err
			if i < len(delays)-1 {
				// log retry na próxima iteração
				continue
			}
		}
		return nil, fmt.Errorf("falha ao buscar JWKS em %s após retries: %w", url, lastErr)
	}
	if pem := os.Getenv("JWT_PUBLIC_KEY"); pem != "" {
		return ParseRSAPublicKeyPEM(pem)
	}
	return nil, fmt.Errorf("defina JWT_PUBLIC_KEY_URL ou JWT_PUBLIC_KEY")
}
