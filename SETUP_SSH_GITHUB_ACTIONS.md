# Configuração SSH para GitHub Actions

## 1. Chaves SSH Geradas

### Chave Pública (para adicionar na VPS):
```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOJ1qh5BqTdlF9QxR9Jdl+lXxHdWnBiXKta1p+OPeIfO github-actions@crypgo-machine
```

### Localização das Chaves:
- **Chave Privada**: `~/.ssh/github-actions-crypgo`
- **Chave Pública**: `~/.ssh/github-actions-crypgo.pub`

## 2. Configuração na VPS (31.97.249.4)

### Passo 1: Conectar na VPS
```bash
ssh root@31.97.249.4
```

### Passo 2: Adicionar chave pública ao authorized_keys
```bash
# Criar diretório .ssh se não existir
mkdir -p ~/.ssh

# Adicionar a chave pública
echo "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOJ1qh5BqTdlF9QxR9Jdl+lXxHdWnBiXKta1p+OPeIfO github-actions@crypgo-machine" >> ~/.ssh/authorized_keys

# Configurar permissões corretas
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys
```

### Passo 3: Testar conexão SSH
```bash
# Do seu computador local, testar conexão:
ssh -i ~/.ssh/github-actions-crypgo root@31.97.249.4
```

## 3. Configuração dos GitHub Secrets

Acesse: `https://github.com/SEU_USUARIO/crypgo-machine/settings/secrets/actions`

### Secrets necessários:

#### SSH_PRIVATE_KEY
```bash
# Copiar conteúdo da chave privada:
cat ~/.ssh/github-actions-crypgo
```
**⚠️ IMPORTANTE**: Copie TODO o conteúdo, incluindo as linhas:
- `-----BEGIN OPENSSH PRIVATE KEY-----`
- `-----END OPENSSH PRIVATE KEY-----`

#### SSH_HOST
```
31.97.249.4
```

#### SSH_USER
```
root
```

#### SSH_PORT
```
22
```

## 4. Configuração Adicional (Opcional)

### Para maior segurança, restringir comandos SSH na VPS:
```bash
# Editar authorized_keys para restringir comandos
nano ~/.ssh/authorized_keys

# Adicionar antes da chave:
command="cd /opt/crypgo-machine && docker-compose -f docker-compose.full.yml pull && docker-compose -f docker-compose.full.yml up -d" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5...
```

## 5. Teste de Validação

### Comando de teste local:
```bash
ssh -i ~/.ssh/github-actions-crypgo root@31.97.249.4 "echo 'GitHub Actions SSH funcionando!'"
```

### Se tudo estiver correto, você deve ver:
```
GitHub Actions SSH funcionando!
```

## 6. Segurança

- ✅ Chave dedicada apenas para CI/CD
- ✅ Sem passphrase (necessário para automação)
- ✅ Acesso restrito ao usuário root na VPS
- ✅ Fingerprint documentado para auditoria
- ⚠️ **NUNCA** commitar a chave privada no repositório
- ⚠️ **SEMPRE** usar GitHub Secrets para armazenar credenciais

## 7. Rotação de Chaves (Recomendado a cada 90 dias)

1. Gerar novo par de chaves
2. Adicionar nova chave pública na VPS
3. Atualizar GitHub Secret `SSH_PRIVATE_KEY`
4. Remover chave antiga da VPS
5. Deletar arquivos de chave antiga do computador local