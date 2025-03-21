package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
	"github.com/pressly/goose/v3"
	"github.com/trancong12102/jina-http-proxy/config"
	"github.com/trancong12102/jina-http-proxy/key"
	"github.com/trancong12102/jina-http-proxy/proxy"
)

const (
	ReadHeaderTimeout = 5 * time.Second
	ProxyListenAddr   = ":5555"
	ApiListenAddr     = ":5556"
)

func runSrv() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errGroup, ctx := errgroup.WithContext(ctx)

	// Load config
	serverConfig, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Create database
	db, err := sql.Open("pgx", serverConfig.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Run migrations
	err = goose.Up(db, serverConfig.MigrationDir)
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	// Create key repository
	keyRepository := key.NewKeyDBRepository(db)

	// Create key service
	keyService := key.NewKeyService(keyRepository)

	// Create key handler
	keyHandler := key.NewKeyHandler(keyService)

	// Create proxy handler
	proxyHandler := proxy.CreateProxyHandler(ctx, keyService)

	// Create apiRouter
	apiRouter := createApiRouter(keyHandler)

	// Create apiHttpServer
	apiHttpServer := &http.Server{
		Addr:              ApiListenAddr,
		ReadHeaderTimeout: ReadHeaderTimeout,
		Handler:           apiRouter,
	}

	// Create proxyHttpServer
	proxyHttpServer := &http.Server{
		Addr:              ProxyListenAddr,
		ReadHeaderTimeout: ReadHeaderTimeout,
		Handler:           proxyHandler,
	}

	// Start apiHttpServer
	errGroup.Go(func() error {
		slog.Info("api server started", slog.String("listen_addr", ApiListenAddr))

		listenErr := apiHttpServer.ListenAndServe()
		if listenErr != nil {
			return fmt.Errorf("http server listen: %w", listenErr)
		}

		return nil
	})

	// Shutdown apiHttpServer
	errGroup.Go(func() error {
		<-ctx.Done()

		shutdownErr := apiHttpServer.Shutdown(ctx)
		if shutdownErr != nil {
			return fmt.Errorf("http server shutdown: %w", shutdownErr)
		}

		return nil
	})

	// Start proxyHttpServer
	errGroup.Go(func() error {
		slog.Info("proxy server started", slog.String("listen_addr", ProxyListenAddr))

		listenErr := proxyHttpServer.ListenAndServe()
		if listenErr != nil {
			return fmt.Errorf("http server listen: %w", listenErr)
		}

		return nil
	})

	// Shutdown proxyHttpServer
	errGroup.Go(func() error {
		<-ctx.Done()

		shutdownErr := proxyHttpServer.Shutdown(ctx)
		if shutdownErr != nil {
			return fmt.Errorf("http server shutdown: %w", shutdownErr)
		}

		return nil
	})

	err = errGroup.Wait()

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("error group wait: %w", err)
	}

	return nil
}

func main() {
	err := runSrv()
	if err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
