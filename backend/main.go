package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"jade-gamble/backend/api"
	"jade-gamble/backend/store"
)

func main() {
	port := getenv("PORT", "3002")
	dbPath := getenv("DB_PATH", "./data/jade.db")
	if err := os.MkdirAll(dbPath[:strings.LastIndex(dbPath, "/")], 0o755); err != nil {
		// windows paths may use backslashes
		_ = err
	}
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer st.Close()

	a := &api.API{
		Store:               st,
		BaseURL:             getenv("BASE_URL", "http://localhost:3002"),
		DiscordClientID:     os.Getenv("DISCORD_CLIENT_ID"),
		DiscordClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
		DiscordRedirectURI:  os.Getenv("DISCORD_REDIRECT_URI"),
		MockAuth:            os.Getenv("MOCK_AUTH") == "1",
	}

	mux := a.Routes()
	mux.Handle("/", spaHandler())

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("jade-gamble listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// spaHandler serves the embedded frontend (prod) — index.html, app.js,
// stone-render.js — and falls back to index.html for unknown non-API paths.
func spaHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		data, err := frontendFS.ReadFile("frontend/" + name)
		if err != nil {
			// SPA fallback to index
			data, err = frontendFS.ReadFile("frontend/index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			name = "index.html"
		}
		ctype := "application/octet-stream"
		switch {
		case strings.HasSuffix(name, ".html"):
			ctype = "text/html; charset=utf-8"
		case strings.HasSuffix(name, ".js"):
			ctype = "text/javascript; charset=utf-8"
		case strings.HasSuffix(name, ".css"):
			ctype = "text/css"
		}
		w.Header().Set("Content-Type", ctype)
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(data)
	})
}

//go:embed frontend
var frontendFS embed.FS

var _ = strconv.Itoa
