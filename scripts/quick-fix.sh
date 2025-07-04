#!/bin/bash

# 🔧 Script de Solução Rápida para Problemas Comuns
# Tenta resolver automaticamente os problemas mais frequentes

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "🔧 CrypGo Trading Bot - Solução Rápida"
echo "======================================"
echo ""

# 1. Parar aplicação se estiver rodando
echo -e "${BLUE}[STEP 1]${NC} Parando aplicação..."
sudo systemctl stop crypgo-machine 2>/dev/null || echo "Aplicação já estava parada"

# 2. Verificar e iniciar containers
echo -e "${BLUE}[STEP 2]${NC} Verificando containers Docker..."
if [ -f "docker-compose.production.yml" ]; then
    echo "Iniciando PostgreSQL e RabbitMQ..."
    docker-compose -f docker-compose.production.yml up -d postgres rabbitmq
    echo "Aguardando containers iniciarem..."
    sleep 15
else
    echo "Usando docker-compose.yml padrão..."
    docker-compose up -d postgres rabbitmq
    sleep 15
fi

# 3. Verificar se binário existe
echo -e "${BLUE}[STEP 3]${NC} Verificando binário da aplicação..."
if [ ! -f "crypgo-machine" ]; then
    echo "Compilando aplicação..."
    go mod download
    go build -ldflags="-w -s" -o crypgo-machine main.go
    echo -e "${GREEN}[SUCCESS]${NC} Aplicação compilada"
else
    echo -e "${GREEN}[OK]${NC} Binário já existe"
fi

# 4. Verificar configurações
echo -e "${BLUE}[STEP 4]${NC} Verificando configurações..."
if [ -f ".env.production" ]; then
    cp .env.production .env
    echo -e "${GREEN}[OK]${NC} Usando .env.production"
elif [ ! -f ".env" ]; then
    echo -e "${RED}[ERROR]${NC} Arquivo .env não encontrado!"
    echo "Criando .env básico..."
    cat > .env << 'EOF'
BINANCE_API_KEY=YOUR_API_KEY_HERE
BINANCE_SECRET_KEY=YOUR_SECRET_KEY_HERE
DB_HOST=localhost
DB_PORT=5432
DB_NAME=crypgo_machine
DB_USER=crypgo
DB_PASSWORD=crypgo
RABBIT_MQ_URL=amqp://guest:guest@localhost:5672/
EOF
    echo -e "${YELLOW}[WARNING]${NC} Configure suas chaves da Binance em .env"
fi

# 5. Aplicar migrations
echo -e "${BLUE}[STEP 5]${NC} Aplicando migrations..."
if [ -f "scripts/run-migrations.sh" ]; then
    ./scripts/run-migrations.sh
else
    echo "Script de migrations não encontrado, aplicando manualmente..."
    # Carregar configurações
    export $(grep -v '^#' .env | xargs)
    
    # Aplicar migrations uma por uma
    for migration in src/infra/database/migrations/*.sql; do
        if [ -f "$migration" ]; then
            echo "Aplicando: $migration"
            PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$migration" || echo "Migration já aplicada ou erro: $migration"
        fi
    done
fi

# 6. Configurar firewall
echo -e "${BLUE}[STEP 6]${NC} Configurando firewall..."
if command -v ufw &> /dev/null; then
    ufw allow 8080/tcp 2>/dev/null || echo "Regra já existe"
    echo -e "${GREEN}[OK]${NC} Porta 8080 liberada"
fi

# 7. Iniciar aplicação
echo -e "${BLUE}[STEP 7]${NC} Iniciando aplicação..."
sudo systemctl start crypgo-machine

# 8. Aguardar e verificar
echo "Aguardando aplicação iniciar..."
sleep 10

# 9. Testar
echo -e "${BLUE}[STEP 8]${NC} Testando aplicação..."
if curl -s http://localhost:8080/api/v1/trading/list &>/dev/null; then
    echo -e "${GREEN}[SUCCESS]${NC} ✅ API está funcionando!"
    echo ""
    echo "🎉 Aplicação está online!"
    echo "• API: http://$(hostname -I | awk '{print $1}'):8080"
    echo "• Teste: curl http://localhost:8080/api/v1/trading/list"
else
    echo -e "${RED}[ERROR]${NC} ❌ API ainda não está respondendo"
    echo ""
    echo "🔍 Verificando logs..."
    sudo journalctl -u crypgo-machine --no-pager -l --since "2 minutes ago"
    echo ""
    echo -e "${YELLOW}[DICA]${NC} Execute para ver logs em tempo real:"
    echo "sudo journalctl -u crypgo-machine -f"
fi

echo ""
echo "🔧 Solução rápida concluída!"