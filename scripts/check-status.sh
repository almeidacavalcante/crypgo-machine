#!/bin/bash

# 🔍 Script de Diagnóstico do CrypGo Trading Bot
# Verifica o status completo da aplicação e infraestrutura

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "🔍 CrypGo Trading Bot - Diagnóstico Completo"
echo "============================================="
echo ""

# 1. Verificar aplicação principal
echo -e "${BLUE}[1/7]${NC} Verificando aplicação principal..."
if systemctl is-active --quiet crypgo-machine; then
    echo -e "${GREEN}[✅ OK]${NC} Aplicação está rodando"
    echo -e "${BLUE}[INFO]${NC} Status detalhado:"
    systemctl status crypgo-machine --no-pager -l
else
    echo -e "${RED}[❌ ERRO]${NC} Aplicação não está rodando"
    echo -e "${YELLOW}[INFO]${NC} Últimos logs de erro:"
    journalctl -u crypgo-machine --no-pager -l --since "10 minutes ago"
fi
echo ""

# 2. Verificar porta 8080
echo -e "${BLUE}[2/7]${NC} Verificando porta 8080..."
if netstat -tlnp 2>/dev/null | grep -q ":8080"; then
    echo -e "${GREEN}[✅ OK]${NC} Porta 8080 está em uso"
    netstat -tlnp | grep ":8080"
else
    echo -e "${RED}[❌ ERRO]${NC} Porta 8080 não está em uso"
    echo -e "${YELLOW}[INFO]${NC} Portas abertas:"
    netstat -tlnp | grep LISTEN | head -10
fi
echo ""

# 3. Verificar Docker containers
echo -e "${BLUE}[3/7]${NC} Verificando containers Docker..."
if command -v docker &> /dev/null; then
    if docker ps | grep -q postgres; then
        echo -e "${GREEN}[✅ OK]${NC} PostgreSQL container rodando"
    else
        echo -e "${RED}[❌ ERRO]${NC} PostgreSQL container não está rodando"
    fi
    
    if docker ps | grep -q rabbitmq; then
        echo -e "${GREEN}[✅ OK]${NC} RabbitMQ container rodando"
    else
        echo -e "${RED}[❌ ERRO]${NC} RabbitMQ container não está rodando"
    fi
    
    echo -e "${BLUE}[INFO]${NC} Containers ativos:"
    docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
else
    echo -e "${RED}[❌ ERRO]${NC} Docker não encontrado"
fi
echo ""

# 4. Verificar conexão com banco
echo -e "${BLUE}[4/7]${NC} Verificando conexão com banco..."
if [ -f ".env.production" ]; then
    export $(grep -v '^#' .env.production | xargs)
elif [ -f ".env" ]; then
    export $(grep -v '^#' .env | xargs)
fi

DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-crypgo}
DB_PASSWORD=${DB_PASSWORD:-crypgo}
DB_NAME=${DB_NAME:-crypgo_machine}

if command -v psql &> /dev/null; then
    if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT version();" &>/dev/null; then
        echo -e "${GREEN}[✅ OK]${NC} Conexão com PostgreSQL funcionando"
    else
        echo -e "${RED}[❌ ERRO]${NC} Não conseguiu conectar ao PostgreSQL"
        echo -e "${YELLOW}[INFO]${NC} Testando conexão manual..."
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT version();"
    fi
else
    echo -e "${YELLOW}[WARNING]${NC} psql não encontrado - instalando..."
    apt update && apt install -y postgresql-client
fi
echo ""

# 5. Verificar firewall
echo -e "${BLUE}[5/7]${NC} Verificando firewall..."
if command -v ufw &> /dev/null; then
    if ufw status | grep -q "8080"; then
        echo -e "${GREEN}[✅ OK]${NC} Porta 8080 liberada no firewall"
    else
        echo -e "${YELLOW}[WARNING]${NC} Porta 8080 pode não estar liberada"
        echo -e "${BLUE}[INFO]${NC} Status do firewall:"
        ufw status
    fi
else
    echo -e "${YELLOW}[INFO]${NC} UFW não encontrado"
fi
echo ""

# 6. Verificar arquivo binário
echo -e "${BLUE}[6/7]${NC} Verificando arquivo da aplicação..."
if [ -f "crypgo-machine" ]; then
    echo -e "${GREEN}[✅ OK]${NC} Binário da aplicação existe"
    ls -la crypgo-machine
    echo -e "${BLUE}[INFO]${NC} Testando execução:"
    if ./crypgo-machine --help 2>/dev/null || echo "Binário existe mas pode precisar de compilação"; then
        echo -e "${GREEN}[INFO]${NC} Binário parece funcional"
    fi
else
    echo -e "${RED}[❌ ERRO]${NC} Binário da aplicação não encontrado"
    echo -e "${YELLOW}[INFO]${NC} Execute: go build -o crypgo-machine main.go"
fi
echo ""

# 7. Verificar configurações
echo -e "${BLUE}[7/7]${NC} Verificando configurações..."
if [ -f ".env.production" ]; then
    echo -e "${GREEN}[✅ OK]${NC} Arquivo .env.production encontrado"
elif [ -f ".env" ]; then
    echo -e "${YELLOW}[WARNING]${NC} Usando .env (recomendado: .env.production)"
else
    echo -e "${RED}[❌ ERRO]${NC} Arquivo de configuração não encontrado"
fi

echo -e "${BLUE}[INFO]${NC} Configurações atuais:"
echo "- DB_HOST: $DB_HOST"
echo "- DB_PORT: $DB_PORT" 
echo "- DB_NAME: $DB_NAME"
echo "- DB_USER: $DB_USER"
echo ""

# Resumo e próximos passos
echo "🔍 Diagnóstico concluído!"
echo ""
echo -e "${BLUE}[PRÓXIMOS PASSOS]${NC}"
echo "1. Se aplicação não está rodando: sudo systemctl start crypgo-machine"
echo "2. Se binário não existe: go build -o crypgo-machine main.go"
echo "3. Se banco não conecta: docker-compose -f docker-compose.production.yml up -d"
echo "4. Ver logs: sudo journalctl -u crypgo-machine -f"
echo "5. Testar API: curl http://localhost:8080/api/v1/trading/list"
echo ""
echo -e "${YELLOW}[DICA]${NC} Execute este comando para ver logs em tempo real:"
echo "sudo journalctl -u crypgo-machine -f"