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

func TestLoadPublicKey_PEMEnv(t *testing.T) {
	priv, _ := NewTestKeyPair()
	pemStr := marshalPublicKeyPEM(t, &priv.PublicKey)

	os.Unsetenv("JWT_PUBLIC_KEY_URL")
	os.Setenv("JWT_PUBLIC_KEY", pemStr)
	defer os.Unsetenv("JWT_PUBLIC_KEY")

	key, err := LoadPublicKey()
	if err != nil {
		t.Fatalf("LoadPublicKey: %v", err)
	}
	if key.N.Cmp(priv.PublicKey.N) != 0 {
		t.Fatal("chave PEM carregada diferente")
	}
}

func TestLoadPublicKey_URLEnv(t *testing.T) {
	_, pub := NewTestKeyPair()
	body, _ := json.Marshal(jwksResponse{Keys: jwk2(RSAPublicKeyToJWK(pub, "k1"))})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	os.Unsetenv("JWT_PUBLIC_KEY")
	os.Setenv("JWT_PUBLIC_KEY_URL", srv.URL)
	defer os.Unsetenv("JWT_PUBLIC_KEY_URL")

	key, err := LoadPublicKey()
	if err != nil {
		t.Fatalf("LoadPublicKey: %v", err)
	}
	if key.N.Cmp(pub.N) != 0 {
		t.Fatal("chave JWKS carregada diferente")
	}
}

func TestLoadPublicKey_VazioRetornaErro(t *testing.T) {
	os.Unsetenv("JWT_PUBLIC_KEY")
	os.Unsetenv("JWT_PUBLIC_KEY_URL")
	if _, err := LoadPublicKey(); err == nil {
		t.Fatal("esperava erro quando nenhuma var definida")
	}
}
