package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// HTML-nav misses fall back to index.html; others 404
func NewSPAHandler(distFS fs.FS) (http.Handler, error) {
	index, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		cleaned := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if cleaned == "" {
			cleaned = "index.html"
		}

		if info, err := fs.Stat(distFS, cleaned); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		if isHTMLNavigation(r) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			if r.Method == http.MethodGet {
				_, _ = w.Write(index)
			}
			return
		}

		http.NotFound(w, r)
	}), nil
}

func isHTMLNavigation(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/html")
}
