package mailer

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/smtp"
)

//go:embed templates/*
var templateFS embed.FS

type Mailer struct {
	dialer *smtp.Auth
	host   string
	port   string
	from   string
}

func New(host, port, username, password, from string) *Mailer {
	auth := smtp.PlainAuth("", username, password, host)
	return &Mailer{
		dialer: &auth,
		host:   host,
		port:   port,
		from:   from,
	}
}

func (m *Mailer) Send(to, templateFile string, data any) error {
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	plainBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	htmlBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	msg := "To: " + to + "\r\n" +
		"From: " + m.from + "\r\n" +
		"Subject: " + subject.String() + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/alternative; boundary=\"boundary\"\r\n" +
		"\r\n" +
		"--boundary\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
		"\r\n" +
		plainBody.String() + "\r\n" +
		"\r\n" +
		"--boundary\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
		"\r\n" +
		htmlBody.String() + "\r\n" +
		"\r\n" +
		"--boundary--"

	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	auth := *m.dialer
	// If no password is provided (e.g. dev), we might want to skip auth or use nil,
	// but standard smtp.PlainAuth requires it.
	// For local testing with Mailhog, auth can often be ignored if not enforced.
	// Adapting for flexibility:
	if m.host == "localhost" || m.host == "127.0.0.1" {
		// Assuming no auth for local dev tools like Mailhog often
		return smtp.SendMail(addr, nil, m.from, []string{to}, []byte(msg))
	}

	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}
