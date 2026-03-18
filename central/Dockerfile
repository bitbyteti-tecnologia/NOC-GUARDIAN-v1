# Dockerfile da CENTRAL
# - Build binário Go estático e leve.
# - Executa 'go mod tidy' para gerar go.sum e resolver deps dos imports.

FROM golang:1.22-alpine AS build
WORKDIR /app
RUN apk add --no-cache build-base
# Copia o go.mod (e go.sum se existir) e baixa deps básicas
COPY go.mod ./
RUN go mod download
# Copia o código e ajusta deps conforme imports reais
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o central .

FROM alpine:3.19
WORKDIR /app
COPY --from=build /app/central /app/central
EXPOSE 8080
ENTRYPOINT ["/app/central"]
