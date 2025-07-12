#!/bin/bash

# 🚨 Sistema de Alertas - CrypGo Trading Bot
# Monitora logs e envia alertas para erros críticos

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

# Configurações de alerta
LOG_FILE="/tmp/crypgo-alerts.log"
ALERT_INTERVAL=60  # segundos entre verificações
MAX_ERRORS=5       # máximo de erros antes de alertar

# Padrões de erro para monitorar
ERROR_PATTERNS=(
    "panic"
    "fatal"
    "FATAL"
    "ERROR.*database"
    "ERROR.*connection"
    "ERROR.*binance"
    "exception"
    "timeout"
    "failed to"
    "cannot connect"
    "authentication failed"
    "invalid.*key"
)

# Função para log com timestamp
log_message() {
    local level="$1"
    local message="$2"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] [$level] $message" >> "$LOG_FILE"
    echo -e "${BLUE}[$timestamp]${NC} ${message}"
}

# Função para enviar alerta (pode ser expandida para email, Slack, etc.)
send_alert() {
    local alert_type="$1"
    local message="$2"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    echo -e "${RED}🚨 ALERTA CRÍTICO 🚨${NC}"
    echo -e "${WHITE}Tipo: ${alert_type}${NC}"
    echo -e "${WHITE}Hora: ${timestamp}${NC}"
    echo -e "${WHITE}Mensagem: ${message}${NC}"
    echo "========================================="
    
    # Log do alerta
    log_message "ALERT" "$alert_type: $message"
    
    # Aqui você pode adicionar integração com:
    # - Email
    # - Slack
    # - Telegram
    # - Discord
    # - SMS
    # - Webhook personalizado
    
    # Exemplo de webhook (descomente e configure se necessário):
    # curl -X POST -H "Content-Type: application/json" \
    #      -d "{\"text\":\"🚨 CrypGo Alert: $alert_type - $message\"}" \
    #      "YOUR_WEBHOOK_URL"
}

# Função para verificar se os serviços estão rodando
check_services() {
    log_message "INFO" "Verificando status dos serviços..."
    
    local services_status=$(ssh ${VPS_USER}@${VPS_HOST} "cd ${PROJECT_PATH} && docker-compose -f docker-compose.full.yml ps --format json" 2>/dev/null || echo "[]")
    
    if [ "$services_status" = "[]" ]; then
        send_alert "SERVICE_DOWN" "Não foi possível conectar aos serviços Docker"
        return 1
    fi
    
    # Verificar se a aplicação principal está rodando
    local app_running=$(ssh ${VPS_USER}@${VPS_HOST} "cd ${PROJECT_PATH} && docker-compose -f docker-compose.full.yml ps crypgo-app --format json" 2>/dev/null | grep -c '"State":"running"' || echo "0")
    
    if [ "$app_running" = "0" ]; then
        send_alert "APP_DOWN" "Aplicação CrypGo não está rodando"
        return 1
    fi
    
    log_message "INFO" "Todos os serviços estão operacionais"
    return 0
}

# Função para verificar conectividade da API
check_api() {
    log_message "INFO" "Verificando API..."
    
    # Testar endpoint principal
    if ! curl -s --max-time 10 "http://${VPS_HOST}:8080/api/v1/trading/list" > /dev/null; then
        send_alert "API_DOWN" "API não está respondendo na porta 8080"
        return 1
    fi
    
    log_message "INFO" "API está respondendo normalmente"
    return 0
}

# Função para analisar logs em busca de erros
analyze_logs() {
    log_message "INFO" "Analisando logs para erros críticos..."
    
    local error_count=0
    local recent_logs=$(ssh ${VPS_USER}@${VPS_HOST} "cd ${PROJECT_PATH} && docker-compose -f docker-compose.full.yml logs --tail 200 --since ${ALERT_INTERVAL}s crypgo-app" 2>/dev/null || echo "")
    
    if [ -z "$recent_logs" ]; then
        send_alert "LOG_ERROR" "Não foi possível obter logs da aplicação"
        return 1
    fi
    
    # Verificar cada padrão de erro
    for pattern in "${ERROR_PATTERNS[@]}"; do
        local matches=$(echo "$recent_logs" | grep -i -c "$pattern" || echo "0")
        if [ "$matches" -gt 0 ]; then
            error_count=$((error_count + matches))
            log_message "WARN" "Encontrados $matches erros do tipo: $pattern"
            
            # Se encontrar erros críticos específicos, alertar imediatamente
            if [[ "$pattern" =~ (panic|fatal|FATAL) ]]; then
                local error_lines=$(echo "$recent_logs" | grep -i "$pattern" | tail -3)
                send_alert "CRITICAL_ERROR" "Erro crítico detectado: $pattern\n$error_lines"
            fi
        fi
    done
    
    # Alertar se muitos erros em geral
    if [ "$error_count" -gt "$MAX_ERRORS" ]; then
        send_alert "HIGH_ERROR_RATE" "Muitos erros detectados nos últimos ${ALERT_INTERVAL}s: $error_count erros"
    fi
    
    log_message "INFO" "Análise de logs concluída. Erros encontrados: $error_count"
}

# Função para verificar uso de recursos
check_resources() {
    log_message "INFO" "Verificando uso de recursos..."
    
    local stats=$(ssh ${VPS_USER}@${VPS_HOST} "docker stats --no-stream --format 'table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}' | grep crypgo-app" 2>/dev/null || echo "")
    
    if [ -z "$stats" ]; then
        send_alert "MONITORING_ERROR" "Não foi possível obter estatísticas de recursos"
        return 1
    fi
    
    # Extrair uso de CPU (remover %)
    local cpu_usage=$(echo "$stats" | awk '{print $2}' | sed 's/%//')
    
    # Alertar se CPU alta (>80%)
    if (( $(echo "$cpu_usage > 80" | bc -l) )); then
        send_alert "HIGH_CPU" "Uso alto de CPU detectado: ${cpu_usage}%"
    fi
    
    log_message "INFO" "Recursos dentro dos limites normais"
}

# Função para executar verificação completa
run_health_check() {
    log_message "INFO" "Iniciando verificação de saúde..."
    
    local checks_passed=0
    local total_checks=4
    
    # Verificar serviços
    if check_services; then
        checks_passed=$((checks_passed + 1))
    fi
    
    # Verificar API
    if check_api; then
        checks_passed=$((checks_passed + 1))
    fi
    
    # Analisar logs
    if analyze_logs; then
        checks_passed=$((checks_passed + 1))
    fi
    
    # Verificar recursos
    if check_resources; then
        checks_passed=$((checks_passed + 1))
    fi
    
    local health_score=$((checks_passed * 100 / total_checks))
    log_message "INFO" "Health check concluído. Score: ${health_score}% (${checks_passed}/${total_checks})"
    
    if [ "$health_score" -lt 75 ]; then
        send_alert "HEALTH_WARNING" "Sistema com problemas. Health score: ${health_score}%"
    fi
}

# Função para mostrar menu
show_menu() {
    echo -e "${RED}🚨 CrypGo Trading Bot - Sistema de Alertas${NC}"
    echo "============================================="
    echo ""
    echo -e "${CYAN}Escolha uma opção:${NC}"
    echo "1. 🔄 Executar verificação única"
    echo "2. 👁️  Monitoramento contínuo (cada ${ALERT_INTERVAL}s)"
    echo "3. 📊 Verificar status dos serviços"
    echo "4. 🌐 Testar conectividade da API"
    echo "5. 🔍 Analisar logs para erros"
    echo "6. 💾 Verificar uso de recursos"
    echo "7. 📋 Ver log de alertas"
    echo "8. ⚙️  Configurar alertas"
    echo "9. 🚪 Sair"
    echo ""
    echo -n -e "${YELLOW}Digite sua escolha (1-9): ${NC}"
}

# Função para monitoramento contínuo
continuous_monitoring() {
    echo -e "${GREEN}[INFO]${NC} Iniciando monitoramento contínuo..."
    echo -e "${BLUE}[INFO]${NC} Verificando a cada ${ALERT_INTERVAL} segundos"
    echo -e "${BLUE}[INFO]${NC} Pressione Ctrl+C para parar"
    echo ""
    
    # Criar arquivo de log se não existir
    touch "$LOG_FILE"
    
    local iteration=1
    
    while true; do
        echo -e "${CYAN}[$(date)] Iteração #${iteration}${NC}"
        run_health_check
        echo ""
        
        iteration=$((iteration + 1))
        sleep "$ALERT_INTERVAL"
    done
}

# Função para ver log de alertas
view_alert_log() {
    if [ -f "$LOG_FILE" ]; then
        echo -e "${CYAN}📋 Últimos 50 alertas:${NC}"
        echo "======================"
        tail -50 "$LOG_FILE"
    else
        echo -e "${YELLOW}[INFO]${NC} Nenhum log de alerta encontrado"
    fi
    echo ""
    echo -e "${YELLOW}[INFO]${NC} Pressione Enter para continuar..."
    read
}

# Função para configurar alertas
configure_alerts() {
    echo -e "${CYAN}⚙️ Configuração de Alertas${NC}"
    echo "=========================="
    echo ""
    echo "Configurações atuais:"
    echo "• Intervalo: ${ALERT_INTERVAL}s"
    echo "• Máximo de erros: ${MAX_ERRORS}"
    echo "• Arquivo de log: ${LOG_FILE}"
    echo ""
    echo -n -e "${YELLOW}Deseja alterar o intervalo? (atual: ${ALERT_INTERVAL}s): ${NC}"
    read new_interval
    if [[ "$new_interval" =~ ^[0-9]+$ ]] && [ "$new_interval" -gt 0 ]; then
        ALERT_INTERVAL="$new_interval"
        echo -e "${GREEN}[INFO]${NC} Intervalo atualizado para ${ALERT_INTERVAL}s"
    fi
    
    echo -n -e "${YELLOW}Deseja alterar o máximo de erros? (atual: ${MAX_ERRORS}): ${NC}"
    read new_max_errors
    if [[ "$new_max_errors" =~ ^[0-9]+$ ]] && [ "$new_max_errors" -gt 0 ]; then
        MAX_ERRORS="$new_max_errors"
        echo -e "${GREEN}[INFO]${NC} Máximo de erros atualizado para ${MAX_ERRORS}"
    fi
    echo ""
}

# Verificar dependências
if ! command -v bc &> /dev/null; then
    echo -e "${YELLOW}[WARN]${NC} 'bc' não encontrado. Instalando..."
    # Tentar instalar bc dependendo do sistema
    if [[ "$OSTYPE" == "darwin"* ]]; then
        brew install bc 2>/dev/null || echo -e "${RED}[ERROR]${NC} Instale bc manualmente: brew install bc"
    elif command -v apt-get &> /dev/null; then
        sudo apt-get install -y bc
    elif command -v yum &> /dev/null; then
        sudo yum install -y bc
    fi
fi

# Loop do menu principal
while true; do
    show_menu
    read choice
    echo ""
    
    case $choice in
        1)
            run_health_check
            echo ""
            echo -e "${YELLOW}[INFO]${NC} Pressione Enter para continuar..."
            read
            ;;
        2)
            continuous_monitoring
            echo ""
            ;;
        3)
            check_services
            echo ""
            echo -e "${YELLOW}[INFO]${NC} Pressione Enter para continuar..."
            read
            ;;
        4)
            check_api
            echo ""
            echo -e "${YELLOW}[INFO]${NC} Pressione Enter para continuar..."
            read
            ;;
        5)
            analyze_logs
            echo ""
            echo -e "${YELLOW}[INFO]${NC} Pressione Enter para continuar..."
            read
            ;;
        6)
            check_resources
            echo ""
            echo -e "${YELLOW}[INFO]${NC} Pressione Enter para continuar..."
            read
            ;;
        7)
            view_alert_log
            ;;
        8)
            configure_alerts
            ;;
        9)
            echo -e "${GREEN}[INFO]${NC} Saindo do sistema de alertas..."
            exit 0
            ;;
        *)
            echo -e "${RED}[ERROR]${NC} Opção inválida. Escolha entre 1-9."
            echo ""
            ;;
    esac
done