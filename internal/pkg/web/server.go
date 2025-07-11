package web

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v4/pgxpool"
)

var queryPrefix = `SELECT id, sender, recipient, subject, body, links, received_at FROM emails `

type EmailMessage struct {
	ID         int       `json:"id"`
	Sender     string    `json:"sender"`
	Recipient  string    `json:"recipient"`
	Subject    string    `json:"subject"`
	Body       string    `json:"body"`
	Links      []string  `json:"links"`
	ReceivedAt time.Time `json:"received_at"`
}

type Server struct {
	app *fiber.App
	db  *pgxpool.Pool
}

func NewServer(db *pgxpool.Pool, bindAddr string) *Server {
	app := fiber.New()
	app.Use("/emails/:recipient/stream", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	server := &Server{
		app: app,
		db:  db,
	}

	app.Get("/emails/:recipient/messages", server.handleGetAllMessages)
	app.Get("/emails/:recipient/messages/:which", server.handleGetMessage)
	app.Get("/emails/:recipient/stream", websocket.New(server.waitForMessages))

	log.Printf("Starting web server at :%s", bindAddr)
	go app.Listen(bindAddr)

	return server
}

func (s *Server) Stop() error {
	return s.app.Shutdown()
}

func (s *Server) handleGetAllMessages(c *fiber.Ctx) error {
	sql := fmt.Sprintf(`%s WHERE recipient = $1 ORDER BY received_at DESC`, queryPrefix)
	recipient, err := url.QueryUnescape(c.Params("recipient"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid recipient format",
		})
	}
	rows, err := s.db.Query(context.Background(), sql, recipient)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal Server Error",
		})
	}
	defer rows.Close()

	var messages []EmailMessage
	for rows.Next() {
		var msg EmailMessage
		if err := rows.Scan(
			&msg.ID, &msg.Sender, &msg.Recipient, &msg.Subject, &msg.Body, &msg.Links, &msg.ReceivedAt,
		); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Internal Server Error",
			})
		}
		messages = append(messages, msg)
	}

	if len(messages) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No messages found"})
	}

	return c.JSON(messages)
}

func (s *Server) handleGetMessage(c *fiber.Ctx) error {
	recipient, err := url.QueryUnescape(c.Params("recipient"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid recipient format",
		})
	}
	which := c.Params("which")

	var sql string
	var args []any
	switch which {
	case "first":
		sql = fmt.Sprintf(`%s WHERE recipient = $1 ORDER BY received_at ASC LIMIT 1`, queryPrefix)
		args = []any{recipient}
	case "last":
		sql = fmt.Sprintf(`%s WHERE recipient = $1 ORDER BY received_at DESC LIMIT 1`, queryPrefix)
		args = []any{recipient}
	default:
		index, err := strconv.Atoi(which)
		if err != nil || index < 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid index"})
		}
		sql = fmt.Sprintf(`%s WHERE recipient = $1 ORDER BY received_at ASC OFFSET $2 LIMIT 1`, queryPrefix)
		args = []any{recipient, index}
	}

	row := s.db.QueryRow(context.Background(), sql, args...)
	var msg EmailMessage
	if err := row.Scan(
		&msg.ID, &msg.Sender, &msg.Recipient, &msg.Subject, &msg.Body, &msg.Links, &msg.ReceivedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No message found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(msg)
}

func (s *Server) waitForMessages(c *websocket.Conn) {
	recipient, err := url.QueryUnescape(c.Params("recipient"))
	if err != nil {
		c.Close()
		return
	}
	channel := channelNameForRecipient(recipient)

	// Acquire a dedicated connection from the pool
	conn, err := s.db.Acquire(context.Background())
	if err != nil {
		log.Errorf("Failed to acquire DB connection: %v", err)
		c.Close()
		return
	}
	defer conn.Release()

	// LISTEN on the channel using the acquired connection
	_, err = conn.Exec(context.Background(), "LISTEN "+pgx.Identifier{channel}.Sanitize())
	if err != nil {
		log.Errorf("LISTEN error: %v", err)
		c.Close()
		return
	}
	defer func() {
		_, err := conn.Exec(context.Background(), "UNLISTEN "+pgx.Identifier{channel}.Sanitize())
		if err != nil {
			log.Errorf("UNLISTEN error: %v", err)
		}
	}()

	for {
		notification, err := conn.Conn().WaitForNotification(context.Background())
		if err != nil {
			log.Errorf("WaitForNotification error: %v", err)
			return
		}
		// Forward notification payload to websocket client
		if err := c.WriteMessage(websocket.TextMessage, []byte(notification.Payload)); err != nil {
			log.Errorf("websocket write error: %v", err)
			return
		}
	}
}

func channelNameForRecipient(recipient string) string {
	// Simple sanitization: replace @ and . (customize as needed)
	return "mailhole_recipient_" + strings.ReplaceAll(strings.ReplaceAll(recipient, "@", "_at_"), ".", "_dot_")
}
