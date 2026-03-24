// Package mailer sends transactional notification emails via SMTP.
// All sending is best-effort: errors are logged but never returned to callers
// so that a misconfigured mailer never blocks normal request handling.
package mailer

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Config holds SMTP connection and addressing parameters.
type Config struct {
	Host       string
	Port       int
	User       string
	Password   string
	From       string
	AdminEmail string
}

// Mailer sends notification emails. The zero value is safe but silently
// disabled (when Host is empty).
type Mailer struct {
	cfg Config
}

// New returns a Mailer configured with cfg.
func New(cfg Config) *Mailer {
	return &Mailer{cfg: cfg}
}

// Enabled reports whether SMTP sending is configured.
func (m *Mailer) Enabled() bool {
	return m.cfg.Host != ""
}

// NewAppNotification holds the data used to build a new-app notification email.
type NewAppNotification struct {
	AppName      string
	AppID        string
	OwnerHandle  string
	RedirectURIs []string
	VerifiedOnly bool
	Listed       bool
	CreatedAt    time.Time
	// AdminURL is the link shown in the email for the admin to act on the app.
	AdminURL string
}

// SendNewAppNotification fires an admin notification for a newly submitted app.
// It is non-blocking: it logs errors but does not return them.
func (m *Mailer) SendNewAppNotification(n NewAppNotification) {
	if !m.Enabled() || m.cfg.AdminEmail == "" {
		return
	}

	subject := fmt.Sprintf("[SCID] New app pending review: %s", n.AppName)
	body, err := renderNewAppEmail(n)
	if err != nil {
		slog.Error("mailer: render new-app email failed", "err", err)
		return
	}

	if err := m.send(m.cfg.AdminEmail, subject, body); err != nil {
		slog.Error("mailer: send new-app notification failed", "err", err)
	}
}

// AppDecisionNotification holds the data used to build an approval/rejection email.
type AppDecisionNotification struct {
	AppName        string
	AppID          string
	OwnerHandle    string
	OutcomeTitle   string
	OutcomeMessage string
	Reason         string
	ActionLabel    string
	ActionURL      string
}

// SendAppApprovedNotification informs the owner that their app was approved.
func (m *Mailer) SendAppApprovedNotification(to string, n AppDecisionNotification) {
	if !m.Enabled() || to == "" {
		return
	}
	n.OutcomeTitle = "Application approved"
	n.OutcomeMessage = "Your SCID application has been approved and is now active."
	if n.ActionLabel == "" {
		n.ActionLabel = "Open your application"
	}
	m.sendAppDecisionNotification(to, fmt.Sprintf("[SCID] Application approved: %s", n.AppName), n)
}

// SendAppRejectedNotification informs the owner that their app was rejected.
func (m *Mailer) SendAppRejectedNotification(to string, n AppDecisionNotification) {
	if !m.Enabled() || to == "" {
		return
	}
	n.OutcomeTitle = "Application rejected"
	n.OutcomeMessage = "Your SCID application was reviewed and rejected by an administrator."
	if n.ActionLabel == "" {
		n.ActionLabel = "Open your application"
	}
	m.sendAppDecisionNotification(to, fmt.Sprintf("[SCID] Application rejected: %s", n.AppName), n)
}

func (m *Mailer) sendAppDecisionNotification(to, subject string, n AppDecisionNotification) {
	body, err := renderAppDecisionEmail(n)
	if err != nil {
		slog.Error("mailer: render app decision email failed", "err", err)
		return
	}
	if err := m.send(to, subject, body); err != nil {
		slog.Error("mailer: send app decision notification failed", "err", err)
	}
}

// send delivers a single HTML email to one recipient.
func (m *Mailer) send(to, subject, htmlBody string) error {
	from := m.cfg.From
	if from == "" {
		from = m.cfg.User
	}

	header := strings.Join([]string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		`Content-Type: text/html; charset="utf-8"`,
	}, "\r\n")
	msg := []byte(header + "\r\n\r\n" + htmlBody)

	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)

	// Use STARTTLS on port 587 / plain TLS on port 465.
	if m.cfg.Port == 465 {
		return m.sendTLS(addr, from, to, msg)
	}
	return m.sendSTARTTLS(addr, from, to, msg)
}

func (m *Mailer) sendSTARTTLS(addr, from, to string, msg []byte) error {
	host, _, _ := net.SplitHostPort(addr)
	auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, host)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

func (m *Mailer) sendTLS(addr, from, to string, msg []byte) error {
	host, _, _ := net.SplitHostPort(addr)
	tlsCfg := &tls.Config{ServerName: host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Quit() //nolint:errcheck

	if m.cfg.User != "" {
		auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp MAIL: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT: %w", err)
	}
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := wc.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	return wc.Close()
}

// --- HTML template ---

var tmplFuncs = template.FuncMap{
	"formatTime": formatTime,
	"yesNo":      yesNo,
	"list":       func(args ...any) []any { return args },
}

var newAppTmpl = template.Must(template.New("new-app").Funcs(tmplFuncs).Parse(newAppTmplSrc))

var appDecisionTmpl = template.Must(template.New("app-decision").Funcs(tmplFuncs).Parse(appDecisionTmplSrc))

// row is a sub-template for label/value pairs inside the detail card.
var _ = template.Must(newAppTmpl.New("row").Parse(`
<tr>
  <td style="padding:4px 0;font-size:12px;color:#64748b;white-space:nowrap;padding-right:16px;vertical-align:top;">{{index . 0}}</td>
  <td style="padding:4px 0;font-size:13px;color:#e2e8f0;word-break:break-all;">{{index . 1}}</td>
</tr>
`))

func renderNewAppEmail(n NewAppNotification) (string, error) {
	var buf bytes.Buffer
	if err := newAppTmpl.Execute(&buf, n); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderAppDecisionEmail(n AppDecisionNotification) (string, error) {
	var buf bytes.Buffer
	if err := appDecisionTmpl.Execute(&buf, n); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format("2 Jan 2006, 15:04 UTC")
}

func yesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
