# Multi-stage build para otimizar o tamanho da imagem final
FROM golang:1.23-alpine AS builder

# Instalar dependências necessárias
RUN apk add --no-cache git ca-certificates tzdata

# Definir diretório de trabalho
WORKDIR /app

# Copiar arquivos de dependências
COPY go.mod go.sum ./

# Baixar dependências
RUN go mod download

# Copiar código fonte
COPY . .

# Compilar aplicação
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o crypgo-machine main.go

# Imagem final (muito menor)
FROM alpine:latest

# Instalar certificados SSL e timezone
RUN apk --no-cache add ca-certificates tzdata

# Criar usuário não-root por segurança
RUN adduser -D -s /bin/sh crypgo

# Definir diretório de trabalho
WORKDIR /app

# Copiar binário da aplicação do stage anterior
COPY --from=builder /app/crypgo-machine .

# Copiar arquivos de configuração
COPY --from=builder /app/.env.production .env

# Copiar arquivos do dashboard web
COPY --from=builder /app/web ./web

# Dar permissões apropriadas
RUN chown -R crypgo:crypgo /app
USER crypgo

# Expor porta da aplicação
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/trading/list || exit 1

# Comando para executar a aplicação
CMD ["./crypgo-machine"]