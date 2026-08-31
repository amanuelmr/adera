package auth

import (
	"bufio"
	"context"
	"encoding/base64"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSMTPOptions(addr string) SMTPOptions {
	host, port, _ := net.SplitHostPort(addr)
	p, _ := net.LookupPort("tcp", port)
	return SMTPOptions{
		Host:           host,
		Port:           p,
		FromAddress:    "no-reply@adera.example.com",
		FromName:       "Adera",
		AllowPlaintext: true,
		Timeout:        5 * time.Second,
	}
}

// fakeSMTP serves exactly one plaintext SMTP session and returns the message
// body it received on the returned channel.
func fakeSMTP(t *testing.T, advertiseSTARTTLS bool) (addr string, received <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	out := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		r := bufio.NewReader(conn)
		write := func(s string) { _, _ = conn.Write([]byte(s)) }

		write("220 localhost ESMTP\r\n")
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				if advertiseSTARTTLS {
					write("250-localhost\r\n250 STARTTLS\r\n")
				} else {
					write("250-localhost\r\n250 SIZE 10485760\r\n")
				}
			case strings.HasPrefix(cmd, "MAIL"), strings.HasPrefix(cmd, "RCPT"):
				write("250 OK\r\n")
			case strings.HasPrefix(cmd, "DATA"):
				write("354 End data with <CRLF>.<CRLF>\r\n")
				var body strings.Builder
				for {
					dl, err := r.ReadString('\n')
					if err != nil {
						return
					}
					if dl == ".\r\n" {
						break
					}
					body.WriteString(dl)
				}
				out <- body.String()
				write("250 OK\r\n")
			case strings.HasPrefix(cmd, "QUIT"):
				write("221 Bye\r\n")
				return
			default:
				write("250 OK\r\n")
			}
		}
	}()
	return ln.Addr().String(), out
}

func TestNewEmailProviderValidatesOptions(t *testing.T) {
	base := testSMTPOptions("127.0.0.1:2525")

	tests := []struct {
		name   string
		mutate func(*SMTPOptions)
	}{
		{"missing host", func(o *SMTPOptions) { o.Host = "" }},
		{"port out of range", func(o *SMTPOptions) { o.Port = 0 }},
		{"invalid from address", func(o *SMTPOptions) { o.FromAddress = "nonsense" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := base
			tt.mutate(&opts)
			_, err := NewEmailProvider(opts)
			assert.Error(t, err)
		})
	}

	t.Run("defaults a missing timeout", func(t *testing.T) {
		opts := base
		opts.Timeout = 0
		p, err := NewEmailProvider(opts)
		require.NoError(t, err)
		assert.Positive(t, p.opts.Timeout)
	})
}

func TestEmailProviderRejectsPhoneChannel(t *testing.T) {
	p, err := NewEmailProvider(testSMTPOptions("127.0.0.1:2525"))
	require.NoError(t, err)

	err = p.SendCode(context.Background(), "phone", "+251911234567", "phone_verify", "123456")

	// Reported unavailable rather than attempted, so the API returns 503.
	assert.ErrorIs(t, err, ErrProviderUnavailable)
}

func TestBuildCodeMessage(t *testing.T) {
	for _, purpose := range []string{"email_verify", "password_reset"} {
		t.Run(purpose, func(t *testing.T) {
			msg, err := buildCodeMessage("Adera", "no-reply@adera.example.com", "abebe@example.com", purpose, "483920")
			require.NoError(t, err)

			text := string(msg)
			headers, encoded, found := strings.Cut(text, "\r\n\r\n")
			require.True(t, found, "headers must be separated from the body by a blank line")

			assert.Contains(t, headers, `From: "Adera" <no-reply@adera.example.com>`)
			assert.Contains(t, headers, "To: abebe@example.com")
			assert.Contains(t, headers, `Content-Type: text/plain; charset="UTF-8"`)
			assert.Contains(t, headers, "Auto-Submitted: auto-generated")
			// Amharic in the subject must be RFC 2047 encoded, never raw.
			assert.Contains(t, headers, "Subject: =?UTF-8?b?")
			assert.NotContains(t, headers, "የአደራ")

			// Encoded words are folded onto continuation lines. The bound is
			// RFC 5322's hard 998-character limit: the stdlib encoder sizes
			// each encoded-word at 75 characters without accounting for the
			// "Subject: " prefix, so the 78-character soft limit is not
			// reachable without hand-rolling an encoder.
			for _, line := range strings.Split(headers, "\r\n") {
				assert.LessOrEqual(t, len(line), 998, "header lines must not exceed the RFC 5322 limit")
			}
			assert.Contains(t, headers, "?=\r\n =?", "long subjects must fold onto continuation lines")

			for _, line := range strings.Split(strings.TrimRight(encoded, "\r\n"), "\r\n") {
				assert.LessOrEqual(t, len(line), 76, "base64 lines must fold at 76 characters")
			}

			body, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(encoded, "\r\n", ""))
			require.NoError(t, err)
			assert.Contains(t, string(body), "483920")
			assert.Contains(t, string(body), "15 minutes")
			assert.Contains(t, string(body), "ኮድዎ")
		})
	}
}

func TestBuildCodeMessageRejectsBadRecipients(t *testing.T) {
	tests := []struct {
		name string
		to   string
	}{
		{"header injection via newline", "abebe@example.com\r\nBcc: attacker@evil.example"},
		{"display name wrapper", "Abebe <abebe@example.com>"},
		{"not an address", "abebe"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildCodeMessage("Adera", "no-reply@adera.example.com", tt.to, "email_verify", "483920")
			assert.Error(t, err)
		})
	}
}

func TestBuildCodeMessageUnknownPurpose(t *testing.T) {
	_, err := buildCodeMessage("Adera", "no-reply@adera.example.com", "abebe@example.com", "phone_verify", "483920")

	assert.ErrorContains(t, err, "no email template")
}

func TestEmailProviderDeliversMessage(t *testing.T) {
	addr, received := fakeSMTP(t, false)
	p, err := NewEmailProvider(testSMTPOptions(addr))
	require.NoError(t, err)

	err = p.SendCode(context.Background(), "email", "abebe@example.com", "email_verify", "483920")
	require.NoError(t, err)

	select {
	case body := <-received:
		assert.Contains(t, body, "To: abebe@example.com")
		decoded, _, _ := strings.Cut(body, "\r\n\r\n")
		assert.Contains(t, decoded, "Subject: =?UTF-8?b?")
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for the message")
	}
}

func TestEmailProviderRefusesPlaintextRelayWhenNotAllowed(t *testing.T) {
	addr, _ := fakeSMTP(t, false)
	opts := testSMTPOptions(addr)
	opts.AllowPlaintext = false
	p, err := NewEmailProvider(opts)
	require.NoError(t, err)

	err = p.SendCode(context.Background(), "email", "abebe@example.com", "email_verify", "483920")

	assert.ErrorContains(t, err, "STARTTLS")
}

func TestEmailProviderErrorsNeverLeakTheCode(t *testing.T) {
	// Nothing is listening on the reserved discard port.
	opts := testSMTPOptions("127.0.0.1:9")
	opts.Timeout = 500 * time.Millisecond
	p, err := NewEmailProvider(opts)
	require.NoError(t, err)

	err = p.SendCode(context.Background(), "email", "abebe@example.com", "email_verify", "483920")

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "483920")
}
