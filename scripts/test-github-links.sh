#!/bin/bash

# 🔗 Script para Testar Links do GitHub
# Verifica se os arquivos estão disponíveis no repositório

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "🔗 Testando Links do GitHub..."
echo "============================="
echo ""

# Lista de arquivos para testar
files=(
    "scripts/install-vps.sh"
    "scripts/deploy.sh"
    "scripts/run-migrations.sh"
    "scripts/backup-database.sh"
    "scripts/update.sh"
    ".env.production"
    "docker-compose.production.yml"
    "DEPLOY.md"
)

BASE_URL="https://raw.githubusercontent.com/almeidacavalcante/crypgo-machine/main"

# Função para testar um arquivo
test_file() {
    local file="$1"
    local url="$BASE_URL/$file"
    
    echo -e "${BLUE}[TESTE]${NC} $file"
    
    if curl -s --head "$url" | head -n 1 | grep -q "200 OK"; then
        echo -e "${GREEN}[✅ OK]${NC} $url"
    else
        echo -e "${RED}[❌ ERRO]${NC} $url"
        echo -e "${YELLOW}[INFO]${NC} Arquivo não encontrado no GitHub"
    fi
    echo ""
}

# Testar todos os arquivos
for file in "${files[@]}"; do
    test_file "$file"
done

echo "🔗 Teste de links concluído!"
echo ""
echo -e "${YELLOW}[INFO]${NC} Se algum arquivo deu erro:"
echo "1. Verifique se o repositório é público"
echo "2. Confirme se os arquivos foram commitados:"
echo "   git add . && git commit -m 'add deploy scripts' && git push origin main"
echo "3. Verifique se a branch é 'main' (não 'master')"