# CHANGELOG — common

## [0.25.0] — 2026-04-25
### BREAKING (ADR-013 — extração IDP pra idp-colmeia)
- `JWTIssuer`: const → var, default `http://idp-colmeia:8088/idp-colmeia`. Override via env `CPA_JWT_ISSUER`. Antes: `"svc-login"`.
- `CpaClaims` JSON tags renomeadas:
  - `cpa_prefeitura_id` → `tenant_uuid` (genérico, sem ref a CPA)
  - `cpa_email` → `email` (OIDC standard, drop prefix)
- Go fields preservados (`CpaPrefeituraID`, `CpaEmail`) — accessors em svcs continuam funcionando. Só wire format JWT muda.
- Tokens emitidos com issuer/claims antigos viram inválidos após esse bump. 1h TTL = max 1h disruption durante deploy.

## [0.24.1] — 2026-04-23
### Changed
- `DefaultQuotas.PedidosAtivos`: 500 → 1_000_000. Pedidos são temporais (crescem com uso); limite de 500 era restritivo pra operação real. 1M efetivamente ilimitado pra escala de prefeitura municipal.

## [0.24.0] — 2026-04-23
### Added
- `quotas.go`: struct `Quotas` + `DefaultQuotas` com limites hardcoded por recurso (tratores:20, ferramentas:30, motoristas:30, funcionarios:50, agricultores:2000, terras:6000, users:100, pedidosAtivos:500). Source of truth único — sem override per-tenant, sem persistência em DB. Enforcement in-process em cada svc que faz POST.
- `ErrCodeQuotaExceeded = "CPA_ERROR_QUOTA_EXCEEDED"` — errorCode padronizado retornado em HTTP 422 quando POST excederia quota.

## [0.23.0] — 2026-04-22
### Added
- `DebugIDMiddleware` emite novo log `message="incoming request"` na entrada de cada request, com atributo `requestBody` (string, cap 8KB — `requestBodyLogCap`). Body maior é truncado e ganha `requestBodyTruncated=true`. Body do request é preservado/restaurado (handlers funcionam normal).
- Paths em `loginPathsSkipped` (`/cpa/v1/login`) não têm body logado — atributo vira `"[redacted:login]"` pra não vazar senha.

## [0.22.0] — 2026-04-22
### Removed (BREAKING)
- `CORSMiddleware()` e arquivo `cors.go`. CORS é responsabilidade exclusiva do `svc-gateway` (padrão indústria: svcs internal não falam com browser). Callers em cada svc devem remover `r.Use(common.CORSMiddleware())`.
