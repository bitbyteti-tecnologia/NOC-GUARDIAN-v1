# Dockerfile do PROXY
# - Multi-arch (buildx), roda sem portas expostas (outbound-only).
FROM golang:1.22-alpine AS build
WORKDIR /app
RUN apk add --no-cache build-base
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o proxy .

FROM alpine:3.19
WORKDIR /app
RUN adduser -D -H -u 10001 proxy
USER proxy
COPY --chown=proxy:proxy .env /app/.env
COPY --chown=proxy:proxy --from=build /app/proxy /app/proxy
VOLUME ["/app/data"]
ENTRYPOINT ["/app/proxy"]
