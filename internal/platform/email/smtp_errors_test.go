package email

import (
	"context"
	"net"
	"strings"
	"testing"
)

func TestSendRequiresHostAndFrom(t *testing.T) {
	for _, mailer := range []*SMTPMailer{{from: "from@test"}, {host: "smtp.test"}} {
		if err := mailer.Send(context.Background(), "to@test", "s", "b"); err == nil || !strings.Contains(err.Error(), "SMTP_HOST and SMTP_FROM are required") {
			t.Fatalf("err = %v, want configuration error", err)
		}
	}
}

func TestSendFailsWhenServerIsUnreachable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	_ = listener.Close()

	mailer := &SMTPMailer{host: host, port: port, from: "from@test"}
	if err := mailer.Send(context.Background(), "to@test", "s", "b"); err == nil {
		t.Fatal("expected dial error")
	}
}

func TestSendFailsWhenServerRejectsGreeting(t *testing.T) {
	server := startFakeSMTP(t, func(s *fakeSMTP) { s.greeting = "554 go away" })
	if err := server.mailer("").Send(context.Background(), "to@test", "s", "b"); err == nil || !strings.Contains(err.Error(), "go away") {
		t.Fatalf("err = %v, want greeting rejection", err)
	}
}

func TestSendRequiresSTARTTLS(t *testing.T) {
	server := startFakeSMTP(t, func(s *fakeSMTP) { s.noTLS = true })
	err := server.mailer("user").Send(context.Background(), "to@test", "s", "b")
	if err == nil || err.Error() != "SMTP server does not support STARTTLS" {
		t.Fatalf("err = %v, want STARTTLS requirement", err)
	}
	if commands, _ := server.received(); strings.Contains(strings.Join(commands, "\n"), "AUTH") {
		t.Fatal("credentials must never be sent over plaintext")
	}
}

func TestSendRejectsUntrustedCertificate(t *testing.T) {
	server := startFakeSMTP(t, nil)
	mailer := server.mailer("user")
	mailer.rootCAs = nil // system roots do not trust the test certificate

	if err := mailer.Send(context.Background(), "to@test", "s", "b"); err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Fatalf("err = %v, want certificate verification error", err)
	}
}

func TestSendPropagatesServerRejections(t *testing.T) {
	cases := []struct{ verb, reply string }{
		{"AUTH", "535 bad credentials"},
		{"MAIL", "550 sender refused"},
		{"RCPT", "550 recipient refused"},
		{"DATA", "554 data refused"},
		{"DATA-END", "554 message refused"},
		{"QUIT", "500 quit refused"},
	}
	for _, tc := range cases {
		t.Run(tc.verb, func(t *testing.T) {
			server := startFakeSMTP(t, func(s *fakeSMTP) { s.reject[tc.verb] = tc.reply })
			err := server.mailer("user").Send(context.Background(), "to@test", "s", "b")
			if err == nil || !strings.Contains(err.Error(), tc.reply[4:]) {
				t.Fatalf("err = %v, want %q", err, tc.reply)
			}
		})
	}
}
