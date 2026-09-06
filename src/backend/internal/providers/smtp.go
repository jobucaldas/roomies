package providers

import (
	"context"
	"fmt"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/roomies/backend/internal/config"
)

// SMTPInvitationProvider delivers invitation notifications through an SMTP server.
type SMTPInvitationProvider struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func NewSMTPInvitationProvider(cfg *config.Config) (*SMTPInvitationProvider, error) {
	if cfg == nil || strings.TrimSpace(cfg.SMTPHost) == "" {
		return nil, fmt.Errorf("SMTP_HOST is required")
	}
	from, err := mail.ParseAddress(cfg.SMTPFrom)
	if err != nil || from.Address == "" {
		return nil, fmt.Errorf("SMTP_FROM must be a valid email address")
	}
	if (cfg.SMTPUsername == "") != (cfg.SMTPPassword == "") {
		return nil, fmt.Errorf("SMTP_USERNAME and SMTP_PASSWORD must be provided together")
	}
	if cfg.SMTPPort <= 0 || cfg.SMTPPort > 65535 {
		return nil, fmt.Errorf("SMTP_PORT must be between 1 and 65535")
	}
	return &SMTPInvitationProvider{
		host: cfg.SMTPHost, port: cfg.SMTPPort, username: cfg.SMTPUsername,
		password: cfg.SMTPPassword, from: cfg.SMTPFrom,
	}, nil
}

// NewInvitationProvider selects fake delivery only for development/test or when
// SMTP is intentionally absent. The fake provider records work for inspection;
// it never represents a claim that an email reached its recipient.
func NewInvitationProvider(cfg *config.Config) (InvitationProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	if cfg.Environment == "development" || cfg.Environment == "test" || strings.TrimSpace(cfg.SMTPHost) == "" {
		return &FakeInvitationProvider{}, nil
	}
	return NewSMTPInvitationProvider(cfg)
}

func (p *SMTPInvitationProvider) DispatchInvitation(ctx context.Context, notification InvitationNotification) error {
	recipientText := strings.TrimSpace(notification.Email)
	if recipientText == "" {
		return fmt.Errorf("invitation recipient is empty")
	}
	recipient, err := mail.ParseAddress(recipientText)
	if err != nil || recipient.Address != recipientText {
		return fmt.Errorf("invalid invitation recipient")
	}
	from, err := mail.ParseAddress(p.from)
	if err != nil {
		return fmt.Errorf("invalid SMTP sender: %w", err)
	}
	message := strings.Join([]string{
		"From: " + p.from,
		"To: " + recipient.Address,
		"Subject: Roomies invitation",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		fmt.Sprintf("You have a Roomies invitation for house %s as %s. Invitation ID: %s.", notification.HouseID, notification.Role, notification.InvitationID),
		"",
	}, "\r\n")
	var auth smtp.Auth
	if p.username != "" {
		auth = smtp.PlainAuth("", p.username, p.password, p.host)
	}
	return smtp.SendMail(p.host+":"+strconv.Itoa(p.port), auth, from.Address, []string{recipient.Address}, []byte(message))
}
