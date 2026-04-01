CENTRAL_DIR=central
PROXY_DIR=proxy
AGENTS_DIR=agents
UI_DIR=ui

.PHONY: central proxy agents ui docker-up docker-down

central:
	cd $(CENTRAL_DIR) && go build ./cmd/server

proxy:
	cd $(PROXY_DIR) && go build .

agents:
	cd $(AGENTS_DIR) && go build ./cmd/nocguardian-agent

ui:
	cd $(UI_DIR) && npm install && npm run build

docker-up:
	docker compose -f docker/compose.central.yml up -d --build

docker-down:
	docker compose -f docker/compose.central.yml down
