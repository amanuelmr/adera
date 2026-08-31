package auth

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SMTPOptions configures EmailProvider. Values come from config; the booleans
// are set from SMTP_TLS ("starttls" is the default and leaves both false).
type SMTPOptions struct {
	Host     string
	Port     int
	Username string
	Password string

	FromAddress string
	FromName    string

	// ImplicitTLS negotiates TLS from the first byte (port 465) instead of
	// upgrading a plaintext connection with STARTTLS (port 587).
	ImplicitTLS bool
	// AllowPlaintext permits delivery to a relay that offers no encryption.
	// Development only; config rejects it in production.
	AllowPlaintext bool

	Timeout time.Duration
}

// EmailProvider delivers one-time codes over SMTP. It handles the "email"
// channel only: phone verification reports ErrProviderUnavailable so the API
// returns 503 instead of pretending an SMS was sent.
type EmailProvider struct {
	opts SMTPOptions
}

// NewEmailProvider validates opts and returns a ready provider.
func NewEmailProvider(opts SMTPOptions) (EmailProvider, error) {
	if opts.Host == "" {
		return EmailProvider{}, errors.New("smtp host is required")
	}
	if opts.Port <= 0 || opts.Port > 65535 {
		return EmailProvider{}, fmt.Errorf("smtp port %d out of range", opts.Port)
	}
	if _, err := mail.ParseAddress(opts.FromAddress); err != nil {
		return EmailProvider{}, fmt.Errorf("smtp from address: %w", err)
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Second
	}
	return EmailProvider{opts: opts}, nil
}

// SendCode delivers code to an email destination. Errors never include the
// code itself, so a failed send cannot leak it into logs.
func (p EmailProvider) SendCode(ctx context.Context, channel, destination, purpose, code string) error {
	if channel != "email" {
		return ErrProviderUnavailable
	}
	msg, err := buildCodeMessage(p.opts.FromName, p.opts.FromAddress, destination, purpose, code)
	if err != nil {
		return err
	}
	if err := p.deliver(ctx, destination, msg); err != nil {
		return fmt.Errorf("delivering %s email: %w", purpose, err)
	}
	return nil
}

// deliver opens an SMTP session and writes one message.
func (p EmailProvider) deliver(ctx context.Context, to string, msg []byte) error {
	addr := net.JoinHostPort(p.opts.Host, strconv.Itoa(p.opts.Port))
	dialer := net.Dialer{Timeout: p.opts.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dialing smtp relay: %w", err)
	}
	// net/smtp predates context, so the deadline is what enforces the
	// caller's timeout for the rest of the conversation.
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(p.opts.Timeout)
	}
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return fmt.Errorf("setting smtp deadline: %w", err)
	}

	tlsCfg := &tls.Config{ServerName: p.opts.Host, MinVersion: tls.VersionTLS12}
	if p.opts.ImplicitTLS {
		tlsConn := tls.Client(conn, tlsCfg)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return fmt.Errorf("smtp tls handshake: %w", err)
		}
		conn = tlsConn
	}

	client, err := smtp.NewClient(conn, p.opts.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("starting smtp session: %w", err)
	}
	defer func() { _ = client.Close() }()

	if !p.opts.ImplicitTLS {
		if supported, _ := client.Extension("STARTTLS"); supported {
			if err := client.StartTLS(tlsCfg); err != nil {
				return fmt.Errorf("smtp starttls: %w", err)
			}
		} else if !p.opts.AllowPlaintext {
			return errors.New("smtp relay does not offer STARTTLS")
		}
	}

	if p.opts.Username != "" {
		// PlainAuth refuses to send credentials over an unencrypted
		// connection unless the relay is local, which is the behaviour we
		// want: a misconfigured relay fails instead of leaking the password.
		if err := client.Auth(smtp.PlainAuth("", p.opts.Username, p.opts.Password, p.opts.Host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(p.opts.FromAddress); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("closing message: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}

// codeSubjects and codeBodies hold the message text per OTP purpose. Copy is
// bilingual because a reader's language is not known at delivery time.
var codeSubjects = map[string]string{
	"email_verify":   "የአደራ ማረጋገጫ ኮድ / Adera verification code",
	"password_reset": "የይለፍ ቃል ዳግም ማስጀመሪያ ኮድ / Adera password reset code",
}

var codeBodies = map[string]string{
	"email_verify": "የአደራ ማረጋገጫ ኮድዎ: %s\n" +
		"ኮዱ በ%d ደቂቃ ውስጥ ጊዜው ያበቃል። ይህንን ካልጠየቁ ይህን መልእክት ችላ ይበሉ።\n\n" +
		"Your Adera verification code is: %s\n" +
		"It expires in %d minutes. If you did not request it, you can ignore this email.\n",
	"password_reset": "የይለፍ ቃል ዳግም ለማስጀመር ኮድዎ: %s\n" +
		"ኮዱ በ%d ደቂቃ ውስጥ ጊዜው ያበቃል። ይህንን ካልጠየቁ የይለፍ ቃልዎ አልተቀየረም።\n\n" +
		"Your Adera password reset code is: %s\n" +
		"It expires in %d minutes. If you did not request it, your password has not changed.\n",
}

// buildCodeMessage renders an RFC 5322 message with CRLF line endings.
func buildCodeMessage(fromName, fromAddr, to, purpose, code string) ([]byte, error) {
	subject, ok := codeSubjects[purpose]
	if !ok {
		return nil, fmt.Errorf("no email template for purpose %q", purpose)
	}
	recipient, err := mail.ParseAddress(to)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient address: %w", err)
	}
	// mail.ParseAddress accepts display names, which could smuggle a header
	// break into the To: line; only a bare address is ever sent.
	if recipient.Address != to || strings.ContainsAny(to, "\r\n") {
		return nil, errors.New("invalid recipient address")
	}

	minutes := int(otpTTL.Minutes())
	body := fmt.Sprintf(codeBodies[purpose], code, minutes, code, minutes)

	from := mail.Address{Name: fromName, Address: fromAddr}
	var b strings.Builder
	writeHeader(&b, "From", from.String())
	writeHeader(&b, "To", recipient.Address)
	writeHeader(&b, "Subject", foldEncodedWords(mime.BEncoding.Encode("UTF-8", subject)))
	writeHeader(&b, "Date", time.Now().UTC().Format(time.RFC1123Z))
	writeHeader(&b, "Message-ID", messageID(fromAddr))
	writeHeader(&b, "MIME-Version", "1.0")
	writeHeader(&b, "Content-Type", `text/plain; charset="UTF-8"`)
	writeHeader(&b, "Content-Transfer-Encoding", "base64")
	// Keep out-of-office replies and other automation off a no-reply mailbox.
	writeHeader(&b, "Auto-Submitted", "auto-generated")
	writeHeader(&b, "X-Auto-Response-Suppress", "All")
	b.WriteString("\r\n")
	b.WriteString(wrapBase64(body))
	return []byte(b.String()), nil
}

// foldEncodedWords puts each RFC 2047 encoded-word on its own continuation
// line. BEncoding.Encode separates them with a plain space, which can push a
// Subject header past the 78-character line limit RFC 5322 recommends.
func foldEncodedWords(v string) string {
	return strings.ReplaceAll(v, "?= =?", "?=\r\n =?")
}

func writeHeader(b *strings.Builder, name, value string) {
	b.WriteString(name)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteString("\r\n")
}

// messageID builds a globally unique ID scoped to the sender's domain.
func messageID(fromAddr string) string {
	domain := "localhost"
	if at := strings.LastIndex(fromAddr, "@"); at >= 0 && at+1 < len(fromAddr) {
		domain = fromAddr[at+1:]
	}
	return "<" + uuid.NewString() + "@" + domain + ">"
}

// wrapBase64 encodes body and folds it to the 76-character limit RFC 2045
// sets for base64 transfer encoding.
func wrapBase64(body string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	const lineLen = 76
	var b strings.Builder
	for len(encoded) > lineLen {
		b.WriteString(encoded[:lineLen])
		b.WriteString("\r\n")
		encoded = encoded[lineLen:]
	}
	b.WriteString(encoded)
	b.WriteString("\r\n")
	return b.String()
}
