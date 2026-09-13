package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/config"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/httpapi"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
	"github.com/cmcgeedev/deepcutsCRM/internal/storage"
	"github.com/cmcgeedev/deepcutsCRM/web"
)

func runServe(cfg config.Config, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d, err := db.OpenAndMigrate(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer d.Close()
	svc := service.New(d, cfg.Location)
	proofs, err := storage.NewLocal(filepath.Join(cfg.DataDir, "uploads"))
	if err != nil {
		return err
	}
	svc.Proofs = proofs
	a := auth.New(queries.New(d))
	webFS, err := web.FS()
	if err != nil {
		return err
	}
	handler := httpapi.NewRouter(httpapi.Deps{Svc: svc, Auth: a, Proofs: proofs, Secure: cfg.TLS() || cfg.SecureCookies, Web: webFS})
	srv := &http.Server{Addr: cfg.Addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = queries.New(d).DeleteExpiredSessions(ctx, time.Now().UTC())
			}
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		scheme := "http"
		if cfg.TLS() {
			scheme = "https"
		}
		log.Printf("deepcuts listening on %s://localhost%s (office: /office, driver: /driver)", scheme, cfg.Addr)
		if cfg.TLS() {
			errCh <- srv.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
		} else {
			errCh <- srv.ListenAndServe()
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-stop:
		fmt.Println("shutting down")
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
		defer shutdownCancel()
		err := srv.Shutdown(shutdownCtx)
		cancel()
		return err
	}
}
