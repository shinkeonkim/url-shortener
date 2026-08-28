package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shinkeonkim/url-shortener/internal/config"
	"github.com/shinkeonkim/url-shortener/internal/httpapi"
	"github.com/shinkeonkim/url-shortener/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := store.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer db.Close()
	handler := httpapi.New(db).WithBaseDomain(cfg.BaseDomain).WithAuth(httpapi.AuthConfig{
		Username: cfg.AdminUser, PasswordHash: cfg.AdminHash, Token: cfg.AdminToken,
		SessionKey: cfg.SessionKey, CookieSecure: cfg.CookieSecure,
	})
	srv := &http.Server{Addr: cfg.Address, Handler: handler, ReadHeaderTimeout: cfg.ReadTimeout, WriteTimeout: cfg.WriteTimeout}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go maintain(ctx, db)
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	slog.Info("server listening", "address", cfg.Address)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
