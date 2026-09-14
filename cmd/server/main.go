// Business logic belongs under internal/, not here
package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"

	dbfs "project/db"
	"project/internal/auth"
	gen "project/internal/gen/api"
	"project/internal/gen/db"
	"project/internal/health"
	"project/internal/mail"
	"project/internal/platform/config"
	"project/internal/platform/database"
	"project/internal/user"
)

// Aliases only exist to give each embedded field a distinct name.
type (
	healthService = health.Service
	authService   = auth.Service
	userService   = user.Service
)

// One embedded field per domain; together they satisfy gen.Service. Method
// names must not collide across domains — the compiler enforces it.
type service struct {
	*healthService
	*authService
	*userService
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Up-only; failure aborts startup, never guesses schema
	migrations, err := fs.Sub(dbfs.Migrations, "migrations")
	if err != nil {
		return err
	}
	if err := database.Migrate(ctx, pool, migrations); err != nil {
		return err
	}

	queries := db.New(pool)

	jwtManager, err := auth.NewJWTManager(cfg.JWTSecret)
	if err != nil {
		return err
	}

	// Unset MAIL_URL discards codes; malformed aborts startup
	mailSender, err := mail.NewSMTPSenderFromURL(cfg.MailURL)
	if err != nil {
		return err
	}
	var loginCodeSender auth.LoginCodeSender = auth.NoopLoginCodeSender{}
	if mailSender != nil {
		loginCodeSender = auth.NewEmailLoginCodeSender(mailSender)
	}

	svc := service{
		healthService: health.NewService(pool),
		authService:   auth.NewService(auth.NewRepository(queries), loginCodeSender, jwtManager, cfg.IsGuestRegistration),
		userService:   user.NewService(user.NewRepository(queries)),
	}

	// Jobs declared under apic.yaml crons:; stops when ctx is cancelled.
	go gen.RunCrons(ctx, svc)

	// Which routes need auth is declared in apic.yaml (group use: [auth]).
	mux := http.NewServeMux()
	if err := gen.RegisterRoutes(mux, svc, gen.Middlewares{Auth: auth.Middleware(jwtManager)}); err != nil {
		return err
	}
	if err := registerFrontend(mux); err != nil {
		return err
	}

	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}
