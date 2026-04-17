package channel

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"strings"
)

// smtpClient matches the methods of *smtp.Client we use.
type smtpClient interface {
	Auth(auth smtp.Auth) error
	Mail(from string) error
	Rcpt(to string) error
	Data() (io.WriteCloser, error)
	Quit() error
	Extension(ext string) (bool, string)
	StartTLS(config *tls.Config) error
}

// EmailDriver implements Driver for SMTP email sending.
type EmailDriver struct {
	dial func(addr, host string, secure bool) (smtpClient, error)
}

// NewEmailDriver creates an EmailDriver with real network dialing.
func NewEmailDriver() *EmailDriver {
	return &EmailDriver{dial: realDialSMTP}
}

func realDialSMTP(addr, host string, secure bool) (smtpClient, error) {
	if secure {
		tlsConfig := &tls.Config{ServerName: host}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("TLS connection failed: %w", err)
		}
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return nil, err
		}
		return client, nil
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return client, nil
}

func (d *EmailDriver) Send(config map[string]any, payload Payload) error {
	host := strVal(config, "host")
	port := intVal(config, "port")
	username := strVal(config, "username")
	password := strVal(config, "password")
	from := strVal(config, "from")
	secure := boolVal(config, "secure")

	addr := fmt.Sprintf("%s:%d", host, port)

	// Build MIME message
	to := strings.Join(payload.To, ", ")
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", payload.Subject))
	msg.WriteString("MIME-Version: 1.0\r\n")

	if payload.HTML != "" {
		boundary := "==METIS_BOUNDARY=="
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary))
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
		msg.WriteString(payload.Body)
		msg.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n\r\n")
		msg.WriteString(payload.HTML)
		msg.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))
	} else {
		msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
		msg.WriteString(payload.Body)
	}

	auth := smtp.PlainAuth("", username, password, host)

	client, err := d.dial(addr, host, secure)
	if err != nil {
		return err
	}
	defer client.Quit()

	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, addr := range payload.To {
		if err := client.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg.String())); err != nil {
		return err
	}
	return w.Close()
}

func (d *EmailDriver) Test(config map[string]any) error {
	host := strVal(config, "host")
	port := intVal(config, "port")
	username := strVal(config, "username")
	password := strVal(config, "password")
	secure := boolVal(config, "secure")

	addr := fmt.Sprintf("%s:%d", host, port)
	auth := smtp.PlainAuth("", username, password, host)

	client, err := d.dial(addr, host, secure)
	if err != nil {
		return err
	}
	defer client.Quit()

	if !secure {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{ServerName: host}
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("STARTTLS failed: %w", err)
			}
		}
	}

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}
	return nil
}

func strVal(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func intVal(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func boolVal(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}
