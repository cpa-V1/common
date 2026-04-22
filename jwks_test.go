package common

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func marshalPublicKeyPEM(t *testing.T, pub *rsa.PublicKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func TestJWK_Roundtrip(t *testing.T) {
	_, pub := NewTestKeyPair()
	jwkMap := RSAPublicKeyToJWK(pub, "test-kid")
	body, _ := json.Marshal(jwksResponse{Keys: jwk2(jwkMap)})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	fetched, err := FetchJWKS(srv.URL)
	if err != nil {
		t.Fatalf("FetchJWKS: %v", err)
	}
	if fetched.N.Cmp(pub.N) != 0 || fetched.E != pub.E {
		t.Fatal("chave pública roundtrip diferente")
	}
}

// jwk2 converte o map retornado por RSAPublicKeyToJWK para struct jwk interno.
func jwk2(m map[string]any) []jwk {
	return []jwk{{
		Kty: m["kty"].(string),
		Alg: m["alg"].(string),
		Use: m["use"].(string),
		Kid: m["kid"].(string),
		N:   m["n"].(string),
		E:   m["e"].(string),
	}}
}

func TestJWKSCache_PEMEnv(t *testing.T) {
	priv, _ := NewTestKeyPair()
	pemStr := marshalPublicKeyPEM(t, &priv.PublicKey)

	os.Unsetenv("CPA_JWT_PUBLIC_KEY_URL")
	os.Setenv("CPA_JWT_PUBLIC_KEY", pemStr)
	defer os.Unsetenv("CPA_JWT_PUBLIC_KEY")

	cache := NewJWKSCache()
	key, err := cache.Get()
	if err != nil {
		t.Fatalf("cache.Get: %v", err)
	}
	if key.N.Cmp(priv.PublicKey.N) != 0 {
		t.Fatal("chave PEM carregada diferente")
	}
}

func TestJWKSCache_URLEnv(t *testing.T) {
	_, pub := NewTestKeyPair()
	body, _ := json.Marshal(jwksResponse{Keys: jwk2(RSAPublicKeyToJWK(pub, "k1"))})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	os.Unsetenv("CPA_JWT_PUBLIC_KEY")
	os.Setenv("CPA_JWT_PUBLIC_KEY_URL", srv.URL)
	defer os.Unsetenv("CPA_JWT_PUBLIC_KEY_URL")

	cache := NewJWKSCache()
	key, err := cache.Get()
	if err != nil {
		t.Fatalf("cache.Get: %v", err)
	}
	if key.N.Cmp(pub.N) != 0 {
		t.Fatal("chave JWKS carregada diferente")
	}
}

func TestJWKSCache_VazioRetornaErro(t *testing.T) {
	os.Unsetenv("CPA_JWT_PUBLIC_KEY")
	os.Unsetenv("CPA_JWT_PUBLIC_KEY_URL")
	cache := NewJWKSCache()
	if _, err := cache.Get(); err == nil {
		t.Fatal("esperava erro quando nenhuma var definida")
	}
}

func TestFetchJWKS_RetrySucceedsOnSecondAttempt(t *testing.T) {
	_, pub := NewTestKeyPair()
	body, _ := json.Marshal(jwksResponse{Keys: jwk2(RSAPublicKeyToJWK(pub, "k1"))})
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write(body)
	}))
	defer srv.Close()

	key, err := FetchJWKS(srv.URL)
	if err != nil {
		t.Fatalf("FetchJWKS esperava sucesso no retry: %v", err)
	}
	if key.N.Cmp(pub.N) != 0 {
		t.Fatal("chave errada após retry")
	}
	if attempts != 2 {
		t.Errorf("esperava 2 tentativas, got %d", attempts)
	}
}
