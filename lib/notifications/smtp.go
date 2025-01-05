package notifications

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/projectsveltos/libsveltos/api/v1beta1"
)

type smtpInfo struct {
	to        []string
	cc        []string
	bcc       []string
	fromEmail string
	password  string
	host      string
	port      string
}

type SmtpMailer struct {
	smtpInfo *smtpInfo
}

type messageInfo struct {
	smtpInfo
	subject     string
	body        string
	attachments map[string][]byte
}

func NewMailer(ctx context.Context, c client.Client, n *v1beta1.Notification) (*SmtpMailer, error) {
	s, err := getSmtpInfo(ctx, c, n)
	if err != nil {
		return nil, fmt.Errorf("could not create mailer, %w", err)
	}
	return &SmtpMailer{smtpInfo: s}, nil
}

func (m *SmtpMailer) SendMail(subject, message string, sendAsHtml bool, compressedFile *os.File) error {
	msgInfo := messageInfo{
		smtpInfo: *m.smtpInfo,
		subject:  subject,
		body:     message,
	}

	if compressedFile != nil {
		msgInfo.attachments = map[string][]byte{}
		b, err := os.ReadFile(compressedFile.Name())
		if err != nil {
			return err
		}

		_, fileName := filepath.Split(compressedFile.Name())
		msgInfo.attachments[fileName] = b
	}

	msg := toBytes(&msgInfo, sendAsHtml)

	// Sending "Bcc" messages is accomplished by including an email address in the to
	// parameter but not including it in the msg headers.
	to := msgInfo.to
	if len(msgInfo.cc) > 0 {
		to = append(to, msgInfo.cc...)
	}
	if len(msgInfo.bcc) > 0 {
		to = append(to, msgInfo.bcc...)
	}
	if msgInfo.password == "" {
		server := fmt.Sprintf("%s:%s", msgInfo.host, msgInfo.port)
		return sendWithoutAuth(server, msgInfo.fromEmail, to, msg)
	}
	return sendWithAuth(msgInfo.host, msgInfo.port, msgInfo.fromEmail, msgInfo.password,
		to, msg)
}

func toBytes(msgInfo *messageInfo, sendAsHtml bool) []byte {
	buf := bytes.NewBuffer(nil)
	withAttachments := len(msgInfo.attachments) > 0
	fmt.Fprintf(buf, "From: %s\n", msgInfo.fromEmail)
	fmt.Fprintf(buf, "Subject: %s\n", msgInfo.subject)
	fmt.Fprintf(buf, "To: %s\n", strings.Join(msgInfo.to, ","))

	if sendAsHtml {
		fmt.Fprintf(buf, "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n")
	}

	if len(msgInfo.cc) > 0 {
		fmt.Fprintf(buf, "Cc: %s\n", strings.Join(msgInfo.cc, ","))
	}

	buf.WriteString("MIME-Version: 1.0\n")
	writer := multipart.NewWriter(buf)
	boundary := writer.Boundary()
	if withAttachments {
		fmt.Fprintf(buf, "Content-Type: multipart/mixed; boundary=%s\n", boundary)
		fmt.Fprintf(buf, "--%s\n", boundary)
	} else {
		buf.WriteString("Content-Type: text/plain; charset=utf-8\n")
	}

	buf.WriteString(msgInfo.body)
	if withAttachments {
		for k, v := range msgInfo.attachments {
			fmt.Fprintf(buf, "\n\n--%s\n", boundary)
			fmt.Fprintf(buf, "Content-Type: %s\n", http.DetectContentType(v))
			buf.WriteString("Content-Transfer-Encoding: base64\n")
			fmt.Fprintf(buf, "Content-Disposition: attachment; filename=%s\n", k)

			b := make([]byte, base64.StdEncoding.EncodedLen(len(v)))
			base64.StdEncoding.Encode(b, v)
			buf.Write(b)
			fmt.Fprintf(buf, "\n--%s", boundary)
		}

		buf.WriteString("--")
	}

	return buf.Bytes()
}

func sendWithAuth(host, port, from, password string, to []string, msg []byte) error {
	auth := smtp.PlainAuth("", from, password, host)
	server := fmt.Sprintf("%s:%s", host, port)
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

	cc := secret.Data[v1beta1.SmtpCc]

	bcc := secret.Data[v1beta1.SmtpBcc]

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

	info := &smtpInfo{
		to:        strings.Split(string(to), ","),
		fromEmail: string(from),
		password:  string(pass),
		host:      string(host),
		port:      string(port),
	}
	if cc != nil {
		info.cc = strings.Split(string(cc), ",")
	}
	if bcc != nil {
		info.bcc = strings.Split(string(bcc), ",")
	}

	return info, nil
}
