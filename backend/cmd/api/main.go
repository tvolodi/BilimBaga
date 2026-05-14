package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/bilimbaga/bilimbaga/internal/config"
	dbpkg "github.com/bilimbaga/bilimbaga/internal/db"
	"github.com/bilimbaga/bilimbaga/internal/router"
	"github.com/bilimbaga/bilimbaga/internal/tenant"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "startup error: %v\n", err)
		os.Exit(1)
	}

	db, err := dbpkg.New(dbpkg.Config{
		Host:            cfg.DBHost,
		Port:            cfg.DBPort,
		Name:            cfg.DBName,
		User:            cfg.DBUser,
		Password:        cfg.DBPassword,
		SSLMode:         cfg.DBSSLMode,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxIdleTime: time.Duration(cfg.DBConnMaxIdleSeconds) * time.Second,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "startup error: open database: %v\n", err)
		os.Exit(1)
	}

	if err := dbpkg.RunMigrations(db, "migrations"); err != nil {
		fmt.Fprintf(os.Stderr, "startup error: run migrations: %v\n", err)
		os.Exit(1)
	}
	log.Println("database migrations applied")

	// Wire up tenant configuration.
	tenantRepo := tenant.NewRepository(db)
	tenantSvc := tenant.NewService(tenantRepo)
	tenantHandler := tenant.NewHandler(tenantSvc)

	// Populate the tenant config cache before accepting requests.
	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 10*time.Second)
	if err := tenantSvc.LoadCache(startupCtx); err != nil {
		cancelStartup()
		fmt.Fprintf(os.Stderr, "startup error: load tenant config cache: %v\n", err)
		os.Exit(1)
	}
	cancelStartup()
	log.Println("tenant config cache loaded")

	// Wire up authentication.
	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(auth.ServiceConfig{
		JWTSecret:         cfg.JWTSecret,
		JWTAccessTTLMin:   cfg.JWTAccessTTLMinutes,
		JWTRefreshTTLDays: cfg.JWTRefreshTTLDays,
		BcryptCost:        cfg.BcryptCost,
		CookieDomain:      cfg.CookieDomain,
		CookieSecure:      cfg.CookieSecure,
	}, authRepo)
	authHandler := auth.NewHandler(authSvc)

	r := router.New(tenantHandler, authHandler, cfg.JWTSecret)

	srv := &http.Server{
		Addr:         ":" + cfg.APIPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("API listening on :%s", cfg.APIPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}

	log.Println("server stopped")
}
