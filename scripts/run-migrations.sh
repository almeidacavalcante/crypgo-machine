#!/bin/bash

# 🗄️ Script para Aplicar Migrations do CrypGo Trading Bot
# Executa todas as migrations em ordem sequencial

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configurações do banco (carrega do .env)
if [ -f ".env.production" ]; then
    export $(grep -v '^#' .env.production | xargs)
    echo -e "${BLUE}[INFO]${NC} Carregando configurações de .env.production"
elif [ -f ".env" ]; then
    export $(grep -v '^#' .env | xargs)
    echo -e "${BLUE}[INFO]${NC} Carregando configurações de .env"
else
    echo -e "${RED}[ERROR]${NC} Arquivo .env não encontrado!"
    exit 1
fi

# Configurações padrão se não definidas
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-crypgo}
DB_PASSWORD=${DB_PASSWORD:-crypgo}
DB_NAME=${DB_NAME:-crypgo_machine}

echo -e "${BLUE}[INFO]${NC} Configurações do banco:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $DB_NAME"
echo "  User: $DB_USER"
echo ""

# Definir diretório das migrations
MIGRATIONS_DIR="src/infra/database/migrations"

if [ ! -d "$MIGRATIONS_DIR" ]; then
    echo -e "${RED}[ERROR]${NC} Diretório de migrations não encontrado: $MIGRATIONS_DIR"
    exit 1
fi

# Lista de migrations em ordem
migrations=(
    "001_create_trade_bots_table.sql"
    "002_add_strategy_params_column.sql"
    "003_create_trading_decision_logs_table.sql"
    "004_add_interval_seconds_column.sql"
    "005_add_current_possible_profit_column.sql"
    "006_add_financial_parameters.sql"
)

# Função para executar uma migration
run_migration() {
    local migration_file="$1"
    local migration_path="$MIGRATIONS_DIR/$migration_file"
    
    if [ ! -f "$migration_path" ]; then
        echo -e "${RED}[ERROR]${NC} Migration não encontrada: $migration_path"
        return 1
    fi
    
    echo -e "${YELLOW}[MIGRATION]${NC} Executando: $migration_file"
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$migration_path"
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}[SUCCESS]${NC} Migration executada com sucesso: $migration_file"
    else
        echo -e "${RED}[ERROR]${NC} Falha ao executar migration: $migration_file"
        return 1
    fi
}

# Testar conexão com o banco
echo -e "${BLUE}[INFO]${NC} Testando conexão com o banco de dados..."
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT version();" > /dev/null

if [ $? -eq 0 ]; then
    echo -e "${GREEN}[SUCCESS]${NC} Conexão com banco estabelecida"
else
    echo -e "${RED}[ERROR]${NC} Não foi possível conectar ao banco de dados"
    echo "Verifique se:"
    echo "  1. PostgreSQL está rodando"
    echo "  2. As credenciais estão corretas"
    echo "  3. O banco de dados '$DB_NAME' existe"
    exit 1
fi

# Executar todas as migrations
echo -e "${BLUE}[INFO]${NC} Iniciando execução das migrations..."
echo ""

for migration in "${migrations[@]}"; do
    run_migration "$migration"
    echo ""
done

echo -e "${GREEN}[SUCCESS]${NC} 🎉 Todas as migrations foram executadas com sucesso!"
echo ""

# Verificar estrutura final das tabelas
echo -e "${BLUE}[INFO]${NC} Verificando estrutura das tabelas criadas:"
echo ""

PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
SELECT 
    table_name,
    column_name,
    data_type,
    is_nullable
FROM information_schema.columns 
WHERE table_schema = 'public' 
ORDER BY table_name, ordinal_position;
"

echo ""
echo -e "${GREEN}[SUCCESS]${NC} ✅ Database setup completo!"