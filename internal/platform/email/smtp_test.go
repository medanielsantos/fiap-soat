package email

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/platform/config"
)

func TestNewSMTPMailerCopiesConfiguration(t *testing.T) {
	mailer := NewSMTPMailer(config.Config{SMTPHost: "smtp.test", SMTPPort: "2525", SMTPUser: "user", SMTPPassword: "pass", SMTPFrom: "from@test"})
	want := SMTPMailer{host: "smtp.test", port: "2525", username: "user", password: "pass", from: "from@test"}
	if *mailer != want {
		t.Fatalf("mailer = %+v, want %+v", *mailer, want)
	}
}

func TestSendDeliversMessageOverSTARTTLSWithAuth(t *testing.T) {
	server := startFakeSMTP(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.mailer("user").Send(ctx, "client@test", "Redefinição de senha", "line one\nline two"); err != nil {
		t.Fatal(err)
	}

	commands, message := server.received()
	joined := strings.Join(commands, "\n")
	for _, want := range []string{"STARTTLS", "AUTH PLAIN", "MAIL FROM:<noreply@test>", "RCPT TO:<client@test>", "DATA", "QUIT"} {
		if !strings.Contains(joined, want) {
			t.Errorf("command %q not received; got:\n%s", want, joined)
		}
	}
	for _, want := range []string{
		"From: noreply@test\r\n", "To: client@test\r\n", "Subject: =?UTF-8?q?Redefini=C3=A7=C3=A3o_de_senha?=\r\n",
		"Content-Type: text/plain; charset=UTF-8\r\n", "\r\n\r\nline one\r\nline two",
	} {
		if !strings.Contains(message, want) {
			t.Errorf("message missing %q; got:\n%q", want, message)
		}
	}
}

func TestSendSkipsAuthWithoutUsername(t *testing.T) {
	server := startFakeSMTP(t, nil)
	if err := server.mailer("").Send(context.Background(), "client@test", "Hi", "body"); err != nil {
		t.Fatal(err)
	}
	commands, _ := server.received()
	for _, command := range commands {
		if strings.HasPrefix(command, "AUTH") {
			t.Fatalf("unexpected AUTH command without username: %q", command)
		}
	}
}
