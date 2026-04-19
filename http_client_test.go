package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDoRequest_PropagaDebugID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var receivedRequestID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequestID = r.Header.Get(HeaderRequestID)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Simula ctx gin com debugID injetado (como DebugIDMiddleware faria)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(ctxDebugID, "test-debug-id-42")

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := DoRequest(c, req)
	if err != nil {
		t.Fatalf("DoRequest: %v", err)
	}
	defer resp.Body.Close()

	if receivedRequestID != "test-debug-id-42" {
		t.Errorf("esperava X-Request-Id='test-debug-id-42', got %q", receivedRequestID)
	}
}

func TestDoRequest_SemDebugIDNoCtx(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var receivedRequestID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequestID = r.Header.Get(HeaderRequestID)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	// NÃO seta debugID

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, _ := DoRequest(c, req)
	if resp != nil {
		resp.Body.Close()
	}
	if receivedRequestID != "" {
		t.Errorf("esperava header vazio quando ctx sem debugID, got %q", receivedRequestID)
	}
}

func TestDebugIDMiddleware_HerdaHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DebugIDMiddleware())

	var seen string
	r.GET("/x", func(c *gin.Context) {
		seen = DebugIDFromCtx(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	req.Header.Set(HeaderRequestID, "herdado-abc")
	r.ServeHTTP(w, req)

	if seen != "herdado-abc" {
		t.Errorf("esperava debugID='herdado-abc' vindo do header, got %q", seen)
	}
}

func TestDebugIDMiddleware_GeraQuandoAusente(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(DebugIDMiddleware())

	var seen string
	r.GET("/x", func(c *gin.Context) {
		seen = DebugIDFromCtx(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)

	if seen == "" {
		t.Error("esperava debugID gerado quando header ausente")
	}
}
