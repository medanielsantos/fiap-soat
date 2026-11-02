package email

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/platform/config"
)

type SMTPMailer struct {
	host, port, username, password, from string
	// rootCAs overrides the trusted certificate authorities; nil uses the system pool.
	rootCAs *x509.CertPool
}

func NewSMTPMailer(cfg config.Config) *SMTPMailer {
	return &SMTPMailer{host: cfg.SMTPHost, port: cfg.SMTPPort, username: cfg.SMTPUser, password: cfg.SMTPPassword, from: cfg.SMTPFrom}
}

func (m *SMTPMailer) Send(ctx context.Context, to, subject, body string) error {
	if m.host == "" || m.from == "" {
		return fmt.Errorf("SMTP_HOST and SMTP_FROM are required")
	}
	address := net.JoinHostPort(m.host, m.port)
	dialer := net.Dialer{Timeout: 10 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	defer func() { _ = connection.Close() }()
	deadline := time.Now().Add(10 * time.Second)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := connection.SetDeadline(deadline); err != nil {
		return err
	}
	client, err := smtp.NewClient(connection, m.host)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(&tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12, RootCAs: m.rootCAs}); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("SMTP server does not support STARTTLS")
	}
	if m.username != "" {
		if err = client.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
			return err
		}
	}
	if err = client.Mail(m.from); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	message := "From: " + m.from + "\r\nTo: " + to + "\r\nSubject: " + mime.QEncoding.Encode("UTF-8", subject) + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + strings.ReplaceAll(body, "\n", "\r\n")
	if _, err = writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
