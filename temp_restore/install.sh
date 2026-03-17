#!/usr/bin/env bash
# install.sh - Instalador do PROXY no cliente
# - Requisitos: Docker e Docker Buildx (se compilar local).
# - Configura variáveis e sobe o container do Proxy.

set -e

TENANT_ID="$1"
CENTRAL_URL="https://nocguardian.bitbyteti.tec.br"

if [ -z "$TENANT_ID" ]; then
  echo "Uso: $0 <TENANT_ID>"
  exit 1
fi

mkdir -p /opt/noc-guardian-proxy
cd /opt/noc-guardian-proxy

cat <<EOCFG > .env
TENANT_ID=$TENANT_ID
CENTRAL_URL=$CENTRAL_URL
INGEST_ENDPOINT=/api/v1/${TENANT_ID}/metrics/ingest
AUTH_TOKEN=jwt-token-fake-dev
SNMP_COMMUNITY=public
SNMP_TARGETS=10.0.0.0/24
SCAN_INTERVAL_SEC=60
BUFFER_DB=/data/buffer.sqlite
AES_KEY_BASE64=ZmFrZV9rZXlfMzJfYnl0ZXNfQUVTXzI1Nl9HQ00hIQ==
EOCFG

cat <<'EOYML' > docker-compose.yml
services:
  proxy:
    image: nocguardian/proxy:latest
    env_file: .env
    volumes:
      - ./data:/app/data
    restart: unless-stopped
EOYML

docker compose up -d
echo "Proxy instalado e iniciado."
