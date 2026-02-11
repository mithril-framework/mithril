package providers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/mithril-framework/mithril/pkg/mail"
)

type SMTPConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	SecureTLS bool // implicit TLS (465). If false, use STARTTLS when available.
	FromName  string
	FromEmail string
}

type SMTPProvider struct {
	cfg SMTPConfig
}

func NewSMTPProvider(cfg SMTPConfig) *SMTPProvider {
	return &SMTPProvider{cfg: cfg}
}

func (s *SMTPProvider) Send(ctx context.Context, msg *mail.Message) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	from := formatFrom(s.cfg.FromName, s.cfg.FromEmail)
	headers := map[string]string{
		"From":         from,
		"To":           strings.Join(msg.To, ", "),
		"Subject":      msg.Subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}
	body := msg.TextBody
	if msg.HTMLBody != "" {
		headers["Content-Type"] = "text/html; charset=UTF-8"
		body = msg.HTMLBody
	}
	var sb strings.Builder
	for k, v := range headers {
		sb.WriteString(k)
		sb.WriteString(": ")
		sb.WriteString(v)
		sb.WriteString("\r\n")
	}
	sb.WriteString("\r\n")
	sb.WriteString(body)
	raw := []byte(sb.String())

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	if s.cfg.SecureTLS {
		// Implicit TLS (port 465)
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.cfg.Host})
		if err != nil {
			return err
		}
		c, err := smtp.NewClient(conn, s.cfg.Host)
		if err != nil {
			return err
		}
		defer func() { _ = c.Quit() }()
		if s.cfg.Username != "" {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
		if err := c.Mail(s.cfg.FromEmail); err != nil {
			return err
		}
		for _, rcpt := range msg.To {
			if err := c.Rcpt(rcpt); err != nil {
				return err
			}
		}
		wc, err := c.Data()
		if err != nil {
			return err
		}
		if _, err := wc.Write(raw); err != nil {
			wc.Close()
			return err
		}
		return wc.Close()
	}

	// STARTTLS if available
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
			return err
		}
	}
	if s.cfg.Username != "" {
		if err := c.Auth(auth); err != nil {
			return err
		}
	}
	if err := c.Mail(s.cfg.FromEmail); err != nil {
		return err
	}
	for _, rcpt := range msg.To {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	wc, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write(raw); err != nil {
		wc.Close()
		return err
	}
	return wc.Close()
}

func formatFrom(name, email string) string {
	if name == "" {
		return email
	}
	// Simple format, not full RFC2047 encoding
	return fmt.Sprintf("%s <%s>", name, email)
}
