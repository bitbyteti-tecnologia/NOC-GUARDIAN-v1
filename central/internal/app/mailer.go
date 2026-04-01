package app

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type smtpConfig struct {
	Host      string
	Port      int
	User      string
	Pass      string
	From      string
	UseTLS    bool // implicit TLS (SMTPS)
	StartTLS  bool
	Enabled   bool
	AppEnv    string
	PublicURL string
	ResetMode string // "auto" | "log" | "disabled"
}

func getSMTPConfig() smtpConfig {
	cfg := smtpConfig{
		Host:      strings.TrimSpace(os.Getenv("SMTP_HOST")),
		User:      strings.TrimSpace(os.Getenv("SMTP_USER")),
		Pass:      strings.TrimSpace(os.Getenv("SMTP_PASS")),
		From:      strings.TrimSpace(os.Getenv("SMTP_FROM")),
		AppEnv:    strings.TrimSpace(os.Getenv("APP_ENV")),
		PublicURL: strings.TrimSpace(os.Getenv("PUBLIC_URL")),
		ResetMode: strings.TrimSpace(os.Getenv("RESET_EMAIL_MODE")),
	}
	if cfg.ResetMode == "" {
		cfg.ResetMode = "auto"
	}

	if v := strings.TrimSpace(os.Getenv("SMTP_PORT")); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			cfg.Port = p
		}
	}
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	cfg.UseTLS = parseBoolEnv("SMTP_USE_TLS")
	cfg.StartTLS = parseBoolEnv("SMTP_STARTTLS")

	if cfg.From == "" && cfg.User != "" {
		cfg.From = cfg.User
	}

	cfg.Enabled = cfg.Host != "" && cfg.User != "" && cfg.Pass != ""
	return cfg
}

func parseBoolEnv(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func BuildResetURL(r *http.Request, rawToken string) string {
	base := strings.TrimRight(getSMTPConfig().PublicURL, "/")
	if base == "" {
		proto := r.Header.Get("X-Forwarded-Proto")
		if proto == "" {
			if r.TLS != nil {
				proto = "https"
			} else {
				proto = "http"
			}
		}
		host := r.Host
		base = proto + "://" + host
	}
	return base + "/reset-password?token=" + url.QueryEscape(rawToken)
}

func SendResetEmail(to, resetURL string) error {
	cfg := getSMTPConfig()

	if !cfg.Enabled {
		if cfg.ResetMode == "disabled" {
			return errors.New("smtp not configured")
		}
		// "auto" e "log" retornam erro para permitir fallback controlado
		return errors.New("smtp not configured")
	}

	subject := "NOC Guardian - Redefinir senha"
	body := "Olá,\n\n" +
		"Para redefinir sua senha, acesse o link abaixo:\n" +
		resetURL + "\n\n" +
		"Se você não solicitou a redefinição, ignore este email.\n"

	msg := buildMessage(cfg.From, to, subject, body)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	if cfg.UseTLS {
		return sendMailTLS(cfg, addr, msg, to)
	}

	// SMTP sem TLS implícito (pode usar STARTTLS)
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()

	if cfg.StartTLS {
		tlsCfg := &tls.Config{ServerName: cfg.Host}
		if err := c.StartTLS(tlsCfg); err != nil {
			return err
		}
	}

	if err := c.Auth(smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)); err != nil {
		return err
	}
	if err := c.Mail(cfg.From); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

func sendMailTLS(cfg smtpConfig, addr, msg, to string) error {
	tlsCfg := &tls.Config{ServerName: cfg.Host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	c, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return err
	}
	defer func() { _ = c.Quit() }()

	if err := c.Auth(smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)); err != nil {
		return err
	}
	if err := c.Mail(cfg.From); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

func buildMessage(from, to, subject, body string) string {
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(from)
	b.WriteString("\r\n")
	b.WriteString("To: ")
	b.WriteString(to)
	b.WriteString("\r\n")
	b.WriteString("Subject: ")
	b.WriteString(subject)
	b.WriteString("\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}

func logResetFallback(err error, resetURL string) {
	cfg := getSMTPConfig()
	if strings.EqualFold(cfg.AppEnv, "production") && cfg.ResetMode != "log" {
		log.Printf("[AUTH] reset email not sent: %v", err)
		return
	}
	log.Printf("[AUTH] reset email not sent (%v). DEV URL: %s", err, resetURL)
}
