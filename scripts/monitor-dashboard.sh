#!/bin/bash

# 🖥️ Dashboard de Monitoramento - CrypGo Trading Bot
# Dashboard avançado com múltiplas janelas e informações em tempo real

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m'

# Configurações da VPS
VPS_HOST="31.97.249.4"
VPS_USER="root"
PROJECT_PATH="/opt/crypgo-machine"

# Verificar se tmux está instalado
check_tmux() {
    if ! command -v tmux &> /dev/null; then
        echo -e "${RED}[ERROR]${NC} tmux não está instalado!"
        echo "Instale o tmux primeiro:"
        echo "  macOS: brew install tmux"
        echo "  Ubuntu: sudo apt install tmux"
        echo "  CentOS: sudo yum install tmux"
        exit 1
    fi
}

# Função para criar sessão tmux com dashboard
create_dashboard() {
    local session_name="crypgo-dashboard"
    
    echo -e "${BLUE}🖥️ CrypGo Trading Bot - Dashboard de Monitoramento${NC}"
    echo "=================================================="
    echo ""
    echo -e "${YELLOW}[INFO]${NC} Criando dashboard com tmux..."
    echo -e "${YELLOW}[INFO]${NC} Pressione Ctrl+B e depois D para sair (detach)"
    echo -e "${YELLOW}[INFO]${NC} Use 'tmux attach -t ${session_name}' para retornar"
    echo ""
    sleep 2
    
    # Remover sessão existente se houver
    tmux kill-session -t ${session_name} 2>/dev/null || true
    
    # Criar nova sessão
    tmux new-session -d -s ${session_name}
    
    # Janela 1: Logs da aplicação CrypGo
    tmux rename-window -t ${session_name}:0 "CrypGo-App"
    tmux send-keys -t ${session_name}:0 "ssh -t ${VPS_USER}@${VPS_HOST} 'cd ${PROJECT_PATH} && echo -e \"${GREEN}📱 LOGS DA APLICAÇÃO CRYPGO${NC}\" && docker-compose -f docker-compose.full.yml logs -f --tail 50 crypgo-app'" Enter
    
    # Janela 2: Status dos containers
    tmux new-window -t ${session_name} -n "Status"
    tmux send-keys -t ${session_name}:1 "while true; do clear; ssh ${VPS_USER}@${VPS_HOST} 'cd ${PROJECT_PATH} && echo -e \"${CYAN}📊 STATUS DOS CONTAINERS - \$(date)${NC}\" && echo && docker-compose -f docker-compose.full.yml ps && echo && echo -e \"${BLUE}💾 USO DE RECURSOS:${NC}\" && docker stats --no-stream --format \"table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\"'; sleep 30; done" Enter
    
    # Janela 3: Logs do banco PostgreSQL
    tmux new-window -t ${session_name} -n "PostgreSQL"
    tmux send-keys -t ${session_name}:2 "ssh -t ${VPS_USER}@${VPS_HOST} 'cd ${PROJECT_PATH} && echo -e \"${PURPLE}🐘 LOGS DO POSTGRESQL${NC}\" && docker-compose -f docker-compose.full.yml logs -f --tail 30 postgres'" Enter
    
    # Janela 4: Logs do RabbitMQ
    tmux new-window -t ${session_name} -n "RabbitMQ"
    tmux send-keys -t ${session_name}:3 "ssh -t ${VPS_USER}@${VPS_HOST} 'cd ${PROJECT_PATH} && echo -e \"${YELLOW}🐰 LOGS DO RABBITMQ${NC}\" && docker-compose -f docker-compose.full.yml logs -f --tail 30 rabbitmq'" Enter
    
    # Janela 5: Logs do Nginx
    tmux new-window -t ${session_name} -n "Nginx"
    tmux send-keys -t ${session_name}:4 "ssh -t ${VPS_USER}@${VPS_HOST} 'cd ${PROJECT_PATH} && echo -e \"${CYAN}🌐 LOGS DO NGINX${NC}\" && docker-compose -f docker-compose.full.yml logs -f --tail 30 nginx'" Enter
    
    # Janela 6: Erros e Warnings
    tmux new-window -t ${session_name} -n "Errors"
    tmux send-keys -t ${session_name}:5 "ssh -t ${VPS_USER}@${VPS_HOST} 'cd ${PROJECT_PATH} && echo -e \"${RED}⚠️ ERROS E WARNINGS${NC}\" && docker-compose -f docker-compose.full.yml logs -f --tail 100 | grep -i --color=always -E \"error|warn|fatal|exception|panic|ERROR|WARN|FATAL\"'" Enter
    
    # Voltar para a primeira janela
    tmux select-window -t ${session_name}:0
    
    # Anexar à sessão
    tmux attach-session -t ${session_name}
}

# Função para mostrar menu de opções
show_menu() {
    echo -e "${BLUE}🖥️ CrypGo Trading Bot - Dashboard de Monitoramento${NC}"
    echo "=================================================="
    echo ""
    echo -e "${CYAN}Escolha uma opção:${NC}"
    echo "1. 🚀 Iniciar Dashboard Completo (tmux)"
    echo "2. 📱 Monitor Simples - Logs da App"
    echo "3. 📊 Monitor de Status e Recursos"
    echo "4. 🔍 Monitor de Erros em Tempo Real"
    echo "5. 🌐 Teste de Conectividade da API"
    echo "6. 🗂️  Anexar a Dashboard Existente"
    echo "7. 🚪 Sair"
    echo ""
    echo -n -e "${YELLOW}Digite sua escolha (1-7): ${NC}"
}

# Função para monitor simples
simple_monitor() {
    echo -e "${GREEN}[INFO]${NC} Monitorando logs da aplicação..."
    echo -e "${BLUE}[INFO]${NC} Pressione Ctrl+C para sair"
    echo ""
    ssh -t ${VPS_USER}@${VPS_HOST} "cd ${PROJECT_PATH} && docker-compose -f docker-compose.full.yml logs -f --tail 50 crypgo-app"
}

# Função para monitor de status
status_monitor() {
    echo -e "${GREEN}[INFO]${NC} Monitorando status e recursos..."
    echo -e "${BLUE}[INFO]${NC} Pressione Ctrl+C para sair"
    echo ""
    while true; do
        clear
        echo -e "${CYAN}📊 CrypGo Trading Bot - Status - $(date)${NC}"
        echo "=============================================="
        echo ""
        
        ssh ${VPS_USER}@${VPS_HOST} "cd ${PROJECT_PATH} && docker-compose -f docker-compose.full.yml ps"
        echo ""
        echo -e "${BLUE}💾 Uso de Recursos:${NC}"
        ssh ${VPS_USER}@${VPS_HOST} "docker stats --no-stream --format 'table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}'"
        echo ""
        echo -e "${YELLOW}[INFO]${NC} Atualizando em 30 segundos..."
        sleep 30
    done
}

# Função para monitor de erros
error_monitor() {
    echo -e "${RED}[INFO]${NC} Monitorando erros e warnings..."
    echo -e "${BLUE}[INFO]${NC} Pressione Ctrl+C para sair"
    echo ""
    ssh -t ${VPS_USER}@${VPS_HOST} "cd ${PROJECT_PATH} && docker-compose -f docker-compose.full.yml logs -f --tail 100 | grep -i --color=always -E 'error|warn|fatal|exception|panic|ERROR|WARN|FATAL'"
}

# Função para teste de conectividade
test_api() {
    echo -e "${GREEN}[INFO]${NC} Testando conectividade da API..."
    echo ""
    
    echo -e "${BLUE}[TEST]${NC} Testando endpoint direto (porta 8080)..."
    if curl -s --max-time 5 http://${VPS_HOST}:8080/api/v1/health > /dev/null; then
        echo -e "${GREEN}✅ API direta: OK${NC}"
    else
        echo -e "${RED}❌ API direta: FALHOU${NC}"
    fi
    
    echo -e "${BLUE}[TEST]${NC} Testando endpoint via Nginx (porta 80)..."
    if curl -s --max-time 5 http://${VPS_HOST}/api/v1/health > /dev/null; then
        echo -e "${GREEN}✅ API via Nginx: OK${NC}"
    else
        echo -e "${RED}❌ API via Nginx: FALHOU${NC}"
    fi
    
    echo -e "${BLUE}[TEST]${NC} Testando RabbitMQ Management..."
    if curl -s --max-time 5 http://${VPS_HOST}:15672 > /dev/null; then
        echo -e "${GREEN}✅ RabbitMQ Management: OK${NC}"
    else
        echo -e "${RED}❌ RabbitMQ Management: FALHOU${NC}"
    fi
    
    echo ""
    echo -e "${YELLOW}[INFO]${NC} Pressione Enter para continuar..."
    read
}

# Função para anexar a dashboard existente
attach_dashboard() {
    if tmux has-session -t crypgo-dashboard 2>/dev/null; then
        echo -e "${GREEN}[INFO]${NC} Anexando à dashboard existente..."
        tmux attach-session -t crypgo-dashboard
    else
        echo -e "${RED}[ERROR]${NC} Nenhuma dashboard ativa encontrada."
        echo -e "${YELLOW}[INFO]${NC} Use a opção 1 para criar uma nova dashboard."
        echo ""
    fi
}

# Verificar dependências
check_tmux

# Loop do menu principal
while true; do
    show_menu
    read choice
    echo ""
    
    case $choice in
        1)
            create_dashboard
            ;;
        2)
            simple_monitor
            echo ""
            ;;
        3)
            status_monitor
            echo ""
            ;;
        4)
            error_monitor
            echo ""
            ;;
        5)
            test_api
            ;;
        6)
            attach_dashboard
            echo ""
            ;;
        7)
            echo -e "${GREEN}[INFO]${NC} Saindo do monitor..."
            exit 0
            ;;
        *)
            echo -e "${RED}[ERROR]${NC} Opção inválida. Escolha entre 1-7."
            echo ""
            ;;
    esac
done