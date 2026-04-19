package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HeaderRequestID é o header padrão de correlação (industry standard).
// Internamente mantemos nome "debugID" no ctx + logs; só o wire format muda.
const HeaderRequestID = "X-Request-Id"

// DoRequest propaga o debugID do contexto gin via X-Request-Id e executa a chamada.
// Se o ctx não tem debugID (ex: chamadas de startup), envia sem header — o serviço
// destino gera um novo debugID.
func DoRequest(c *gin.Context, req *http.Request) (*http.Response, error) {
	if id := DebugIDFromCtx(c); id != "" {
		req.Header.Set(HeaderRequestID, id)
	}
	return http.DefaultClient.Do(req)
}
