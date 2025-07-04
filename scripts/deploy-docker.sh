#!/bin/bash

# 🐳 Script de Deploy com Docker - CrypGo Trading Bot
# Deploy completo usando apenas Docker (sem precisar instalar Go)

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "🐳 CrypGo Trading Bot - Deploy com Docker"
echo "========================================="
echo ""

# Verificar se Docker está instalado
if ! command -v docker &> /dev/null; then
    echo -e "${RED}[ERROR]${NC} Docker não está instalado!"
    echo "Instale o Docker primeiro:"
    echo "curl -fsSL https://get.docker.com | sh"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}[ERROR]${NC} Docker Compose não está instalado!"
    echo "Instale o Docker Compose primeiro"
    exit 1
fi

# Verificar se estamos no diretório correto
if [ ! -f "go.mod" ] || [ ! -f "main.go" ]; then
    echo -e "${RED}[ERROR]${NC} Execute este script no diretório raiz do projeto!"
    exit 1
fi

# 1. Verificar configurações
echo -e "${BLUE}[STEP 1/6]${NC} Verificando configurações..."
if [ ! -f ".env.production" ]; then
    echo -e "${RED}[ERROR]${NC} Arquivo .env.production não encontrado!"
    echo "Configure suas credenciais em .env.production"
    exit 1
fi

# Usar .env.production
cp .env.production .env
echo -e "${GREEN}[SUCCESS]${NC} Configurações carregadas"

# 2. Parar containers antigos
echo -e "${BLUE}[STEP 2/6]${NC} Parando containers antigos..."
docker-compose -f docker-compose.full.yml down 2>/dev/null || echo "Nenhum container estava rodando"

# 3. Limpar imagens antigas (opcional)
echo -e "${BLUE}[STEP 3/6]${NC} Limpando imagens antigas..."
docker image prune -f
docker system prune -f

# 4. Construir e subir todos os serviços
echo -e "${BLUE}[STEP 4/6]${NC} Construindo e iniciando serviços..."
echo "Isso pode demorar alguns minutos na primeira vez..."

# Build da aplicação
docker-compose -f docker-compose.full.yml build crypgo-app

# Subir todos os serviços
docker-compose -f docker-compose.full.yml up -d

# 5. Aguardar serviços iniciarem
echo -e "${BLUE}[STEP 5/6]${NC} Aguardando serviços iniciarem..."
echo "Aguardando banco de dados..."
sleep 30

echo "Verificando health checks..."
for i in {1..30}; do
    if docker-compose -f docker-compose.full.yml ps | grep -q "healthy"; then
        echo -e "${GREEN}[INFO]${NC} Serviços estão saudáveis!"
        break
    fi
    echo "Aguardando... ($i/30)"
    sleep 10
done

# 6. Verificar se tudo está funcionando
echo -e "${BLUE}[STEP 6/6]${NC} Verificando aplicação..."

# Aguardar aplicação estar pronta
sleep 20

# Testar API
if curl -s http://localhost:8080/api/v1/trading/list &>/dev/null; then
    echo -e "${GREEN}[SUCCESS]${NC} ✅ API está funcionando!"
elif curl -s http://localhost/api/v1/trading/list &>/dev/null; then
    echo -e "${GREEN}[SUCCESS]${NC} ✅ API está funcionando via Nginx!"
else
    echo -e "${YELLOW}[WARNING]${NC} API ainda não está respondendo"
    echo "Verificando logs..."
    docker-compose -f docker-compose.full.yml logs crypgo-app --tail 20
fi

# Mostrar status dos containers
echo ""
echo -e "${BLUE}[STATUS]${NC} Containers rodando:"
docker-compose -f docker-compose.full.yml ps

echo ""
echo -e "${GREEN}[SUCCESS]${NC} 🎉 Deploy Docker concluído!"
echo ""
echo -e "${BLUE}[INFO]${NC} Endpoints disponíveis:"
echo "• API direta: http://$(hostname -I | awk '{print $1}'):8080"
echo "• API via Nginx: http://$(hostname -I | awk '{print $1}')"
echo "• RabbitMQ Management: http://$(hostname -I | awk '{print $1}'):15672"
echo ""
echo -e "${BLUE}[INFO]${NC} Comandos úteis:"
echo "• Ver logs: docker-compose -f docker-compose.full.yml logs -f"
echo "• Ver status: docker-compose -f docker-compose.full.yml ps"
echo "• Parar tudo: docker-compose -f docker-compose.full.yml down"
echo "• Reiniciar app: docker-compose -f docker-compose.full.yml restart crypgo-app"
echo ""
echo -e "${BLUE}[INFO]${NC} Teste a API:"
echo "curl http://localhost:8080/api/v1/trading/list"
echo "curl http://localhost/api/v1/trading/list"