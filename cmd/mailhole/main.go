package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2/log"
	"github.com/kmulvey/mailhole/internal/pkg/mailhole"
)

func main() {
	var ctx = context.Background()

	dbURL := os.Getenv("DB_URL")
	smtpAddr := os.Getenv("SMTP_ADDR")
	httpAddr := os.Getenv("HTTP_ADDR")

	var hole, err = mailhole.New(ctx, dbURL, smtpAddr, httpAddr)
	if err != nil {
		log.Fatalf("Error starting application: %v", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop

	if err := hole.Stop(); err != nil {
		log.Errorf("Error stopping application: %v", err)
	}
}
