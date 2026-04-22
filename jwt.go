package common

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTIssuer é o único issuer aceito. Emitido por svc-login; verificado por
// ParseJWT. Fail-closed: tokens com iss diferente → erro de parse.
const JWTIssuer = "svc-login"

// CpaClaims são os claims JWT usados pelo sistema CPA.
// Emitidos por svc-login; verificados por TenantMiddleware.
//
// `CpaEmail` é informativo — usado só pra debug/observabilidade (ex: mostrar
// no UI, logar). Authz sempre usa `Subject` (UUID do user).
type CpaClaims struct {
	CpaPrefeituraID string `json:"cpa_prefeitura_id"`
	CpaEmail        string `json:"cpa_email,omitempty"`
	jwt.RegisteredClaims
}

// ParseJWT valida e parseia um JWT RS256, retornando os claims.
// Valida: assinatura RSA + algoritmo + exp + iss == "svc-login".
func ParseJWT(tokenStr string, pubKey *rsa.PublicKey) (*CpaClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CpaClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("algoritmo inesperado: %v", t.Header["alg"])
		}
		return pubKey, nil
	}, jwt.WithIssuer(JWTIssuer))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*CpaClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("token inválido")
	}
	return claims, nil
}

// ParseRSAPublicKeyPEM parseia uma chave pública RSA em formato PEM (PKIX).
func ParseRSAPublicKeyPEM(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("falha ao decodificar PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("falha ao parsear chave pública: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("chave pública não é RSA")
	}
	return rsaPub, nil
}

// MintJWT gera um JWT RS256 assinado. Usado por svc-token (produção) e ferramentas dev/teste.
func MintJWT(privKey *rsa.PrivateKey, prefeituraUUID, sub string) (string, error) {
	return MintJWTWithEmail(privKey, prefeituraUUID, sub, "")
}

// MintJWTWithEmail é como MintJWT mas inclui claim informativo `cpa_email`.
// `email` vazio omite o claim.
func MintJWTWithEmail(privKey *rsa.PrivateKey, prefeituraUUID, sub, email string) (string, error) {
	claims := CpaClaims{
		CpaPrefeituraID: prefeituraUUID,
		CpaEmail:        email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   sub,
			Issuer:    JWTIssuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(privKey)
}

// NewTestKeyPair gera par de chaves RSA 2048-bit para uso em testes.
func NewTestKeyPair() (*rsa.PrivateKey, *rsa.PublicKey) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return privKey, &privKey.PublicKey
}
