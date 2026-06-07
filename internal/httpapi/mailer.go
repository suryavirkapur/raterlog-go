package httpapi

import (
	"fmt"
	"net/mail"
	"net/smtp"
	"net/url"
	"strings"
)

func (api *API) sendInvite(to, companyName, token string) error {
	host := strings.TrimSpace(api.cfg.SMTPHost)
	port := strings.TrimSpace(api.cfg.SMTPPort)
	if host == "" || port == "" {
		return nil
	}
	inviteURL, err := url.JoinPath(api.cfg.PublicURL, "invite", token)
	if err != nil {
		return err
	}
	body := fmt.Sprintf("You have been invited to join %s on Raterlog.\n\nAccept: %s\n\nThis link expires in 7 days.\n", companyName, inviteURL)
	message := strings.Join([]string{
		"From: " + api.cfg.SMTPFrom,
		"To: " + to,
		"Subject: You've been invited to join " + companyName + " on Raterlog",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	from := api.cfg.SMTPFrom
	if parsed, err := mail.ParseAddress(api.cfg.SMTPFrom); err == nil {
		from = parsed.Address
	}
	return smtp.SendMail(host+":"+port, nil, from, []string{to}, []byte(message))
}
