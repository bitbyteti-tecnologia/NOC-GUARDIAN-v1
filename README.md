# NOC Guardian v1

Status: operacional em producao e em evolucao ativa.  
Dominio atual: `nocguardian.bitbyteti.tec.br`.

## Visao Geral
Plataforma de monitoramento NOC com:
- **Central API (Go)** para auth, tenants, usuarios, dispositivos e ingest de metricas.
- **Dashboard API (Go)** com endpoints otimizados para o frontend.
- **UI (React + Vite + Tailwind)** para operacao e visualizacao.
- **Agent (Go)** para coleta local de metricas de servidores.
- **Proxy (Go)** para coleta SNMP, buffer local e envio para a Central.
- **Nginx** como reverse proxy com TLS e rate limit.
- **TimescaleDB/Postgres** para armazenamento de metricas e alertas.

## Estrutura do Repositorio
- `central/` API principal (Go)
- `dashboard-api/` API de dashboard (Go)
- `UI/` frontend (React)
- `agent/` agente oficial (Go) e empacotamentos
- `proxy/` coletor SNMP e buffer local (Go)
- `docker/` compose de producao
- `nginx/` configuracao de proxy/TLS
- `downloads/` instaladores de agentes

## Arquitetura (alto nivel)
1. **Agent** coleta metricas e envia para a Central.
2. **Proxy** coleta SNMP + bufferiza quando sem internet.
3. **Central** recebe ingest, armazena em TimescaleDB e expõe API.
4. **Dashboard API** agrega dados para telas.
5. **UI** consome APIs e apresenta dashboards.

## Endpoints Principais
Central:
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `GET /api/v1/auth/me`
- `POST /api/v1/{tenantID}/metrics/ingest`
- `GET /api/v1/{tenantID}/metrics/latest`
- `GET /api/v1/{tenantID}/metrics/range`
- `POST /api/v1/tenants`
- `GET /api/v1/tenants`

Dashboard API:
- `GET /api/v1/tenants/{tenantId}/dashboard/summary`
- `GET /api/v1/tenants/{tenantId}/dashboard/hosts`
- `GET /api/v1/tenants/{tenantId}/dashboard/series`
- `GET /api/v1/tenants/{tenantId}/dashboard/host/{hostname}/inventory/latest`

## Variaveis de Ambiente (principais)
Central (`central/.env`):
- `APP_ENV`, `APP_PORT`
- `MASTER_DB_HOST`, `MASTER_DB_PORT`, `MASTER_DB_USER`, `MASTER_DB_PASS`, `MASTER_DB_NAME`
- `JWT_SECRET`
- `SUPERADMIN_EMAIL`, `SUPERADMIN_PASSWORD`
- `ACCESS_TOKEN_TTL_HOURS`, `REFRESH_TOKEN_TTL_DAYS`
- `COOKIE_DOMAIN`, `COOKIE_SECURE`, `COOKIE_SAME_SITE`
- `PUBLIC_URL` (para links de reset)
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`
- `SMTP_USE_TLS`, `SMTP_STARTTLS`, `RESET_EMAIL_MODE`
- `DIAG_MODE` (`simulate` | `disabled`)

Dashboard API (`dashboard-api/.env` via compose):
- `MASTER_DB_*`, `DASH_LISTEN`, `DASHBOARD_SECRET`

Proxy (`proxy/.env`):
- `TENANT_ID`, `CENTRAL_URL`, `INGEST_ENDPOINT`, `AUTH_TOKEN`
- `SNMP_COMMUNITY`, `SNMP_TARGETS`, `SCAN_INTERVAL_SEC`
- `BUFFER_DB`, `AES_KEY_BASE64`

Agent:
- Config em `/etc/nocguardian/agent.yml` (gerado pelo proprio agente).

## Deploy (producao)
Compose atual em `docker/compose.central.yml`:
- `db`, `central`, `ui`, `dashboard`, `nginx`
- Rede externa: `docker_nocnet`

Fluxo tipico (ajuste conforme seu ambiente):
1. Atualizar repo no servidor
2. `docker compose -f docker/compose.central.yml up -d --build`

## Desenvolvimento
UI:
1. `cd UI`
2. `npm install`
3. `npm run dev`

Central:
1. `cd central`
2. `go run .`

Dashboard API:
1. `cd dashboard-api`
2. `go run .`

## Observacoes de Seguranca
- Nunca versionar `.env` publicamente com segredos reais.
- Revise secrets antes de abrir o repositorio.
- TLS e rate limit estao em `nginx/nginx.conf`.

## Roadmap Imediato
- Envio real de email de reset (SMTP configurado).
- Diagnosticos com execucao real via Proxy.
- RCA com regras baseadas em topologia e metricas.

