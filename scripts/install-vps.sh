#!/bin/bash

# 🚀 Script de Instalação do CrypGo Trading Bot na VPS
# Autor: Claude
# Data: 2025-07-04

set -e  # Parar script se houver erro

echo "🚀 Iniciando instalação do CrypGo Trading Bot na VPS..."

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Função para logs coloridos
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 1. Atualizar sistema
log_info "Atualizando sistema..."
apt update && apt upgrade -y
log_success "Sistema atualizado"

# 2. Instalar ferramentas básicas
log_info "Instalando ferramentas básicas..."
apt install -y curl wget git unzip software-properties-common ufw htop nano
log_success "Ferramentas instaladas"

# 3. Instalar Go 1.23
log_info "Instalando Go 1.23..."
if ! command -v go &> /dev/null; then
    wget -q https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
    rm go1.23.0.linux-amd64.tar.gz
    
    # Adicionar Go ao PATH
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    export PATH=$PATH:/usr/local/go/bin
    
    log_success "Go 1.23 instalado"
else
    log_warning "Go já está instalado"
fi

# 4. Instalar Docker
log_info "Instalando Docker..."
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh
    rm get-docker.sh
    
    # Iniciar Docker
    systemctl start docker
    systemctl enable docker
    
    log_success "Docker instalado e iniciado"
else
    log_warning "Docker já está instalado"
fi

# 5. Instalar Docker Compose
log_info "Instalando Docker Compose..."
if ! command -v docker-compose &> /dev/null; then
    curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
    log_success "Docker Compose instalado"
else
    log_warning "Docker Compose já está instalado"
fi

# 6. Instalar PostgreSQL client (para executar migrations)
log_info "Instalando PostgreSQL client..."
apt install -y postgresql-client
log_success "PostgreSQL client instalado"

# 7. Configurar firewall básico
log_info "Configurando firewall..."
ufw --force enable
ufw allow ssh
ufw allow 8080/tcp
ufw allow 5432/tcp  # PostgreSQL (apenas se necessário)
ufw allow 5672/tcp  # RabbitMQ (apenas se necessário)
ufw allow 15672/tcp # RabbitMQ Management (apenas se necessário)
log_success "Firewall configurado"

# 8. Criar diretório da aplicação
log_info "Criando diretório da aplicação..."
mkdir -p /opt/crypgo-machine
cd /opt/crypgo-machine
log_success "Diretório criado: /opt/crypgo-machine"

# 9. Criar usuário para a aplicação (opcional, por segurança)
log_info "Criando usuário para aplicação..."
if ! id "crypgo" &>/dev/null; then
    useradd -r -s /bin/false crypgo
    log_success "Usuário 'crypgo' criado"
else
    log_warning "Usuário 'crypgo' já existe"
fi

log_success "🎉 Instalação base completa!"
echo ""
log_info "Próximos passos:"
echo "1. Transferir o projeto para /opt/crypgo-machine/"
echo "2. Configurar .env.production"
echo "3. Executar docker-compose up -d"
echo "4. Aplicar migrations do banco"
echo "5. Compilar e executar a aplicação"
echo ""
log_info "Versões instaladas:"
echo "- Go: $(go version)"
echo "- Docker: $(docker --version)"
echo "- Docker Compose: $(docker-compose --version)"
echo "- PostgreSQL Client: $(psql --version)"