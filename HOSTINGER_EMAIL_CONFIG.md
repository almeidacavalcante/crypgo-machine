# 📧 Configuração Email Hostinger - Opções de Porta

## 🔧 Configurações SMTP Testadas

### Configuração Principal (Recomendada)
```bash
SMTP_HOST=smtp.hostinger.com
SMTP_PORT=587
SMTP_USERNAME=seu-email@seudominio.com
SMTP_PASSWORD=sua_senha_do_email
FROM_EMAIL=seu-email@seudominio.com
TARGET_EMAIL=jalmeidacn@gmail.com
```

### Configurações Alternativas

#### Se porta 587 não funcionar, teste:
```bash
# Opção 1: SSL
SMTP_HOST=smtp.hostinger.com
SMTP_PORT=465

# Opção 2: Servidor alternativo
SMTP_HOST=mail.seudominio.com
SMTP_PORT=587

# Opção 3: Sem criptografia (não recomendado)
SMTP_HOST=smtp.hostinger.com
SMTP_PORT=25
```

## 📋 Checklist de Configuração

### ✅ Antes de Configurar
1. **Conta de email criada** no painel Hostinger
2. **Senha definida** para a conta
3. **Domínio ativo** e configurado

### ✅ Informações Necessárias
- Email completo: `exemplo@seudominio.com`
- Senha da conta de email
- Domínio ativo (não pode ser @gmail.com, @yahoo.com, etc)

### ✅ Teste da Configuração
```bash
# Compilar e testar
go build -o crypgo-machine
./crypgo-machine

# Verificar logs
# Deve aparecer: "✅ Email notification consumer started successfully."
```

## 🚨 Problemas Comuns

### "Authentication failed"
- ❌ **Problema**: Username/password incorretos
- ✅ **Solução**: Verifique credenciais no hPanel

### "Connection refused"
- ❌ **Problema**: Porta bloqueada ou servidor incorreto  
- ✅ **Solução**: Teste porta 465 ou mail.seudominio.com

### "Domain not found"
- ❌ **Problema**: Domínio não configurado
- ✅ **Solução**: Verifique se domínio está ativo no Hostinger

## 📞 Onde Encontrar Ajuda

### No Painel Hostinger
1. **hPanel** → **Emails** → **Configurações**
2. **Suporte** → **Chat ao vivo**  
3. **Tutoriais** → **Configuração de Email**

### Teste Manual
```bash
# Se quiser testar SMTP manualmente
telnet smtp.hostinger.com 587
# ou
telnet smtp.hostinger.com 465
```

---

💡 **Dica**: O Hostinger geralmente fornece configurações específicas para cada domínio no painel de controle!