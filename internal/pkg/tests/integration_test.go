package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"testing"
	"time"

	_ "embed"

	"github.com/jackc/pgx/v4"
	"github.com/kmulvey/mailhole/internal/pkg/mailhole"
	"github.com/kmulvey/mailhole/internal/pkg/web"
	"github.com/stretchr/testify/assert"
)

var testEmail = `
From: verify@example.com
To: kevin@example.com
Subject: PaperlessPost Verification email

Please click here:
https://example.com
https://example.com/wedding
https://example.com/pricing
`

func TestSendingFull(t *testing.T) {
	t.Parallel()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://mailhole:mailhole@localhost:5432/mailhole?sslmode=disable"
	}
	smtpAddr := os.Getenv("SMTP_ADDR")
	if smtpAddr == "" {
		smtpAddr = "localhost:2525"
	}
	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = "localhost:8080"
	}

	var app, err = mailhole.New(context.Background(), dbURL, smtpAddr, httpAddr)
	assert.NoError(t, err, "Failed to create mailhole app")
	defer func() {
		app.Stop()
		db, err := pgx.Connect(context.Background(), dbURL)
		assert.NoError(t, err, "Failed to connect to database for cleanup")
		defer db.Close(context.Background())
		// Cleanup old emails
		_, err = db.Exec(context.Background(), `DELETE FROM emails`)
		assert.NoError(t, err, "Failed to cleanup old emails")
	}()

	// test different recipients
	for i := range 5 {
		var recipient = fmt.Sprintf("kevin%d@example.com", i)
		assert.NoError(t, sendTestEmail(smtpAddr, "verify@example.com", recipient))
		time.Sleep(100 * time.Millisecond) // Give some time for the email to be processed
		code, email, err := makeHttpReq(fmt.Sprintf("http://%s/emails/%s/messages/0", httpAddr, recipient))
		assert.NoError(t, err)
		assert.Equal(t, 200, code, "Expected HTTP 200 OK")
		testHttpResponses(t, code, email, recipient)
	}

	// test multiple messages to one recipient
	var recipient = "kevin@example.com"
	for i := range 5 {
		assert.NoError(t, sendTestEmail(smtpAddr, "verify@example.com", recipient))
		time.Sleep(100 * time.Millisecond) // Give some time for the email to be processed
		code, email, err := makeHttpReq(fmt.Sprintf("http://%s/emails/%s/messages/%d", httpAddr, recipient, i))
		assert.NoError(t, err)
		assert.Equal(t, 200, code, "Expected HTTP 200 OK")
		testHttpResponses(t, code, email, recipient)
	}

	code, email, err := makeHttpReq(fmt.Sprintf("http://%s/emails/%s/messages/first", httpAddr, recipient))
	assert.NoError(t, err)
	assert.Equal(t, 200, code, "Expected HTTP 200 OK")
	testHttpResponses(t, code, email, recipient)

	code, email, err = makeHttpReq(fmt.Sprintf("http://%s/emails/%s/messages/last", httpAddr, recipient))
	assert.NoError(t, err)
	assert.Equal(t, 200, code, "Expected HTTP 200 OK")
	testHttpResponses(t, code, email, recipient)
}

func TestNotification(t *testing.T) {
	// 4. exercise the websocket endpoint to ensure it streams the emails correctly
	t.Parallel()
}

func testHttpResponses(t *testing.T, code int, email *web.EmailMessage, recipient string) {
	t.Helper()

	assert.Equal(t, 200, code, "Expected HTTP 200 OK")
	assert.Equal(t, "verify@example.com", email.Sender, "Expected sender to match")
	assert.Equal(t, recipient, email.Recipient, "Expected recipient to match")
	assert.Contains(t, email.Body, "click here", "Expected email body to match")
	assert.Contains(t, email.Links, "https://example.com", "Expected email to contain link")
	assert.Contains(t, email.Links, "https://example.com/wedding", "Expected email to contain link")
	assert.Contains(t, email.Links, "https://example.com/pricing", "Expected email to contain link")
}

func sendTestEmail(addr, from, to string) error {
	return smtp.SendMail(addr, nil, from, []string{to}, []byte(testEmail))
}

func makeHttpReq(url string) (int, *web.EmailMessage, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	var email = new(web.EmailMessage)
	if err := json.Unmarshal(bodyBytes, &email); err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, email, nil
}
