package emailsender

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"github.com/google/uuid"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

// compile-time check: *SMTPSender implements Sender.
var _ Sender = (*SMTPSender)(nil)

// SMTPSender sends emails via an SMTP relay with STARTTLS.
type SMTPSender struct {
	host      string
	port      string
	user      string
	pass      string
	fromEmail string
	fromName  string
}

func NewSMTP(host, port, user, pass, fromEmail, fromName string) *SMTPSender {
	return &SMTPSender{
		host:      host,
		port:      port,
		user:      user,
		pass:      pass,
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

// Send delivers an email via SMTP.
func (s *SMTPSender) Send(ctx context.Context, email *domain.EmailPayload) (*SendResult, error) {
	addr := net.JoinHostPort(s.host, s.port)

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, &RetryableError{Err: fmt.Errorf("smtp dial: %w", err)}
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		conn.Close()
		return nil, &RetryableError{Err: fmt.Errorf("smtp new client: %w", err)}
	}

	defer client.Close()

	// STARTTLS
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: s.host}
		if err := client.StartTLS(tlsCfg); err != nil {
			return nil, &RetryableError{Err: fmt.Errorf("smtp starttls: %w", err)}
		}
	}

	// Auth
	if s.user != "" {
		auth := smtp.PlainAuth("", s.user, s.pass, s.host)
		if err := client.Auth(auth); err != nil {
			return nil, fmt.Errorf("smtp auth: %w", err)
		}
	}

	// MAIL FROM
	from := s.fromEmail
	if err := client.Mail(from); err != nil {
		return nil, &RetryableError{Err: fmt.Errorf("smtp mail from: %w", err)}
	}

	// RCPT TO — only envelope recipients (To)
	for _, addr := range email.To {
		if err := client.Rcpt(addr); err != nil {
			return nil, fmt.Errorf("smtp rcpt %s: %w", addr, err)
		}
	}

	// Build MIME message
	msg, err := buildMIMEMessage(s.fromEmail, s.fromName, email)
	if err != nil {
		return nil, fmt.Errorf("build mime: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return nil, &RetryableError{Err: fmt.Errorf("smtp data: %w", err)}
	}

	if _, err := w.Write([]byte(msg)); err != nil {
		w.Close()
		return nil, &RetryableError{Err: fmt.Errorf("smtp write: %w", err)}
	}

	if err := w.Close(); err != nil {
		return nil, &RetryableError{Err: fmt.Errorf("smtp close: %w", err)}
	}

	client.Quit()
	return &SendResult{}, nil
}

// buildMIMEMessage constructs an RFC 5322 email with multipart/alternative
// when both text and HTML bodies are present.
func buildMIMEMessage(fromEmail, fromName string, email *domain.EmailPayload) (string, error) {
	var buf bytes.Buffer

	boundary := uuid.NewString()
	hasText := email.TextBody != ""
	hasHTML := email.HTMLBody != ""

	writeHeader(&buf, "From", formatAddress(fromEmail, fromName))

	to := ""
	for i, addr := range email.To {
		if i > 0 {
			to += ", "
		}
		to += formatAddress(addr, "")
	}
	writeHeader(&buf, "To", to)
	writeHeader(&buf, "Subject", encodeHeader(email.Subject))
	writeHeader(&buf, "Date", time.Now().Format(time.RFC1123Z))
	writeHeader(&buf, "Message-ID", "<"+uuid.NewString()+"@"+extractDomain(fromEmail)+">")
	writeHeader(&buf, "MIME-Version", "1.0")

	if hasText && hasHTML {
		writeHeader(&buf, "Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, boundary))
		buf.WriteString("\r\n")
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		writeHeader(&buf, "Content-Type", "text/plain; charset=\"UTF-8\"")
		writeHeader(&buf, "Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		buf.WriteString(qpEncode(email.TextBody))
		buf.WriteString("\r\n")
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		writeHeader(&buf, "Content-Type", "text/html; charset=\"UTF-8\"")
		writeHeader(&buf, "Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		buf.WriteString(qpEncode(email.HTMLBody))
		buf.WriteString("\r\n")
		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if hasHTML {
		writeHeader(&buf, "Content-Type", "text/html; charset=\"UTF-8\"")
		writeHeader(&buf, "Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		buf.WriteString(qpEncode(email.HTMLBody))
	} else {
		writeHeader(&buf, "Content-Type", "text/plain; charset=\"UTF-8\"")
		writeHeader(&buf, "Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		buf.WriteString(qpEncode(email.TextBody))
	}

	return buf.String(), nil
}

func writeHeader(buf *bytes.Buffer, name, value string) {
	buf.WriteString(fmt.Sprintf("%s: %s\r\n", name, value))
}

func formatAddress(email, name string) string {
	if name == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", encodeHeader(name), email)
}

func extractDomain(email string) string {
	for i := len(email) - 1; i >= 0; i-- {
		if email[i] == '@' {
			return email[i+1:]
		}
	}
	return "localhost"
}

// encodeHeader applies RFC 2047 encoded-word encoding for non-ASCII characters.
func encodeHeader(s string) string {
	needsEncoding := false
	for _, r := range s {
		if r > 127 {
			needsEncoding = true
			break
		}
	}
	if !needsEncoding {
		return s
	}
	return fmt.Sprintf("=?UTF-8?Q?%s?=", qpEncode(s))
}

// qpEncode applies quoted-printable encoding, keeping ASCII alphanumerics
// and common punctuation readable.
func qpEncode(s string) string {
	var buf bytes.Buffer
	for _, r := range s {
		if r == '\n' {
			buf.WriteString("\n")
		} else if r == '\r' {
			// skip — we handle \n only
		} else if r >= 32 && r <= 126 && r != '=' {
			buf.WriteRune(r)
		} else {
			buf.WriteString(fmt.Sprintf("=%02X", r))
		}
	}
	return buf.String()
}
