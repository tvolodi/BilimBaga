package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/bilimbaga/bilimbaga/internal/categories"
	"github.com/bilimbaga/bilimbaga/internal/config"
	dbpkg "github.com/bilimbaga/bilimbaga/internal/db"
	"github.com/bilimbaga/bilimbaga/internal/departments"
	"github.com/bilimbaga/bilimbaga/internal/exams"
	"github.com/bilimbaga/bilimbaga/internal/portal"
	"github.com/bilimbaga/bilimbaga/internal/questions"
	"github.com/bilimbaga/bilimbaga/internal/rbac"
	"github.com/bilimbaga/bilimbaga/internal/router"
	"github.com/bilimbaga/bilimbaga/internal/sessions"
	"github.com/bilimbaga/bilimbaga/internal/tags"
	"github.com/bilimbaga/bilimbaga/internal/tenant"
	"github.com/bilimbaga/bilimbaga/internal/users"
)

func main() {
	// appCtx is cancelled on SIGINT/SIGTERM; used by background jobs for clean shutdown.
	appCtx, stopApp := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopApp()

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

	// Set up structured logger and audit writer.
	logger := slog.Default()
	auditWriter := audit.NewWriter(db, logger)

	// Wire up tenant configuration.
	tenantRepo := tenant.NewRepository(db)
	tenantSvc := tenant.NewService(tenantRepo)
	tenantHandler := tenant.NewHandler(tenantSvc, auditWriter)

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
	authHandler := auth.NewHandler(authSvc, auditWriter)

	// Load the RBAC permission cache.
	rbacCache := rbac.NewCache()
	if err := rbacCache.Load(db); err != nil {
		fmt.Fprintf(os.Stderr, "startup error: load rbac cache: %v\n", err)
		os.Exit(1)
	}
	log.Println("rbac permission cache loaded")

	// Wire up department management.
	deptRepo := departments.NewRepository(db)
	deptSvc := departments.NewService(deptRepo)
	deptHandler := departments.NewHandler(deptSvc, auditWriter)

	// Wire up user management.
	usersRepo := users.NewRepository(db)
	usersSvc := users.NewService(usersRepo)
	usersHandler := users.NewHandler(usersSvc, auditWriter)

	// Wire up audit log.
	auditSvc := audit.NewService(db)
	auditHandler := audit.NewHandler(auditSvc, auditWriter)

	// Wire up categories and tags (FR-BB21).
	categoriesRepo := categories.NewRepository(db)
	categoriesSvc := categories.NewService(categoriesRepo)
	categoriesHandler := categories.NewHandler(categoriesSvc, auditWriter)

	tagsRepo := tags.NewRepository(db)
	tagsSvc := tags.NewService(tagsRepo)
	tagsHandler := tags.NewHandler(tagsSvc, auditWriter)

	// Wire up questions (FR-BB22/FR-BB23).
	questionsRepo := questions.NewRepository(db)
	questionsSvc := questions.NewService(questionsRepo)
	questionsHandler := questions.NewHandler(questionsSvc, auditWriter)

	// Wire up question translations (FR-BB24).
	translationsRepo := questions.NewTranslationRepository(db)
	translationsSvc := questions.NewTranslationService(translationsRepo, tenantSvc)
	translationsHandler := questions.NewTranslationHandler(translationsSvc, auditWriter)

	// Wire up exams (FR-BB31).
	examsRepo := exams.NewRepository(db)
	examsSvc := exams.NewService(examsRepo)
	examsHandler := exams.NewHandler(examsSvc, auditWriter)

	// Wire up employee exam portal (FR-BB34).
	portalRepo := portal.NewRepository(db)
	portalSvc := portal.NewService(portalRepo)
	portalHandler := portal.NewHandler(portalSvc)

	// Wire up session creation (FR-BB35).
	gradingEngine := sessions.NewGradingEngine()
	sessionsRepo := sessions.NewRepository(db, gradingEngine)
	sessionsSvc := sessions.NewService(sessionsRepo)
	sessionsHandler := sessions.NewHandler(sessionsSvc)

	// Start FR-BB310 auto-submit background job.
	go sessions.AutoSubmitJob(appCtx, db, logger, gradingEngine)

	r := router.New(tenantHandler, authHandler, deptHandler, usersHandler, auditHandler, categoriesHandler, tagsHandler, questionsHandler, translationsHandler, examsHandler, portalHandler, sessionsHandler, cfg.JWTSecret, rbacCache)

	srv := &http.Server{
		Addr:         ":" + cfg.APIPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("API listening on :%s", cfg.APIPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for SIGINT / SIGTERM (appCtx is cancelled by signal.NotifyContext).
	<-appCtx.Done()
	stopApp()
	log.Println("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}

	log.Println("server stopped")
}
