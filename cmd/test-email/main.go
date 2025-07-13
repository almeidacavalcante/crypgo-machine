package main

import (
	"crypgo-machine/src/infra/notification"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"time"
)

func main() {
	fmt.Println("🧪 Testando configuração de email...")

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Create email service
	emailService := notification.NewEmailService()

	// Get target email from env
	targetEmail := os.Getenv("TARGET_EMAIL")
	if targetEmail == "" {
		targetEmail = "jalmeidacn@gmail.com"
	}

	// Test with a simple email
	emailData := notification.EmailData{
		To:      targetEmail,
		Subject: "🧪 CrypGo: Teste de Email",
		Body: `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
		.header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
		.content { padding: 20px; }
		.success { background-color: #d4edda; padding: 15px; border-radius: 5px; margin: 15px 0; border: 1px solid #c3e6cb; }
		.footer { background-color: #f1f1f1; padding: 15px; text-align: center; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="header">
		<h1>🧪 TESTE DE EMAIL</h1>
		<p>Configuração do CrypGo Trading Bot</p>
	</div>
	
	<div class="content">
		<div class="success">
			<h3>✅ Email funcionando perfeitamente!</h3>
			<p>Suas configurações SMTP do Hostinger estão corretas.</p>
		</div>
		
		<h3>📋 Detalhes do Teste</h3>
		<ul>
			<li><strong>Servidor SMTP:</strong> ` + os.Getenv("SMTP_HOST") + `</li>
			<li><strong>Porta:</strong> ` + os.Getenv("SMTP_PORT") + `</li>
			<li><strong>De:</strong> ` + os.Getenv("FROM_EMAIL") + `</li>
			<li><strong>Para:</strong> ` + targetEmail + `</li>
			<li><strong>Data/Hora:</strong> ` + time.Now().Format("02/01/2006 15:04:05") + `</li>
		</ul>
		
		<p><strong>🎉 Próximos passos:</strong></p>
		<ol>
			<li>Agora você receberá notificações de todas as operações de trading</li>
			<li>Emails de compra (🟢) e venda (🔴) serão enviados automaticamente</li>
			<li>Faça o deploy na VPS para ativar em produção</li>
		</ol>
	</div>
	
	<div class="footer">
		<p>CrypGo Trading Bot - Teste de Configuração</p>
		<p>Este é um email de teste automático do sistema.</p>
	</div>
</body>
</html>`,
	}

	fmt.Printf("📤 Enviando email de teste para: %s\n", targetEmail)
	fmt.Printf("🔧 Servidor SMTP: %s:%s\n", os.Getenv("SMTP_HOST"), os.Getenv("SMTP_PORT"))
	fmt.Printf("📧 De: %s\n", os.Getenv("FROM_EMAIL"))
	fmt.Println()

	// Send the test email
	err := emailService.SendEmail(emailData)
	if err != nil {
		fmt.Printf("❌ Erro ao enviar email: %v\n", err)
		fmt.Println()
		fmt.Println("🔧 Verifique:")
		fmt.Println("  1. Credenciais SMTP no .env")
		fmt.Println("  2. Se a porta 465 está desbloqueada")
		fmt.Println("  3. Se o email está ativo no Hostinger")
		os.Exit(1)
	}

	fmt.Println("✅ Email de teste enviado com sucesso!")
	fmt.Println()
	fmt.Println("📬 Verifique sua caixa de entrada em: " + targetEmail)
	fmt.Println("📁 Se não chegou, verifique também a pasta SPAM")
	fmt.Println()
	fmt.Println("🚀 Configuração email está funcionando! Pode fazer o deploy na VPS.")
}