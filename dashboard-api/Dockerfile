FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates

# Copia módulo primeiro (melhor cache)
COPY go.mod ./
# Copia go.sum se existir (não falha se não existir)
COPY go.sum* ./

# Copia TODOS os arquivos Go (inclui host_inventory_latest.go)
COPY *.go ./
# Copia pacotes internos
COPY internal ./internal

# Gera/ajusta dependências (gera go.sum se faltava)
RUN /usr/local/go/bin/go mod tidy

# Compila o pacote inteiro (todos os arquivos do package main)
RUN CGO_ENABLED=0 GOOS=linux /usr/local/go/bin/go build -o /out/dashboard-api .

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=build /out/dashboard-api /usr/bin/dashboard-api
EXPOSE 8090
ENTRYPOINT ["/usr/bin/dashboard-api"]
