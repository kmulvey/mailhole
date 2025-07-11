package mail

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/jackc/pgx/v4/pgxpool"
)

type SmtpServer struct {
	db     *pgxpool.Pool
	server *smtp.Server
}

// linkRegex is a simple regex to find links in the email body.
var linkRegex = regexp.MustCompile(`https?://[^\s"<>]+`)

// ReceivedEmail represents a parsed email message.
type ReceivedEmail struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	Links   []string `json:"links"`
}

// Backend implements the SMTP server backend.
type Backend struct {
	db *pgxpool.Pool
}

func NewServer(db *pgxpool.Pool, domain, bindAddr string) *SmtpServer {

	be := &Backend{db: db}
	s := smtp.NewServer(be)

	s.Addr = bindAddr
	s.Domain = domain
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 2 * 1024 * 1024 // 2MB
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true // Not needed since we don't declare auth mechs

	log.Printf("Starting SMTP server at :%s", bindAddr)
	go func() {
		if err := s.ListenAndServe(); err != nil {
			log.Fatalf("SMTP server failed: %v", err)
		}
	}()

	return &SmtpServer{
		db:     db,
		server: s,
	}
}

func (s *SmtpServer) Stop() error {
	return s.server.Close()
}

// NewSession passes the mail store to a new session.
func (bkd *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{db: bkd.db}, nil
}

// Session handles a single SMTP session.
type Session struct {
	db   *pgxpool.Pool
	from string
	to   []string
}

// Mail is called when a client sends a MAIL FROM command.
func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	return nil
}

// Rcpt is called when a client sends a RCPT TO command.
func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

// Data is called when a client sends a DATA command.
func (s *Session) Data(r io.Reader) error {
	// Parse the email message
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return fmt.Errorf("failed to read email message: %w", err)
	}

	// Read the email body
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return fmt.Errorf("failed to read email body: %w", err)
	}

	sql := `
		INSERT INTO emails (sender, recipient, subject, body, links)
		VALUES ($1, $2, $3, $4, $5)
	`
	commandTag, err := s.db.Exec(context.Background(), sql, s.from, s.to[0], msg.Header.Get("Subject"), strings.TrimSpace(string(body)), linkRegex.FindAllString(string(body), -1))
	if err != nil {
		return fmt.Errorf("failed to insert email into database: %w", err)
	}

	if commandTag.RowsAffected() != 1 {
		return fmt.Errorf("no row was inserted: %v", err)
	}

	return nil
}

// Reset clears the session state.
func (s *Session) Reset() {
	s.from = ""
	s.to = nil
}

func (s *Session) Logout() error {
	return nil
}
