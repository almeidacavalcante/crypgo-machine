#!/bin/bash

# 📊 Script de Monitoramento - CrypGo Trading Bot
# Conecta na VPS e monitora logs em tempo real

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configurações da VPS
VPS_HOST="crypgo-vps"
PROJECT_PATH="/opt/crypgo-machine"

echo -e "${BLUE}📊 CrypGo Trading Bot - Monitor de Logs${NC}"
echo "========================================"
echo ""

# Função para mostrar menu
show_menu() {
    echo -e "${CYAN}Escolha uma opção:${NC}"
    echo "1. 📱 Logs da aplicação CrypGo (tempo real)"
    echo "2. 🗃️  Logs de todos os serviços"
    echo "3. 🐘 Logs do banco PostgreSQL"
    echo "4. 🐰 Logs do RabbitMQ"
    echo "5. 🌐 Logs do Nginx"
    echo "6. 📋 Status dos containers"
    echo "7. 🔍 Buscar por palavra-chave nos logs"
    echo "8. ⚠️  Apenas erros e warnings"
    echo "9. 🚪 Sair"
    echo ""
    echo -n -e "${YELLOW}Digite sua escolha (1-9): ${NC}"
}

# Função para conectar e executar comando
run_ssh_command() {
    local command="$1"
    echo -e "${BLUE}[INFO]${NC} Conectando na VPS..."
    echo -e "${BLUE}[INFO]${NC} Pressione Ctrl+C para sair dos logs"
    echo ""
    ssh -t ${VPS_HOST} "cd ${PROJECT_PATH} && ${command}"
}

# Função para buscar palavra-chave
search_logs() {
    echo -n -e "${YELLOW}Digite a palavra-chave para buscar: ${NC}"
    read keyword
    if [ ! -z "$keyword" ]; then
        run_ssh_command "docker-compose -f docker-compose.full.yml logs --no-color --since 24h | grep -i '$keyword'"
    else
        echo -e "${RED}[ERROR]${NC} Palavra-chave não pode estar vazia"
    fi
}

# Loop do menu principal
while true; do
    show_menu
    read choice
    echo ""
    
    case $choice in
        1)
            echo -e "${GREEN}[INFO]${NC} Monitorando logs da aplicação CrypGo..."
            run_ssh_command "docker-compose -f docker-compose.full.yml logs -f --since 24h crypgo-app"
            ;;
        2)
            echo -e "${GREEN}[INFO]${NC} Monitorando logs de todos os serviços..."
            run_ssh_command "docker-compose -f docker-compose.full.yml logs -f --since 24h"
            ;;
        3)
            echo -e "${GREEN}[INFO]${NC} Monitorando logs do PostgreSQL..."
            run_ssh_command "docker-compose -f docker-compose.full.yml logs -f --since 24h postgres"
            ;;
        4)
            echo -e "${GREEN}[INFO]${NC} Monitorando logs do RabbitMQ..."
            run_ssh_command "docker-compose -f docker-compose.full.yml logs -f --since 24h rabbitmq"
            ;;
        5)
            echo -e "${GREEN}[INFO]${NC} Monitorando logs do Nginx..."
            run_ssh_command "docker-compose -f docker-compose.full.yml logs -f --since 24h nginx"
            ;;
        6)
            echo -e "${GREEN}[INFO]${NC} Verificando status dos containers..."
            run_ssh_command "docker-compose -f docker-compose.full.yml ps"
            echo ""
            ;;
        7)
            search_logs
            echo ""
            ;;
        8)
            echo -e "${GREEN}[INFO]${NC} Monitorando apenas erros e warnings..."
            run_ssh_command "docker-compose -f docker-compose.full.yml logs -f --since 24h | grep -i -E 'error|warn|fatal|exception|panic'"
            ;;
        9)
            echo -e "${GREEN}[INFO]${NC} Saindo do monitor..."
            exit 0
            ;;
        *)
            echo -e "${RED}[ERROR]${NC} Opção inválida. Escolha entre 1-9."
            echo ""
            ;;
    esac
done