package main

import (
	"context"
	"errors"
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
)

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		RedirectURL:  os.Getenv("REDIRECT_URL"),
		Endpoint: oauth2.Endpoint{
			AuthURL:  os.Getenv("AUTH_SERVER_URL") + "/authorize",
			TokenURL: os.Getenv("AUTH_SERVER_URL") + "/token",
		},
		Scopes: []string{"read"},
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

	slog.Info("redirecting to auth server...")
	http.Redirect(w, r, oauthConfig.AuthCodeURL(state), http.StatusFound)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "Bad Request: missing state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Bad Request: missing code", http.StatusBadRequest)
		return
	}

	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		slog.Error("failed to exchange token", "error", err)
		return
	}

	slog.Info("login successful")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "callback.html", token); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		slog.Error("failed to execute template", "error", err)
		return
	}
}
