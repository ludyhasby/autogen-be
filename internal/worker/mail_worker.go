package worker

import (
	"bytes"
	"context"
	"html/template"
	"log/slog"
	"sync"
	"time"

	modelresponse "logisfy/internal/model/response"

	"go.opentelemetry.io/otel"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
)

type MailWorker struct {
	DB           *gorm.DB
	Log          *slog.Logger
	Location     *time.Location
	EmailAddress string
	DialHost     string
	DialPassword string
	DialUser     string
	DialPort     int
	templates    sync.Map
}

func NewMailWorker(db *gorm.DB, log *slog.Logger, location *time.Location, emailAddress, dialHost, dialPassword, dialUser string, dialPort int) *MailWorker {
	return &MailWorker{
		DB:           db,
		Log:          log,
		Location:     location,
		EmailAddress: emailAddress,
		DialHost:     dialHost,
		DialPassword: dialPassword,
		DialUser:     dialUser,
		DialPort:     dialPort,
	}
}

func (worker *MailWorker) getTemplate(templatePath string) (*template.Template, error) {
	if cached, ok := worker.templates.Load(templatePath); ok {
		return cached.(*template.Template), nil
	}
	t, err := template.ParseFiles(templatePath)
	if err != nil {
		worker.Log.Error("MailWorker.getTemplate(): failed to parse template file", "path", templatePath, "error", err)
		return nil, err
	}
	worker.templates.Store(templatePath, t)
	return t, nil
}

func (worker *MailWorker) ForgotPassword(ctx context.Context, templatePath string, forgotPasswordMail modelresponse.ForgotPasswordMail) (err error) {
	tr := otel.Tracer("worker.MailWorker")
	ctx, span := tr.Start(ctx, "ForgotPassword")
	defer span.End()

	var body bytes.Buffer
	t, err := worker.getTemplate(templatePath)
	if err != nil {
		return err
	}
	err = t.Execute(&body, forgotPasswordMail)
	if err != nil {
		worker.Log.Error("MailWorker.ForgotPassword(): failed to execute template", "error", err)
		return err
	}

	// send with gomail
	m := gomail.NewMessage()
	m.SetHeader("From", worker.EmailAddress)
	m.SetHeader("To", forgotPasswordMail.Email)
	m.SetHeader("Subject", "[NO REPLY] Reset Password Akun - Autogen PLN")
	m.SetBody("text/html", body.String())

	d := gomail.NewDialer(worker.DialHost, worker.DialPort, worker.DialUser, worker.DialPassword)

	if err = d.DialAndSend(m); err != nil {
		worker.Log.Error("MailWorker.ForgotPassword(): failed to send email", "to", forgotPasswordMail.Email, "error", err)
		return err
	}
	worker.Log.Info("MailWorker.ForgotPassword(): email sent successfully", "to", forgotPasswordMail.Email)
	return nil
}
