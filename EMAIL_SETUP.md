# 📧 Configuração de Notificações por Email

Este guia irá te ajudar a configurar notificações automáticas por email para operações de trading.

## 🎯 O que você receberá

- **📧 Email de compra**: Sempre que o bot executar uma ordem de compra
- **📧 Email de venda**: Sempre que o bot executar uma ordem de venda com detalhes do lucro/prejuízo
- **🤖 Emails de status**: Quando bots forem criados, iniciados ou pausados

## 🔧 Configuração do Gmail

### Passo 1: Ativar Autenticação de 2 Fatores
1. Acesse [myaccount.google.com](https://myaccount.google.com)
2. Vá em **Segurança** → **Verificação em duas etapas**
3. Ative a verificação em duas etapas

### Passo 2: Gerar Senha de App
1. Na página de Segurança, vá em **Senhas de app**
2. Selecione **Outro (nome personalizado)**
3. Digite "CrypGo Trading Bot"
4. Copie a senha gerada (16 caracteres)

### Passo 3: Configurar Variáveis de Ambiente

Edite o arquivo `.env` e preencha as configurações de email:

```bash
# Email notifications configuration
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=seu.email@gmail.com
SMTP_PASSWORD=senha_de_app_16_caracteres
FROM_EMAIL=seu.email@gmail.com
TARGET_EMAIL=jalmeidacn@gmail.com
```

**Exemplo:**
```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=crypgobot@gmail.com
SMTP_PASSWORD=abcd efgh ijkl mnop
FROM_EMAIL=crypgobot@gmail.com
TARGET_EMAIL=jalmeidacn@gmail.com
```

## 🚀 Testando a Configuração

### Verificação Local
```bash
# Compilar e executar
go build -o crypgo-machine
./crypgo-machine
```

Se as configurações estiverem corretas, você verá:
```
✅ Email notification consumer started successfully.
```

### Teste Completo
1. Crie um bot de teste
2. Inicie o bot
3. Aguarde uma operação de compra/venda
4. Verifique sua caixa de entrada

## 📧 Exemplos de Email

### Email de Compra
```
Assunto: 🟢 CrypGo: Compra Executada - SOLBRL

Seu trading bot realizou uma compra!
- Símbolo: SOLBRL  
- Preço: 895.50 BRL
- Quantidade: 0.025000
- Valor Total: 2000.00 BRL
- Strategy: MovingAverage
```

### Email de Venda
```
Assunto: 🔴 CrypGo: Venda Executada - SOLBRL (2.45%)

Seu trading bot realizou uma venda!
- Preço de Entrada: 895.50 BRL
- Preço de Saída: 917.50 BRL  
- Lucro: 55.00 BRL (2.45%)
- Strategy: MovingAverage
```

## 🔧 Deploy na VPS

### Configurar no Servidor
```bash
# Na VPS
ssh root@31.97.249.4
cd /opt/crypgo-machine

# Editar configurações de produção
nano .env.production
```

Adicione as mesmas configurações de email:
```bash
# Email notifications configuration  
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=seu.email@gmail.com
SMTP_PASSWORD=senha_de_app_16_caracteres
FROM_EMAIL=seu.email@gmail.com
TARGET_EMAIL=jalmeidacn@gmail.com
```

### Fazer Deploy
```bash
# Parar containers
docker-compose -f docker-compose.full.yml down

# Rebuild e restart
docker-compose -f docker-compose.full.yml build crypgo-app
docker-compose -f docker-compose.full.yml up -d

# Verificar logs
docker-compose -f docker-compose.full.yml logs crypgo-app -f
```

## 🚨 Solução de Problemas

### Emails não estão sendo enviados
1. **Verifique as credenciais**: 
   - Username deve ser o email completo
   - Password deve ser a senha de app (16 caracteres)

2. **Verifique os logs**:
   ```bash
   docker-compose -f docker-compose.full.yml logs crypgo-app | grep -i email
   ```

3. **Teste manual**:
   - Se as credenciais estiverem vazias, o sistema simulará o envio
   - Procure por mensagens como "📧 Simulando envio de email"

### Emails vão para SPAM
- Adicione o endereço do bot aos seus contatos
- Marque um email como "Não é spam"
- Configure regras de filtro no Gmail

### Taxa de envio limitada
- Gmail permite ~100 emails/dia para contas gratuitas
- Para volume maior, considere usar um serviço de email dedicado

## 📊 Monitoramento

### Verificar se emails estão funcionando
```bash
# Ver logs de email
docker-compose -f docker-compose.full.yml logs crypgo-app | grep "📧\|Email\|SMTP"

# Forçar uma operação (teste)
curl -X POST http://localhost:8080/api/v1/trading/start \
  -H "Content-Type: application/json" \
  -d '{"bot_id": "your-bot-id"}'
```

### Logs importantes
- `✅ Email sent successfully` - Email enviado com sucesso
- `📧 Simulando envio de email` - Modo simulação (credenciais não configuradas)
- `❌ Error sending email` - Erro no envio

## 🔐 Segurança

### Boas Práticas
- ✅ Use senha de app específica para o bot
- ✅ Não compartilhe as credenciais SMTP
- ✅ Monitore os logs regularmente
- ✅ Revogue senhas de app não utilizadas

### Desabilitar Emails
Para desabilitar temporariamente:
```bash
# Deixe as variáveis vazias no .env
SMTP_USERNAME=
SMTP_PASSWORD=
```

O sistema continuará funcionando normalmente, mas apenas simulará o envio.

---

✅ **Configuração completa!** Agora você receberá notificações de todas as operações de trading em **jalmeidacn@gmail.com**