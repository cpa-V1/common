package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// mockAuthzServer cria httptest.Server que retorna resposta controlada em /enforce.
func mockAuthzServer(t *testing.T, allowed bool, reason string) (*httptest.Server, *string) {
	t.Helper()
	var receivedXReqID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/authorization/v1/enforce" && r.Method == "POST" {
			receivedXReqID = r.Header.Get(HeaderRequestID)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(enforceResponse{Allowed: allowed, Reason: reason})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	return srv, &receivedXReqID
}

func setupEnforceRouter(t *testing.T, authzURL, permission string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DebugIDMiddleware())
	r.Use(func(c *gin.Context) {
		// Simula sub + prefeitura sem precisar de JWT real
		c.Set(ctxSubject, "alice")
		c.Set(ctxPrefeituraUUID, "pref-1")
		c.Next()
	})
	r.Use(EnforceMiddleware(authzURL, permission))
	r.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func TestEnforce_Allowed(t *testing.T) {
	srv, _ := mockAuthzServer(t, true, "")
	defer srv.Close()
	r := setupEnforceRouter(t, srv.URL, "cpa.tratores.read")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer fake")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("esperado 200, got %d — %s", w.Code, w.Body.String())
	}
}

func TestEnforce_Denied(t *testing.T) {
	srv, _ := mockAuthzServer(t, false, "sem permissão")
	defer srv.Close()
	r := setupEnforceRouter(t, srv.URL, "cpa.tratores.delete")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer fake")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("esperado 403, got %d — %s", w.Code, w.Body.String())
	}
}

func TestEnforce_AuthzDown(t *testing.T) {
	// URL inválida — simula svc-authz inacessível
	r := setupEnforceRouter(t, "http://localhost:1", "cpa.tratores.read")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer fake")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("fail-closed esperado 500, got %d", w.Code)
	}
}

func TestEnforce_Authz500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	r := setupEnforceRouter(t, srv.URL, "cpa.tratores.read")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer fake")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("esperado 500, got %d", w.Code)
	}
}

func TestEnforce_PropagaXRequestID(t *testing.T) {
	srv, received := mockAuthzServer(t, true, "")
	defer srv.Close()
	r := setupEnforceRouter(t, srv.URL, "cpa.tratores.read")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer fake")
	req.Header.Set(HeaderRequestID, "debug-xyz")
	r.ServeHTTP(w, req)
	if *received != "debug-xyz" {
		t.Errorf("esperava X-Request-Id 'debug-xyz' propagado, got %q", *received)
	}
}

func TestEnforceScopeFromParam_Allowed(t *testing.T) {
	srv, _ := mockAuthzServer(t, true, "")
	defer srv.Close()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DebugIDMiddleware())
	r.Use(func(c *gin.Context) {
		c.Set(ctxSubject, "alice")
		c.Next()
	})
	r.GET("/prefeituras/:uuid", EnforceScopeFromParam(srv.URL, "cpa.prefeituras.read", "uuid"),
		func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/prefeituras/pref-alvo", nil)
	req.Header.Set("Authorization", "Bearer fake")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("esperado 200, got %d", w.Code)
	}
}

func TestEnforceScopeFromParam_SemParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DebugIDMiddleware())
	r.Use(func(c *gin.Context) {
		c.Set(ctxSubject, "alice")
		c.Next()
	})
	r.GET("/x", EnforceScopeFromParam("http://localhost:1", "cpa.x.read", "uuid"),
		func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer fake")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("esperado 500 quando param vazio, got %d", w.Code)
	}
}
