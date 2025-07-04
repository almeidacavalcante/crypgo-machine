#!/bin/bash

# 🔄 Script de Atualização do CrypGo Trading Bot
# Atualiza a aplicação com a versão mais recente do GitHub

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "🔄 CrypGo Trading Bot - Script de Atualização"
echo "=============================================="
echo ""

# Verificar se estamos no diretório correto
if [ ! -f "go.mod" ] || [ ! -f "main.go" ]; then
    echo -e "${RED}[ERROR]${NC} Execute este script no diretório raiz do projeto!"
    exit 1
fi

# Verificar se é um repositório git
if [ ! -d ".git" ]; then
    echo -e "${RED}[ERROR]${NC} Este não é um repositório git!"
    echo "Para usar este script, clone o projeto via git:"
    echo "git clone https://github.com/almeidacavalcante/crypgo-machine.git"
    exit 1
fi

# 1. Verificar status atual
echo -e "${BLUE}[STEP 1/7]${NC} Verificando status atual..."
if sudo systemctl is-active --quiet crypgo-machine; then
    echo -e "${GREEN}[INFO]${NC} Aplicação está rodando"
    APP_RUNNING=true
else
    echo -e "${YELLOW}[INFO]${NC} Aplicação está parada"
    APP_RUNNING=false
fi

# 2. Parar aplicação
if [ "$APP_RUNNING" = true ]; then
    echo -e "${BLUE}[STEP 2/7]${NC} Parando aplicação..."
    sudo systemctl stop crypgo-machine
    echo -e "${GREEN}[SUCCESS]${NC} Aplicação parada"
else
    echo -e "${BLUE}[STEP 2/7]${NC} Aplicação já estava parada"
fi

# 3. Fazer backup
echo -e "${BLUE}[STEP 3/7]${NC} Fazendo backup do banco de dados..."
if [ -f "scripts/backup-database.sh" ]; then
    ./scripts/backup-database.sh
    echo -e "${GREEN}[SUCCESS]${NC} Backup criado"
else
    echo -e "${YELLOW}[WARNING]${NC} Script de backup não encontrado, pulando..."
fi

# 4. Verificar mudanças remotas
echo -e "${BLUE}[STEP 4/7]${NC} Verificando atualizações no GitHub..."
git fetch origin

# Verificar se há mudanças
if git diff --quiet HEAD origin/main; then
    echo -e "${YELLOW}[INFO]${NC} Nenhuma atualização disponível"
    UPDATES_AVAILABLE=false
else
    echo -e "${GREEN}[INFO]${NC} Atualizações disponíveis!"
    UPDATES_AVAILABLE=true
    
    # Mostrar o que mudou
    echo -e "${BLUE}[INFO]${NC} Mudanças encontradas:"
    git log --oneline HEAD..origin/main | head -5
fi

# 5. Atualizar código
if [ "$UPDATES_AVAILABLE" = true ]; then
    echo -e "${BLUE}[STEP 5/7]${NC} Atualizando código..."
    git pull origin main
    echo -e "${GREEN}[SUCCESS]${NC} Código atualizado"
else
    echo -e "${BLUE}[STEP 5/7]${NC} Código já está atualizado"
fi

# 6. Recompilar aplicação
echo -e "${BLUE}[STEP 6/7]${NC} Recompilando aplicação..."
go mod download
go build -ldflags="-w -s" -o crypgo-machine main.go
echo -e "${GREEN}[SUCCESS]${NC} Aplicação recompilada"

# 7. Aplicar migrations (se houver)
echo -e "${BLUE}[STEP 7/7]${NC} Verificando migrations..."
if [ -f "scripts/run-migrations.sh" ]; then
    ./scripts/run-migrations.sh
    echo -e "${GREEN}[SUCCESS]${NC} Migrations aplicadas"
else
    echo -e "${YELLOW}[WARNING]${NC} Script de migrations não encontrado"
fi

# 8. Reiniciar aplicação
if [ "$APP_RUNNING" = true ]; then
    echo -e "${BLUE}[FINAL]${NC} Reiniciando aplicação..."
    sudo systemctl start crypgo-machine
    
    # Aguardar inicialização
    sleep 5
    
    # Verificar se iniciou corretamente
    if sudo systemctl is-active --quiet crypgo-machine; then
        echo -e "${GREEN}[SUCCESS]${NC} ✅ Aplicação reiniciada com sucesso!"
    else
        echo -e "${RED}[ERROR]${NC} ❌ Falha ao reiniciar aplicação"
        echo "Verificando logs:"
        sudo journalctl -u crypgo-machine --no-pager -l --since "1 minute ago"
        exit 1
    fi
else
    echo -e "${YELLOW}[INFO]${NC} Aplicação não foi reiniciada (estava parada)"
    echo "Para iniciar: sudo systemctl start crypgo-machine"
fi

echo ""
echo -e "${GREEN}[SUCCESS]${NC} 🎉 Atualização concluída!"
echo ""
echo -e "${BLUE}[INFO]${NC} Comandos úteis:"
echo "• Status: sudo systemctl status crypgo-machine"
echo "• Logs: sudo journalctl -u crypgo-machine -f"
echo "• Testar: curl http://localhost:8080/api/v1/trading/list"