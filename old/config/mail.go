package config

import (
	"strconv"
)

type MailDriver string

const (
	MailSMTP MailDriver = "smtp"
)

type MailConfig struct {
	Driver      MailDriver
	FromName    string
	FromAddress string

	// SMTP
	SMTPHost   string
	SMTPPort   int
	SMTPUser   string
	SMTPPass   string
	SMTPSecure bool // TLS (implicit) if true; otherwise STARTTLS if available
}

func LoadMail() *MailConfig {
	port, _ := strconv.Atoi(getEnv("MAIL_PORT", "1025"))
	secure := getEnv("MAIL_SECURE", "false") == "true"
	return &MailConfig{
		Driver:      MailDriver(getEnv("MAIL_DRIVER", "smtp")),
		FromName:    getEnv("MAIL_FROM_NAME", "Mithril"),
		FromAddress: getEnv("MAIL_FROM_ADDRESS", "noreply@example.com"),
		SMTPHost:    getEnv("MAIL_HOST", "localhost"),
		SMTPPort:    port,
		SMTPUser:    getEnv("MAIL_USERNAME", ""),
		SMTPPass:    getEnv("MAIL_PASSWORD", ""),
		SMTPSecure:  secure,
	}
}
