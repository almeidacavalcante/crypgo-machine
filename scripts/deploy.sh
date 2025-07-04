#!/bin/bash

# 🚀 Script Principal de Deploy do CrypGo Trading Bot
# Orquestra todo o processo de deploy na VPS

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "🚀 CrypGo Trading Bot - Deploy Script"
echo "====================================="
echo ""

# Verificar se estamos no diretório correto
if [ ! -f "go.mod" ] || [ ! -f "main.go" ]; then
    echo -e "${RED}[ERROR]${NC} Execute este script no diretório raiz do projeto!"
    exit 1
fi

# 1. Configurar ambiente de produção
echo -e "${BLUE}[STEP 1/8]${NC} Configurando ambiente de produção..."
if [ ! -f ".env.production" ]; then
    echo -e "${RED}[ERROR]${NC} Arquivo .env.production não encontrado!"
    echo "Por favor, configure suas credenciais em .env.production"
    exit 1
fi

# Usar .env.production
cp .env.production .env
echo -e "${GREEN}[SUCCESS]${NC} Ambiente de produção configurado"

# 2. Subir serviços de infraestrutura
echo -e "${BLUE}[STEP 2/8]${NC} Iniciando serviços de infraestrutura..."
docker-compose -f docker-compose.production.yml up -d postgres rabbitmq

# Aguardar serviços iniciarem
echo -e "${YELLOW}[WAIT]${NC} Aguardando serviços iniciarem (30 segundos)..."
sleep 30

echo -e "${GREEN}[SUCCESS]${NC} Serviços de infraestrutura iniciados"

# 3. Aplicar migrations
echo -e "${BLUE}[STEP 3/8]${NC} Aplicando migrations do banco de dados..."
./scripts/run-migrations.sh
echo -e "${GREEN}[SUCCESS]${NC} Migrations aplicadas"

# 4. Baixar dependências Go
echo -e "${BLUE}[STEP 4/8]${NC} Baixando dependências Go..."
go mod download
echo -e "${GREEN}[SUCCESS]${NC} Dependências baixadas"

# 5. Compilar aplicação
echo -e "${BLUE}[STEP 5/8]${NC} Compilando aplicação..."
go build -ldflags="-w -s" -o crypgo-machine main.go
echo -e "${GREEN}[SUCCESS]${NC} Aplicação compilada"

# 6. Configurar como serviço systemd
echo -e "${BLUE}[STEP 6/8]${NC} Configurando serviço systemd..."
sudo cp scripts/crypgo-machine.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable crypgo-machine
echo -e "${GREEN}[SUCCESS]${NC} Serviço systemd configurado"

# 7. Criar diretório de logs
echo -e "${BLUE}[STEP 7/8]${NC} Configurando logs..."
sudo mkdir -p /var/log/crypgo-machine
sudo chown $(whoami):$(whoami) /var/log/crypgo-machine
echo -e "${GREEN}[SUCCESS]${NC} Diretório de logs criado"

# 8. Iniciar aplicação
echo -e "${BLUE}[STEP 8/8]${NC} Iniciando aplicação..."
sudo systemctl start crypgo-machine
sleep 5

# Verificar status
if sudo systemctl is-active --quiet crypgo-machine; then
    echo -e "${GREEN}[SUCCESS]${NC} ✅ Aplicação iniciada com sucesso!"
else
    echo -e "${RED}[ERROR]${NC} ❌ Falha ao iniciar aplicação"
    echo "Verificando logs:"
    sudo journalctl -u crypgo-machine --no-pager -l
    exit 1
fi

echo ""
echo "🎉 Deploy concluído com sucesso!"
echo ""
echo -e "${BLUE}[INFO]${NC} Informações importantes:"
echo "• Aplicação rodando na porta 8080"
echo "• Logs: sudo journalctl -u crypgo-machine -f"
echo "• Status: sudo systemctl status crypgo-machine"
echo "• Parar: sudo systemctl stop crypgo-machine"
echo "• Reiniciar: sudo systemctl restart crypgo-machine"
echo ""
echo -e "${BLUE}[INFO]${NC} Endpoints disponíveis:"
echo "• Health Check: curl http://localhost:8080/api/v1/trading/list"
echo "• RabbitMQ Management: http://$(hostname -I | awk '{print $1}'):15672"
echo ""
echo -e "${BLUE}[INFO]${NC} Para fazer backup do banco:"
echo "• Execute: ./scripts/backup-database.sh"
echo ""
echo -e "${GREEN}[SUCCESS]${NC} 🚀 CrypGo Trading Bot está rodando!"