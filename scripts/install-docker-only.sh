#!/bin/bash

# 🐳 Script para Instalar apenas Docker na VPS
# Mais rápido e simples que o install-vps.sh

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "🐳 Instalação Docker - CrypGo Trading Bot"
echo "========================================"
echo ""

# 1. Atualizar sistema
echo -e "${BLUE}[1/4]${NC} Atualizando sistema..."
apt update && apt upgrade -y

# 2. Instalar Docker
echo -e "${BLUE}[2/4]${NC} Instalando Docker..."
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh
    rm get-docker.sh
    
    # Iniciar Docker
    systemctl start docker
    systemctl enable docker
    
    echo -e "${GREEN}[SUCCESS]${NC} Docker instalado"
else
    echo -e "${YELLOW}[INFO]${NC} Docker já instalado"
fi

# 3. Instalar Docker Compose
echo -e "${BLUE}[3/4]${NC} Instalando Docker Compose..."
if ! command -v docker-compose &> /dev/null; then
    curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
    echo -e "${GREEN}[SUCCESS]${NC} Docker Compose instalado"
else
    echo -e "${YELLOW}[INFO]${NC} Docker Compose já instalado"
fi

# 4. Configurar firewall
echo -e "${BLUE}[4/4]${NC} Configurando firewall..."
if command -v ufw &> /dev/null; then
    ufw --force enable
    ufw allow ssh
    ufw allow 80/tcp    # Nginx
    ufw allow 8080/tcp  # API direta
    ufw allow 5432/tcp  # PostgreSQL (se necessário)
    ufw allow 5672/tcp  # RabbitMQ
    ufw allow 15672/tcp # RabbitMQ Management
    echo -e "${GREEN}[SUCCESS]${NC} Firewall configurado"
else
    apt install -y ufw
    ufw --force enable
    ufw allow ssh
    ufw allow 80/tcp
    ufw allow 8080/tcp
    echo -e "${GREEN}[SUCCESS]${NC} Firewall instalado e configurado"
fi

echo ""
echo -e "${GREEN}[SUCCESS]${NC} 🎉 Instalação Docker concluída!"
echo ""
echo -e "${BLUE}[INFO]${NC} Versões instaladas:"
echo "- Docker: $(docker --version)"
echo "- Docker Compose: $(docker-compose --version)"
echo ""
echo -e "${BLUE}[PRÓXIMOS PASSOS]${NC}"
echo "1. Transferir projeto para /opt/crypgo-machine/"
echo "2. Editar .env.production com suas chaves"
echo "3. Executar: ./scripts/deploy-docker.sh"