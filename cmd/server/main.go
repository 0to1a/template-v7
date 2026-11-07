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

// One embedded field per domain; together they satisfy gen.Service
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

	// Validate the rest of the config before touching the database
	jwtManager, err := auth.NewJWTManager(cfg.JWTSecret)
	if err != nil {
		return err
	}
	firebase, err := auth.ParseFirebaseConfig(cfg.FirebaseConfig)
	if err != nil {
		return err
	}
	mailSender, err := mail.New(cfg.MailURL)
	if err != nil {
		return err
	}

	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	migrations, err := fs.Sub(dbfs.Migrations, "migrations")
	if err != nil {
		return err
	}
	if err := database.Migrate(ctx, pool, migrations); err != nil {
		return err
	}

	queries := db.New(pool)

	authSvc := auth.NewService(auth.NewRepository(queries), auth.NewEmailLoginCodeSender(mailSender), jwtManager, cfg.IsGuestRegistration)
	if firebase != nil {
		authSvc.WithGoogle(*firebase, auth.NewFirebaseVerifier(firebase.ProjectID))
	}

	svc := service{
		healthService: health.NewService(pool),
		authService:   authSvc,
		userService:   user.NewService(user.NewRepository(queries)),
	}

	// Jobs declared under apic.yaml crons:; stops when ctx is cancelled
	go gen.RunCrons(ctx, svc)

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
