package notifications

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/projectsveltos/libsveltos/api/v1beta1"
)

type smtpInfo struct {
	recipients string
	bcc        string
	identity   string
	fromEmail  string
	password   string
	host       string
	port       string
}

type SmtpMailer struct {
	smtpInfo *smtpInfo
}

func NewMailer(ctx context.Context, c client.Client, n *v1beta1.Notification) (*SmtpMailer, error) {
	s, err := getSmtpInfo(ctx, c, n)
	if err != nil {
		return nil, fmt.Errorf("could not create mailer, %w", err)
	}
	return &SmtpMailer{smtpInfo: s}, nil
}

func (m *SmtpMailer) SendMail(subject, message string, sendAsHtml bool) error {
	server := fmt.Sprintf("%s:%s", m.smtpInfo.host, m.smtpInfo.port)
	to := strings.Split(m.smtpInfo.recipients, ",")
	msg := []byte(buildMessageHeaders(sendAsHtml, m.smtpInfo.identity, m.smtpInfo.fromEmail, m.smtpInfo.recipients, subject) + message)
	if m.smtpInfo.password == "" {
		return sendWithoutAuth(server, m.smtpInfo.fromEmail, to, msg)
	}
	return sendWithAuth(server, m.smtpInfo.identity, m.smtpInfo.fromEmail, m.smtpInfo.password, to, msg)
}

func buildMessageHeaders(isHtml bool, identity, from, to, subject string) string {
	if identity != "" {
		from = fmt.Sprintf("%s <%s>", identity, from)
	}
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n", from, to, subject)
	if isHtml {
		msg += "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n"
	}
	return msg
}

func sendWithAuth(server, identity, from, password string, to []string, msg []byte) error {
	auth := smtp.PlainAuth(identity, from, password, server)
	return smtp.SendMail(server, auth, from, to, msg)
}

func sendWithoutAuth(server, from string, to []string, msg []byte) error {
	c, err := smtp.Dial(server)
	if err != nil {
		return err
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err := c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, string(msg))
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}

func getSmtpInfo(ctx context.Context, c client.Client, notification *v1beta1.Notification) (*smtpInfo, error) {
	secret, err := getSecret(ctx, c, notification)
	if err != nil {
		return nil, err
	}

	to, ok := secret.Data[v1beta1.SmtpRecipients]
	if !ok {
		return nil, fmt.Errorf("secret does not contain email recipients")
	}

	bcc, ok := secret.Data[v1beta1.SmtpBcc]
	if !ok {
		bcc = []byte{}
	}

	identity, ok := secret.Data[v1beta1.SmtpIdentity]
	if !ok {
		identity = []byte{}
	}

	from, ok := secret.Data[v1beta1.SmtpSender]
	if !ok {
		return nil, fmt.Errorf("secret does not contain email sender")
	}

	// Password is optional in environments that use e.g. IAM roles
	pass, ok := secret.Data[v1beta1.SmtpPassword]
	if !ok {
		pass = []byte{}
	}

	host, ok := secret.Data[v1beta1.SmtpHost]
	if !ok {
		return nil, fmt.Errorf("secret does not contain email host")
	}

	port, ok := secret.Data[v1beta1.SmtpPort]
	if !ok {
		port = []byte("587")
	}

	return &smtpInfo{
		recipients: string(to),
		bcc:        string(bcc),
		identity:   string(identity),
		fromEmail:  string(from),
		password:   string(pass),
		host:       string(host),
		port:       string(port),
	}, nil
}
