# CHANGELOG — common

## [0.23.0] — 2026-04-22
### Added
- `DebugIDMiddleware` emite novo log `message="incoming request"` na entrada de cada request, com atributo `requestBody` (string, cap 8KB — `requestBodyLogCap`). Body maior é truncado e ganha `requestBodyTruncated=true`. Body do request é preservado/restaurado (handlers funcionam normal).
- Paths em `loginPathsSkipped` (`/cpa/v1/login`) não têm body logado — atributo vira `"[redacted:login]"` pra não vazar senha.

## [0.22.0] — 2026-04-22
### Removed (BREAKING)
- `CORSMiddleware()` e arquivo `cors.go`. CORS é responsabilidade exclusiva do `svc-gateway` (padrão indústria: svcs internal não falam com browser). Callers em cada svc devem remover `r.Use(common.CORSMiddleware())`.
