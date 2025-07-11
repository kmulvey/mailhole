package mailhole

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/kmulvey/mailhole/internal/pkg/mail"
	"github.com/kmulvey/mailhole/internal/pkg/web"
)

type App struct {
	dbPool     *pgxpool.Pool
	smtpServer *mail.SmtpServer
	webServer  *web.Server
}

func New(ctx context.Context, dbUrl, smtpServerAddr, httpServerAddr string) (*App, error) {

	pool, err := pgxpool.Connect(ctx, dbUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	smtpServer := mail.NewServer(pool, "localhost", smtpServerAddr)
	webServer := web.NewServer(pool, httpServerAddr)

	var app = &App{
		dbPool:     pool,
		smtpServer: smtpServer,
		webServer:  webServer,
	}

	app.CleanupDB(ctx)

	return app, nil
}

func (a *App) Stop() error {
	if err := a.smtpServer.Stop(); err != nil {
		return fmt.Errorf("error stopping SMTP server: %w", err)
	}

	if err := a.webServer.Stop(); err != nil {
		return fmt.Errorf("error stopping web server: %w", err)
	}

	a.dbPool.Close()
	return nil
}

func (a *App) CleanupDB(ctx context.Context) {
	sql := `DELETE from emails where received_at < now() - interval '1 day';`

	go func() {
		var ticker = time.NewTicker(12 * time.Hour)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, err := a.dbPool.Exec(context.Background(), sql)
				if err != nil {
					// logging is a bit lazy but this is a non critical operation and will be retried.
					log.Errorf("failed to cleanup old emails: %v", err)
				}
			}
		}
	}()
}
