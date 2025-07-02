# Database Migrations

Este diretório contém as migrações do banco de dados para manter o histórico de mudanças na estrutura do banco.

## Estrutura dos arquivos

Os arquivos de migração seguem o padrão: `{número}_{descrição}.sql`

- **001_create_trade_bots_table.sql** - Criação inicial da tabela trade_bots
- **002_add_strategy_params_column.sql** - Adição da coluna strategy_params para armazenar parâmetros das estratégias em JSON

## Como executar

### Migração completa (banco novo)
```bash
# Execute todas as migrações em ordem
psql -d your_database -f 001_create_trade_bots_table.sql
psql -d your_database -f 002_add_strategy_params_column.sql
```

### Migração incremental (banco existente)
```bash
# Execute apenas a nova migração
psql -d your_database -f 002_add_strategy_params_column.sql
```

## Notas importantes

- **Sempre execute as migrações em ordem numérica**
- **Faça backup do banco antes de executar migrações em produção**
- **Teste as migrações em ambiente de desenvolvimento primeiro**
- **Não modifique migrações já executadas em produção**

## Próximas migrações

Para adicionar uma nova migração:

1. Crie um arquivo com o próximo número sequencial
2. Adicione descrição clara no cabeçalho
3. Escreva o SQL de migração
4. Atualize este README se necessário