package email

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeSMTP struct {
	host, port string
	pool       *x509.CertPool
	tlsConfig  *tls.Config
	greeting   string
	noTLS      bool
	reject     map[string]string
	mu         sync.Mutex
	commands   []string
	message    string
}

func startFakeSMTP(t *testing.T, configure func(*fakeSMTP)) *fakeSMTP {
	t.Helper()
	pool, certificate := newTestCertificate(t)
	server := &fakeSMTP{pool: pool, greeting: "220 fake ESMTP", reject: map[string]string{},
		tlsConfig: &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12}}
	if configure != nil {
		configure(server)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	server.host, server.port, _ = net.SplitHostPort(listener.Addr().String())
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go server.serve(connection)
		}
	}()
	return server
}

func (s *fakeSMTP) mailer(username string) *SMTPMailer {
	return &SMTPMailer{host: s.host, port: s.port, username: username, password: "secret", from: "noreply@test", rootCAs: s.pool}
}

func (s *fakeSMTP) serve(connection net.Conn) {
	defer func() { _ = connection.Close() }()
	_ = connection.SetDeadline(time.Now().Add(5 * time.Second))
	reader, secure := bufio.NewReader(connection), false
	reply := func(line string) { _, _ = connection.Write([]byte(line + "\r\n")) }
	reply(s.greeting)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		verb := strings.ToUpper(strings.Fields(line + " x")[0])
		s.record(line)
		if rejection, ok := s.reject[verb]; ok {
			reply(rejection)
			continue
		}
		switch verb {
		case "EHLO", "HELO":
			switch {
			case secure:
				reply("250-fake\r\n250 AUTH PLAIN")
			case s.noTLS:
				reply("250 fake")
			default:
				reply("250-fake\r\n250 STARTTLS")
			}
		case "STARTTLS":
			reply("220 ready")
			tlsConnection := tls.Server(connection, s.tlsConfig)
			if tlsConnection.Handshake() != nil {
				return
			}
			connection, reader, secure = tlsConnection, bufio.NewReader(tlsConnection), true
		case "AUTH":
			reply("235 authenticated")
		case "MAIL", "RCPT":
			reply("250 ok")
		case "DATA":
			reply("354 send data")
			s.readData(reader)
			if rejection, ok := s.reject["DATA-END"]; ok {
				reply(rejection)
				continue
			}
			reply("250 queued")
		case "QUIT":
			reply("221 bye")
			return
		default:
			reply("502 not implemented")
		}
	}
}

func (s *fakeSMTP) readData(reader *bufio.Reader) {
	var data strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil || line == ".\r\n" {
			break
		}
		data.WriteString(line)
	}
	s.mu.Lock()
	s.message = data.String()
	s.mu.Unlock()
}

func (s *fakeSMTP) record(line string) {
	s.mu.Lock()
	s.commands = append(s.commands, line)
	s.mu.Unlock()
}

func (s *fakeSMTP) received() (commands []string, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.commands...), s.message
}
