# CHANGELOG — common

## [0.22.0] — 2026-04-22
### Removed (BREAKING)
- `CORSMiddleware()` e arquivo `cors.go`. CORS é responsabilidade exclusiva do `svc-gateway` (padrão indústria: svcs internal não falam com browser). Callers em cada svc devem remover `r.Use(common.CORSMiddleware())`.
