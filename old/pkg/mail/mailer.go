package mail

import (
	"context"
)

type Message struct {
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	TextBody    string
	HTMLBody    string
	Attachments []Attachment
}

type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

type Provider interface {
	Send(ctx context.Context, msg *Message) error
}

type Mailer struct {
	provider Provider
	fromName string
	fromAddr string
}

func New(provider Provider, fromName, fromAddr string) *Mailer {
	return &Mailer{provider: provider, fromName: fromName, fromAddr: fromAddr}
}

func (m *Mailer) Send(ctx context.Context, to string, subject string, text string, html string) error {
	return m.provider.Send(ctx, &Message{To: []string{to}, Subject: subject, TextBody: text, HTMLBody: html})
}
