#!/bin/bash

# 💾 Script de Backup do Banco de Dados CrypGo Trading Bot
# Executa backup completo do PostgreSQL com rotação de arquivos

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configurações
BACKUP_DIR="/opt/crypgo-machine/backups"
RETENTION_DAYS=7  # Manter backups por 7 dias
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Carregar configurações do ambiente
if [ -f ".env.production" ]; then
    export $(grep -v '^#' .env.production | xargs)
elif [ -f ".env" ]; then
    export $(grep -v '^#' .env | xargs)
else
    echo -e "${RED}[ERROR]${NC} Arquivo .env não encontrado!"
    exit 1
fi

# Configurações padrão
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-crypgo}
DB_PASSWORD=${DB_PASSWORD:-crypgo}
DB_NAME=${DB_NAME:-crypgo_machine}

echo -e "${BLUE}[INFO]${NC} 💾 Iniciando backup do banco de dados..."
echo "  Database: $DB_NAME"
echo "  Host: $DB_HOST:$DB_PORT"
echo "  Timestamp: $TIMESTAMP"
echo ""

# Criar diretório de backup se não existir
mkdir -p "$BACKUP_DIR"

# Nome do arquivo de backup
BACKUP_FILE="$BACKUP_DIR/crypgo_machine_backup_$TIMESTAMP.sql"
BACKUP_FILE_COMPRESSED="$BACKUP_FILE.gz"

# Executar backup
echo -e "${YELLOW}[BACKUP]${NC} Criando backup..."
PGPASSWORD="$DB_PASSWORD" pg_dump \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --verbose \
    --clean \
    --if-exists \
    --create \
    --format=plain \
    > "$BACKUP_FILE"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}[SUCCESS]${NC} Backup criado: $BACKUP_FILE"
else
    echo -e "${RED}[ERROR]${NC} Falha ao criar backup"
    exit 1
fi

# Comprimir backup
echo -e "${YELLOW}[COMPRESS]${NC} Comprimindo backup..."
gzip "$BACKUP_FILE"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}[SUCCESS]${NC} Backup comprimido: $BACKUP_FILE_COMPRESSED"
else
    echo -e "${RED}[ERROR]${NC} Falha ao comprimir backup"
    exit 1
fi

# Verificar tamanho do backup
BACKUP_SIZE=$(du -h "$BACKUP_FILE_COMPRESSED" | cut -f1)
echo -e "${BLUE}[INFO]${NC} Tamanho do backup: $BACKUP_SIZE"

# Rotação de backups antigos
echo -e "${YELLOW}[CLEANUP]${NC} Removendo backups antigos (mais de $RETENTION_DAYS dias)..."
find "$BACKUP_DIR" -name "crypgo_machine_backup_*.sql.gz" -type f -mtime +$RETENTION_DAYS -delete

REMAINING_BACKUPS=$(find "$BACKUP_DIR" -name "crypgo_machine_backup_*.sql.gz" -type f | wc -l)
echo -e "${BLUE}[INFO]${NC} Backups restantes: $REMAINING_BACKUPS"

# Listar backups disponíveis
echo -e "${BLUE}[INFO]${NC} Backups disponíveis:"
ls -lah "$BACKUP_DIR"/crypgo_machine_backup_*.sql.gz 2>/dev/null || echo "  Nenhum backup encontrado"

echo ""
echo -e "${GREEN}[SUCCESS]${NC} ✅ Backup concluído com sucesso!"
echo ""
echo -e "${BLUE}[INFO]${NC} Para restaurar este backup, use:"
echo "  gunzip $BACKUP_FILE_COMPRESSED"
echo "  PGPASSWORD='$DB_PASSWORD' psql -h $DB_HOST -p $DB_PORT -U $DB_USER < $BACKUP_FILE"