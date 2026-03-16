#!/usr/bin/env bash
set -euo pipefail

HOSTS_FILE="${1:-/opt/noc-guardian/linux_hosts.txt}"

SERVER_URL="${SERVER_URL:-https://nocguardian.bitbyteti.tec.br}"
TENANT_ID="${TENANT_ID:-c5f25c4b-23e1-4f03-a519-58ed96c84fb6}"

if [[ ! -f "$HOSTS_FILE" ]]; then
  echo "Arquivo de hosts não encontrado: $HOSTS_FILE" >&2
  exit 1
fi

echo ">> server_url: $SERVER_URL"
echo ">> tenant_id : $TENANT_ID"
echo ">> hosts     : $HOSTS_FILE"
echo

while IFS= read -r H; do
  [[ -z "$H" ]] && continue
  echo "==== $H ===="

  # backup do config atual
  ssh -o BatchMode=yes "$H" "sudo mkdir -p /etc/nocguardian; \
    if [ -f /etc/nocguardian/agent.yml ]; then sudo cp -a /etc/nocguardian/agent.yml /etc/nocguardian/agent.yml.bak.\$(date +%F-%H%M%S); fi"

  # agent_id único por host (machine-id)
  MID="$(ssh -o BatchMode=yes "$H" "cat /etc/machine-id 2>/dev/null || true")"
  if [[ -z "$MID" ]]; then
    echo "!! $H: /etc/machine-id vazio, abortando para evitar agent_id duplicado." >&2
    continue
  fi

  # escreve o YAML no formato exato do seu agente
  ssh -o BatchMode=yes "$H" "sudo tee /etc/nocguardian/agent.yml >/dev/null" <<YAML
# NOC Guardian Agent (Linux)
server_url: $SERVER_URL
tenant_id: $TENANT_ID
# agent_id único por host
agent_id: $MID
YAML

  # reinicia o serviço
  ssh -o BatchMode=yes "$H" "sudo systemctl restart nocguardian-agent"

  # status resumido + últimas linhas do journal
  ssh -o BatchMode=yes "$H" "\
    echo 'active:'; systemctl is-active nocguardian-agent || true; \
    echo 'status (topo):'; systemctl --no-pager -l status nocguardian-agent | head -n 16 || true; \
    echo 'journal (ultimas 12):'; journalctl -u nocguardian-agent --no-pager -n 12 | tail -n 12 || true \
  "

  echo
done < "$HOSTS_FILE"

echo "Concluído."
