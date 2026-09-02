package mail

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// Config SMTP 配置。
type Config struct {
	Host     string
	Port     string
	User     string
	Pass     string
	From     string
	FromName string
}

// Enabled 是否已配置 SMTP。
func (c Config) Enabled() bool {
	return c.Host != ""
}

// Send 发送一封 HTML 邮件。
func (c Config) Send(to, subject, htmlBody string) error {
	addr := net.JoinHostPort(c.Host, c.Port)

	var conn net.Conn
	var err error
	if c.Port == "465" {
		conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: c.Host})
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("connect smtp: %w", err)
	}

	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if c.Port != "465" {
		if err := client.StartTLS(&tls.Config{ServerName: c.Host}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if c.User != "" {
		if err := client.Auth(smtp.PlainAuth("", c.User, c.Pass, c.Host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(c.From); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	from := c.From
	if c.FromName != "" {
		from = c.FromName + " <" + c.From + ">"
	}
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, htmlBody,
	)
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// escapeHTML 简单转义（用于邮件正文中的用户名）。
func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return r.Replace(s)
}
