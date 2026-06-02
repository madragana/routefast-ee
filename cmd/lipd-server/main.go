// Command lipd-server is the RouteFast Enterprise Edition central credential
// server. It issues ed25519 keypairs and bearer tokens to Community Edition
// nodes, distributes policy bundles, and maintains an append-only audit trail,
// all over mutually authenticated TLS.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/madragana/routefast-ee/internal/api"
	"github.com/madragana/routefast-ee/internal/config"
	"github.com/madragana/routefast-ee/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	store, err := storage.New(ctx, cfg.DatabaseURL, cfg.MaxDBConns)
	cancel()
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	defer store.Close()

	srv := &api.Server{Store: store, Cfg: cfg}

	tlsCfg, err := buildTLS(cfg)
	if err != nil {
		log.Fatalf("tls: %v", err)
	}

	httpSrv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Routes(),
		TLSConfig:         tlsCfg,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("lipd-server listening on %s (mTLS)", cfg.ListenAddr)
		if err := httpSrv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutCancel()
	_ = httpSrv.Shutdown(shutCtx)
}

// buildTLS configures TLS 1.3 with mandatory client-certificate verification.
func buildTLS(cfg *config.Config) (*tls.Config, error) {
	caPEM, err := os.ReadFile(cfg.TLSClientCAFile)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)

	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  pool,
	}, nil
}
