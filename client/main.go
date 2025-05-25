package main

import (
	"context"
	"errors"
	"github.com/my-auth/client/api"
	myoauth2 "github.com/my-auth/client/oauth2"
	"golang.org/x/oauth2"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	oauthConfig *oauth2.Config
	tmpl        *template.Template
	client      *api.Client
)

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	oauthConfig = &oauth2.Config{
		ClientID:     "your-client-id",
		ClientSecret: "your-client-secret",
		RedirectURL:  "http://localhost:16666/callback",
		Scopes:       []string{"read", "write"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.com/oauth/authorize",
			TokenURL: "https://example.com/oauth/token",
		},
	}

	tmpl = template.Must(template.ParseGlob("templates/*.html"))
}

func main() {
	if err := run(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/login", handleLogin)
	mux.HandleFunc("/callback", handleCallback)
	server := &http.Server{
		Addr:    ":16666",
		Handler: mux,
	}

	shutdownCh := make(chan struct{})
	go func() {
		<-sigCh
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		slog.Info("shutting down server...")

		if err := server.Shutdown(ctx); err != nil {
			slog.Info("failed to shutdown server", "error", err)
		}
		close(shutdownCh)
	}()

	slog.Info("starting server...", "address", server.Addr)

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	<-shutdownCh
	return nil
}

func handleIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		slog.Error("failed to execute template", "error", err)
		return
	}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := myoauth2.NewState()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		slog.Error("failed to generate state", "error", err)
		return
	}
	http.Redirect(w, r, oauthConfig.AuthCodeURL(state), http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {

}
