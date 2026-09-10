package main

import (
	"fmt"
	"io/fs"
	"net/http"

	"project/internal/platform/server"
	"project/web"
)

// Own file: keeps main.go one-call-per-concern
func registerFrontend(mux *http.ServeMux) error {
	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		return fmt.Errorf("server: opening embedded frontend: %w", err)
	}

	spaHandler, err := server.NewSPAHandler(dist)
	if err != nil {
		return fmt.Errorf("server: building SPA handler: %w", err)
	}

	mux.Handle("/", spaHandler)
	return nil
}
