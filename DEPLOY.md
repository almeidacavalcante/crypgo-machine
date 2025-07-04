# 🚀 CrypGo Trading Bot - Guia de Deploy na VPS

Este guia irá te ajudar a fazer o deploy completo do CrypGo Trading Bot na sua VPS.

## 📋 Pré-requisitos

- VPS com Ubuntu 20.04+ ou Debian 11+
- Acesso root via SSH
- Pelo menos 2GB de RAM
- 20GB de espaço em disco

## 🚀 Instalação Rápida

**Método 1: Transferir projeto do seu PC (Recomendado até subir no GitHub)**

```bash
# No seu PC - criar arquivo compactado
cd /Users/almeida/GolandProjects/
tar --exclude='.git' --exclude='*.log' --exclude='tmp' -czf crypgo-machine.tar.gz crypgo-machine/

# Transferir para VPS e instalar tudo
scp crypgo-machine.tar.gz root@31.97.249.4:/tmp/
ssh root@31.97.249.4 "cd /tmp && tar -xzf crypgo-machine.tar.gz && cd crypgo-machine && chmod +x scripts/*.sh && ./scripts/install-vps.sh && mv /tmp/crypgo-machine /opt/ && cd /opt/crypgo-machine && nano .env.production"
```

**Método 2: Via GitHub (quando estiver público)**

```bash
ssh root@31.97.249.4 "curl -fsSL https://raw.githubusercontent.com/almeidacavalcante/crypgo-machine/main/scripts/install-vps.sh | bash && cd /opt && git clone https://github.com/almeidacavalcante/crypgo-machine.git && cd crypgo-machine && chmod +x scripts/*.sh"
```

Depois execute o deploy:

```bash
ssh root@31.97.249.4
cd /opt/crypgo-machine
# Editar suas chaves da Binance
nano .env.production  
# Executar deploy
./scripts/deploy.sh
```

## 🔧 Processo de Instalação Detalhado

### 1. **Transferir o Projeto para VPS**

```bash
# No seu PC - criar arquivo compactado
cd /Users/almeida/GolandProjects/
tar --exclude='.git' --exclude='*.log' --exclude='tmp' -czf crypgo-machine.tar.gz crypgo-machine/

# Transferir para VPS
scp crypgo-machine.tar.gz root@31.97.249.4:/opt/

# Na VPS - extrair e configurar
ssh root@31.97.249.4
cd /opt
tar -xzf crypgo-machine.tar.gz
cd crypgo-machine
chmod +x scripts/*.sh
```

### 2. **Instalar Dependências na VPS**

```bash
# Executar script de instalação automática
./scripts/install-vps.sh
```

### 3. **Configurar Ambiente de Produção**

```bash
# Editar arquivo de configuração de produção
nano .env.production
```

**IMPORTANTE**: Altere os seguintes valores:

```env
# Suas chaves reais da Binance
BINANCE_API_KEY=sua_chave_api_real_aqui
BINANCE_SECRET_KEY=sua_chave_secreta_real_aqui

# Senhas fortes para produção
DB_PASSWORD=UmaSenhaFortePara0Banco123!
RABBIT_MQ_URL=amqp://admin:UmaSenhaFortePara0RabbitMQ456!@localhost:5672/
```

### 4. **Executar Deploy Automático**

```bash
# Executar script de deploy completo
./scripts/deploy.sh
```

Este script irá:
- ✅ Configurar ambiente de produção
- ✅ Subir PostgreSQL e RabbitMQ
- ✅ Aplicar todas as migrations
- ✅ Compilar a aplicação
- ✅ Configurar como serviço systemd
- ✅ Iniciar a aplicação

### 5. **Verificar Instalação**

```bash
# Verificar status da aplicação
sudo systemctl status crypgo-machine

# Testar endpoints
curl http://localhost:8080/api/v1/trading/list

# Ver logs em tempo real
sudo journalctl -u crypgo-machine -f
```

## 📊 Monitoramento

### Comandos Úteis:

```bash
# Status dos serviços
sudo systemctl status crypgo-machine
docker-compose -f docker-compose.production.yml ps

# Logs da aplicação
sudo journalctl -u crypgo-machine -f

# Logs do PostgreSQL
docker logs crypgo-postgres-prod -f

# Logs do RabbitMQ
docker logs crypgo-rabbitmq-prod -f

# Reiniciar aplicação
sudo systemctl restart crypgo-machine

# Parar aplicação
sudo systemctl stop crypgo-machine

# Iniciar aplicação
sudo systemctl start crypgo-machine
```

### Endpoints da API:

- **Lista de Bots**: `http://31.97.249.4:8080/api/v1/trading/list`
- **Criar Bot**: `http://31.97.249.4:8080/api/v1/trading/create_trading_bot`
- **Iniciar Bot**: `http://31.97.249.4:8080/api/v1/trading/start`
- **Backtest**: `http://31.97.249.4:8080/api/v1/trading/backtest`

### Interfaces Web:

- **RabbitMQ Management**: `http://31.97.249.4:15672`
  - User: `admin`
  - Password: (conforme configurado em .env.production)

### Testes de API:

```bash
# Testar se API está funcionando
curl http://31.97.249.4:8080/api/v1/trading/list

# Criar um bot de exemplo
curl -X POST http://31.97.249.4:8080/api/v1/trading/create_trading_bot \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "SOLBRL",
    "quantity": 0.02,
    "strategy": "MovingAverage",
    "params": {
      "FastWindow": 7,
      "SlowWindow": 40
    },
    "interval_seconds": 1800,
    "initial_capital": 10000.0,
    "trade_amount": 4000.0,
    "currency": "BRL",
    "trading_fees": 0.001,
    "minimum_profit_threshold": 5.0
  }'
```

## 💾 Backup e Recuperação

### Fazer Backup:

```bash
# Backup automático do banco
./scripts/backup-database.sh
```

### Restaurar Backup:

```bash
# Listar backups disponíveis
ls -la /opt/crypgo-machine/backups/

# Restaurar backup específico
gunzip /opt/crypgo-machine/backups/crypgo_machine_backup_YYYYMMDD_HHMMSS.sql.gz
PGPASSWORD='sua_senha' psql -h localhost -U crypgo_prod -d crypgo_machine_prod < backup_file.sql
```

## 📤 Subir Projeto para GitHub (Opcional)

Para usar o método de instalação via GitHub, primeiro suba o projeto:

```bash
# No seu PC local
cd /Users/almeida/GolandProjects/crypgo-machine

# Adicionar todos os arquivos (incluindo scripts)
git add .
git commit -m "feat: adicionar scripts de deploy e configurações de produção"
git push origin main

# Verificar se subiu corretamente
echo "Verifique se o arquivo existe em:"
echo "https://raw.githubusercontent.com/almeidacavalcante/crypgo-machine/main/scripts/install-vps.sh"
```

Depois disso, você pode usar o método via GitHub no README.

## 🔄 Atualizações

Para atualizar a aplicação:

```bash
# 1. Parar aplicação
sudo systemctl stop crypgo-machine

# 2. Fazer backup
./scripts/backup-database.sh

# 3. Atualizar código do GitHub
git pull origin main

# 4. Recompilar
go build -ldflags="-w -s" -o crypgo-machine main.go

# 5. Aplicar novas migrations (se houver)
./scripts/run-migrations.sh

# 6. Reiniciar aplicação
sudo systemctl start crypgo-machine
```

### Atualização Rápida (Recomendado):

```bash
# Usar o script de atualização automática
./scripts/update.sh
```

Este script irá:
- ✅ Fazer backup automático
- ✅ Verificar atualizações no GitHub
- ✅ Atualizar código automaticamente
- ✅ Recompilar a aplicação
- ✅ Aplicar novas migrations
- ✅ Reiniciar a aplicação

## 🚨 Solução de Problemas

### Aplicação não inicia:

```bash
# Verificar logs detalhados
sudo journalctl -u crypgo-machine --no-pager -l

# Verificar se o banco está rodando
docker ps | grep postgres

# Testar conexão manual
PGPASSWORD='sua_senha' psql -h localhost -U crypgo_prod -d crypgo_machine_prod -c "SELECT version();"
```

### Problemas de performance:

```bash
# Verificar recursos do sistema
htop
df -h
free -h

# Verificar logs do banco
docker logs crypgo-postgres-prod --tail 100
```

### Problemas de conectividade:

```bash
# Verificar portas abertas
netstat -tlnp | grep :8080
netstat -tlnp | grep :5432

# Verificar firewall
ufw status
```

## 📞 Suporte

Se encontrar problemas:

1. Verifique os logs primeiro
2. Consulte a documentação
3. Verifique as configurações do .env.production
4. Teste conexões manuais com banco e APIs

## 🔐 Segurança

Recomendações importantes:

- ✅ Altere todas as senhas padrão
- ✅ Configure firewall adequadamente
- ✅ Mantenha backups regulares
- ✅ Monitore logs regularmente
- ✅ Use chaves API com permissões mínimas necessárias
- ✅ Considere usar HTTPS em produção