# N8N Automation Platform - Setup Guide

## 📋 Resumo da Configuração

O N8N foi adicionado ao stack do CrypGo Machine para permitir automações avançadas de workflows de trading e notificações.

## 🔧 Configuração Implementada

### Docker Services
- **N8N Application**: `crypgo-n8n` na porta `5678`
- **N8N Database**: `crypgo-n8n-postgres` com PostgreSQL dedicado
- **Nginx Proxy**: Proxy na porta `8081` com segurança IP whitelisting

### 🔐 Credenciais de Acesso

**N8N Interface:**
- **URL**: http://31.97.249.4:8081/
- **Usuário**: `admin`
- **Senha**: `CrypGoN8N2024!`

**Banco de Dados N8N:**
- **Host**: `n8n-postgres`
- **Database**: `n8n`
- **Usuário**: `n8n_user`
- **Senha**: `N8NStrongPass123!`

## 🚀 Deploy/Instalação

### 1. Fazer Deploy das Configurações
```bash
# Fazer commit e push das alterações
git add .
git commit -m "feat: adicionar N8N automation platform ao stack"
git push origin main
```

### 2. Instalar no Servidor
```bash
# Conectar ao servidor
ssh root@31.97.249.4

# Navegar para o diretório do projeto
cd /opt/crypgo-machine

# Pull das últimas alterações
git pull origin main

# Fazer backup antes de atualizar
docker-compose -f docker-compose.full.yml exec postgres pg_dump -U crypgo_prod crypgo_machine_prod > backup_before_n8n.sql

# Parar serviços atuais
docker-compose -f docker-compose.full.yml down

# Subir com novos serviços
docker-compose -f docker-compose.full.yml up -d

# Verificar status dos containers
docker-compose -f docker-compose.full.yml ps
```

### 3. Verificar Instalação
```bash
# Verificar logs do N8N
docker-compose -f docker-compose.full.yml logs n8n

# Verificar logs do banco N8N
docker-compose -f docker-compose.full.yml logs n8n-postgres

# Testar acesso
curl -I http://31.97.249.4:8081/
```

## 🔌 Integrações Possíveis com CrypGo

### 1. Webhooks para Eventos de Trading
- **URL do Webhook**: `http://31.97.249.4:8081/webhook/trading-events`
- **Eventos**: BUY, SELL, HOLD decisions
- **Dados**: Bot ID, Symbol, Price, Profit/Loss

### 2. Automações Sugeridas

#### A. Notificações Avançadas
- **Telegram/Discord**: Enviar alertas de trades
- **Email Reports**: Relatórios diários/semanais
- **SMS**: Alertas críticos de perdas

#### B. Análises e Reports
- **Google Sheets**: Exportar dados de trading
- **Dashboard Updates**: Atualizar planilhas automaticamente
- **Performance Analytics**: Cálculos de métricas avançadas

#### C. Trading Automations
- **Conditional Stops**: Parar bots baseado em condições
- **Portfolio Rebalancing**: Ajustar quantidades baseado em performance
- **Risk Management**: Alertas de exposição excessiva

### 3. APIs Disponíveis para N8N

#### CrypGo Machine API
```bash
# Listar Bots
GET http://crypgo-app:8080/api/v1/trading/list

# Logs de Trading
GET http://crypgo-app:8080/api/v1/trading/logs

# Health Check
GET http://crypgo-app:8080/api/v1/health
```

#### RabbitMQ Integration
- **Host**: `rabbitmq:5672`
- **User**: `admin`
- **Exchange**: `trading_bot`
- **Queue**: `email.notification.queue`

## 🛡️ Segurança

### IP Whitelisting
O N8N está protegido pelo mesmo sistema de IP whitelisting do CrypGo:
- Acesso limitado aos IPs autorizados
- Headers de segurança configurados
- Rate limiting aplicado

### Volumes Persistentes
- **N8N Data**: `/home/node/.n8n` → `crypgo_n8n_data`
- **Workflows**: `/home/node/.n8n/workflows` → `crypgo_n8n_workflows`
- **Database**: `/var/lib/postgresql/data` → `crypgo_n8n_postgres_data`

## 🔧 Configurações Avançadas

### Environment Variables
```env
N8N_BASIC_AUTH_ACTIVE=true
N8N_BASIC_AUTH_USER=admin
N8N_BASIC_AUTH_PASSWORD=CrypGoN8N2024!
WEBHOOK_URL=http://31.97.249.4:5678/
GENERIC_TIMEZONE=America/Sao_Paulo
N8N_ENCRYPTION_KEY=CrypGoN8NEncryptionKey2024SuperStrong!
```

### Webhook Configuration
Para receber dados do CrypGo Machine:
1. Criar workflow no N8N
2. Adicionar nó "Webhook"
3. Configurar URL: `http://31.97.249.4:8081/webhook/seu-endpoint`
4. Configurar método HTTP (POST)

## 📊 Monitoramento

### Health Checks
```bash
# N8N Health
curl http://31.97.249.4:8081/healthz

# Database Health
docker-compose -f docker-compose.full.yml exec n8n-postgres pg_isready -U n8n_user -d n8n
```

### Logs
```bash
# Logs em tempo real
docker-compose -f docker-compose.full.yml logs -f n8n

# Logs com timestamp
docker-compose -f docker-compose.full.yml logs -t n8n
```

## 🚨 Troubleshooting

### Problemas Comuns

1. **N8N não inicia**
   - Verificar se banco N8N está healthy
   - Verificar variáveis de ambiente
   - Verificar logs: `docker logs crypgo-n8n`

2. **Acesso negado (403)**
   - Verificar se IP está na whitelist do nginx
   - Verificar credenciais de basic auth

3. **Webhook não funciona**
   - Verificar se porta 8081 está acessível
   - Verificar configuração do nginx proxy

### Comandos Úteis
```bash
# Restart apenas N8N
docker-compose -f docker-compose.full.yml restart n8n

# Rebuild N8N
docker-compose -f docker-compose.full.yml up -d --build n8n

# Ver configuração N8N
docker-compose -f docker-compose.full.yml exec n8n env | grep N8N
```

## 🎯 Próximos Passos

1. **Configurar primeiro workflow** no N8N
2. **Integrar webhooks** do CrypGo Machine
3. **Configurar notificações** (Telegram, Discord, Email)
4. **Criar dashboards automatizados** (Google Sheets, etc.)
5. **Implementar risk management** automático

## 📞 Suporte

- **N8N Documentation**: https://docs.n8n.io/
- **CrypGo Integration**: Via API endpoints documentados
- **Webhook Testing**: Use Postman ou curl para testar