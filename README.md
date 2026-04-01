# NOC Guardian v1

NOC-Guardian: Sistema de Observabilidade - Proatividade - Correção
O NOC-Guardian é uma plataforma moderna de Network Operations Center (NOC) baseada em uma arquitetura de Inteligência Distribuída. Ele foi projetado para ambientes Multi-Tenant de missão crítica, priorizando alta segurança, isolamento de dados e monitoramento proativo.

Diferente de ferramentas legadas monolíticas, o NOC-Guardian elimina gargalos e pontos únicos de falha ao distribuir a coleta, o processamento e a inteligência de forma segura e escalável entre a nuvem e a borda (edge).

🧠 Arquitetura de Inteligência Distribuída
O sistema opera em três camadas independentes e colaborativas:

1. Guardian Central (Cloud)
O cérebro do sistema, hospedado centralmente.

Interface: API e Dashboard Multi-Tenant (React + Tailwind).

Persistência: Banco de dados central (configurações) e bancos de dados tenants (TimescaleDB) separados e isolados por cliente.

Inteligência: Motor de correlação de eventos, Análise de Causa Raiz (RCA) e orquestração de alertas/automações.

2. Guardian PROXY (Edge Proxy)
O coletor inteligente local, implantado na rede do cliente.

Função: Atua como um gateway de monitoramento local.

Segurança: Comunicação outbound-only (sem portas de entrada abertas no cliente). Realiza criptografia e sanitização de dados antes do envio.

Resiliência: Possui buffer local (SQLite) para contingência em caso de falha na internet, garantindo que nenhum dado seja perdido.

3. Guardian Agents
Serviços leves implantados diretamente nos ativos do cliente.

Plataformas: Suporte nativo para Windows e Linux.

Função: Coleta contínua de telemetria detalhada do sistema operacional e aplicações.

Comunicação: Envia dados de forma segura para o Guardian Proxy local ou diretamente para o Guardian Central (configurável).

🧠 IA & AIOps com Protocolo MCP (O Diferencial)
A inteligência do NOC-Guardian não executa comandos cegamente; ela analisa o contexto completo para fornecer insights acionáveis.

O Protocolo MCP (Model Context Protocol)
Utilizamos o padrão emergente MCP para integrar Modelos de Linguagem de Grande Porte (LLMs) diretamente ao fluxo de triagem e remediação. Isso permite que a IA "converse" com a infraestrutura de forma padronizada.

Como funciona o fluxo de IA no NOC-Guardian:

Entrada de Contexto (via MCP): A IA recebe métricas brutas, logs, histórico de desempenho e alertas correlacionados de múltiplas fontes.

Análise Contextual: O LLM utiliza o protocolo MCP para consultar definições de infraestrutura e bases de conhecimento.

Entrega da IA (Output):

📌 Identificação da Causa Raiz: Diagnóstico preciso do problema real, não apenas do sintoma.

📌 Análise de Impacto: Determina quais serviços e usuários são afetados.

📌 Sugestão de Correção: Passo a passo detalhado para resolver o incidente.

📌 Comandos Recomendados (Ansible): Gera playbooks ou comandos Ansible específicos e seguros para a correção.

📌 Prioridade Dinâmica: Ajusta a severidade do alerta com base no risco de negócio.

🌐 Escopo de Análise da Infraestrutura
O NOC-Guardian realiza um monitoramento profundo em todas as camadas da infraestrutura:

🌐 Rede (LAN / WAN / Internet / Portas)
Métricas detalhadas e verificações de integridade de serviços.

Conectividade: Ping, Jitter, Latência, Perda de Pacotes, Traceroute.

Desempenho: Throughput, Variância de Latência, Detecção de Mudança de Rota.

Integridade: DNS Health Check, Gateway Reachability, MTU Check.

Sintéticos: HTTP/HTTPS Synthetic, DNS Q&A.

Segurança de Rede: Varredura de Portas Abertas (Port Scanning).

Protocolos: Telnet/SSH, HTTP/HTTPS, SMTP/IMAP, DNS.

Wireless: Wi-Fi (Status AP, Interferência, RSSI, Clientes, Controladora).

🖥️ Servidores (Windows / Linux)
Monitoramento de saúde do Sistema Operacional.

Recursos: CPU, Memória, Disco (Uso/Espaço).

Performance: Disco IO, Throughput de Rede.

Hardware: Temperatura (via IPMI/SNMP).

Aplicações: Status de Serviços Críticos.

🗄️ Storages / Servidores Físicos
Foco na integridade do hardware e persistência de dados.

Disco: S.M.A.R.T., Latência de Disco, Status de RAID.

Ambiente: Temperatura, Status de Ventoinhas/Fontes.

Proatividade: Detecção de falha iminente.

🧩 Virtualização (VMware/Hyper-V/Proxmox)
Monitoramento do Hypervisor e impacto nas VMs.

Métricas do Host: CPU Ready, Memory Ballooning, Overcommit, Storage Latency, Throughput.

Métricas da VM: CPU, Memória, Disco, Throughput.

Contexto: Identificação de VM impactada por vizinhos barulhentos (noisy neighbors).

📡 Switches / APs / Firewalls (Infra de Borda)
Wireless: Status AP, Interferência, RSSI, Clientes, Syslog.

Firewalls: Sessões Ativas, VPNs Ativas, Logs de Segurança, Interfaces, Rotas, CPU, Memória.

Desempenho: Throughput (MikroTik/Appliances), Interfaces, Fila (Queueing).

🛡️ Compliance / Updates (Visão de Risco)
Gerenciamento de Patches: Status de Windows Update e Linux Updates.

Firmware: Verificação de versões de appliances de rede.

Vulnerabilidades: Correlação de ativos com bases de CVEs conhecidas.

📊 Interface NOC & Dashboard Customizável
A interface do usuário é projetada para fornecer clareza imediata e capacidade de resposta rápida.

### Recursos Principais da UI
Visão Multi-Tenant: Isolamento visual completo entre clientes.

KPIs em Tempo Real: Métricas críticas atualizadas via WebSocket.

Topologia de Rede: Visualização dinâmica e interativa das conexões.

Diagnóstico Interativo: Execução de ferramentas como Ping e MTR diretamente do navegador.

🔧 Ambiente de Dashboard Customizável (Drag & Drop)
O NOC-Guardian oferece um ambiente onde cada cliente pode personalizar sua visualização operacional.

Tecnologia: Implementado com React-Grid-Layout no frontend e persistência JSON no backend Go.

Sistema de Grade: Os widgets magnéticos se alinham automaticamente a uma grade invisível, garantindo organização.

Customização Livre: O cliente pode adicionar, remover, redimensionar e reposicionar componentes (gráficos, KPIs, tabelas de alertas) conforme sua necessidade.

Modo de Edição: A interface possui um "Modo de Edição" dedicado para evitar alterações acidentais durante a operação normal.

🔒 Segurança por Design & Hardening
A segurança não é um recurso opcional, é a base da arquitetura do NOC-Guardian.

Comunicação: Criptografia de ponta a ponta via TLS 1.3 com suporte opcional para mTLS (Mutual TLS) entre Central, Proxies e Agentes.

Criptografia: Dados sensíveis são criptografados com AES-256 (GCM) antes do envio.

Zero Trust: Arquitetura outbound-only no cliente. Nenhuma porta de entrada precisa ser aberta no firewall do cliente para o monitoramento funcionar.

Gerenciamento de Identidade: Uso de certificados rotativos e PKI interna robusta.

Sanitização de Dados: Processo automático de scrubbing remove PII (Informações Pessoais Identificáveis) e senhas dos logs e payloads antes da ingestão.

📦 Deploy & Hardening do Guardian Proxy (Obrigatório)
O deploy do Proxy é simplificado, mas segue padrões rígidos de segurança.

Containerização: Baseado em Docker, compatível com x86-64 e ARM (ex: Raspberry Pi).

Segurança Automática: A comunicação segura é configurada automaticamente no boot.

Scripts de Hardening: O ambiente de deploy inclui scripts obrigatórios para:

Configuração de UFW/Firewall local.

Ativação do Fail2Ban.

Autenticação SSH apenas por chave.

Desativação do login de Root.

Hardening do Kernel Linux.

Ativação de atualizações automáticas do S.O.

🚀 Tecnologias
Frontend: React + Tailwind CSS

Backend: Go (API REST, WebSockets, RCA, Ingestão)

Automação & Remediação: Ansible

Banco de Dados: PostgreSQL + TimeScaleDB (para séries temporais de alta performance)

Infraestrutura: Docker Multi-Arch

📂 Estrutura do Projeto
Plaintext

noc-guardian/
├─ central/         # Backend "Guardian Central" (API, WebSockets, RCA, ingestão)
│  ├─ cmd/server/   # Binário principal do servidor (Go entrypoint)
│  ├─ internal/     # Código interno (http, ws, db, rca, auth, mcp)
│  ├─ migrations/   # Scripts SQL (TimescaleDB)
│  └─ deploy/       # Manifests e Dockerfile do Central
├─ proxy/           # "Guardian Proxy" (Edge Collector)
│  ├─ internal/     # snmp, buffer (SQLite), tunnel, crypto, transport
│  └─ deploy/       # Dockerfile e scripts de instalação
├─ agents/          # Agentes (Linux/Windows) em Go
│  ├─ internal/     # metrics, system, transport
│  └─ deploy/       # installers, service files
├─ ui/              # UI (React + Tailwind)
│  ├─ src/          # components, dashboards, hooks (grid-layout)
│  └─ deploy/       # Dockerfile (Nginx)
├─ infra/           # compose, nginx, hardening, pki
├─ README.md        # Guia principal
└─ Makefile        # Atalhos (build/run/test)

📜 Regra de Ouro
Todo código do NOC-Guardian é educacional, extensivamente documentado e comentado linha por linha. Isso garante transparência total, facilita a manutenção e permite a evolução segura da plataforma por equipes de desenvolvimento e segurança.

💡 Sugestões de Melhoria, Segurança e Compliance
Melhorias de Produto
Observabilidade de Aplicação (APM): Expandir os agentes para coletar métricas de APM (ex: OpenTelemetry) nativamente para linguagens populares.

RCA Visual: Implementar uma visualização de grafo interativo que mostre o caminho exato da Causa Raiz de um incidente.

ChatOps: Integrar a IA (via MCP) com Slack/Microsoft Teams para triagem de alertas via chat.

Segurança Avançada
WAF: Implementar um Web Application Firewall à frente da API Central.

Análise de Comportamento (UEBA): Usar IA para detectar anomalias no comportamento de acesso de usuários ao dashboard.

Segurança de Supply Chain: Implementar assinatura de imagens Docker e SBOM (Software Bill of Materials) para todos os componentes.

### Compliance para o Cliente

ISO 27001: O isolamento de dados por banco e a sanitização automática facilitam a auditoria para a certificação ISO 27001 do cliente.

LGPD/GDPR: O processo de scrubbing de dados garante que nenhuma PII seja armazenada centralmente sem necessidade.

Auditoria de Ações: Implementar log de auditoria imutável de todas as ações tomadas no dashboard e automações Ansible executadas.
