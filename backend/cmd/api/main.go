package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bilimbaga/bilimbaga/internal/ai"
	"github.com/bilimbaga/bilimbaga/internal/audit"
	"github.com/bilimbaga/bilimbaga/internal/auth"
	"github.com/bilimbaga/bilimbaga/internal/categories"
	"github.com/bilimbaga/bilimbaga/internal/certificates"
	"github.com/bilimbaga/bilimbaga/internal/config"
	dbpkg "github.com/bilimbaga/bilimbaga/internal/db"
	"github.com/bilimbaga/bilimbaga/internal/departments"
	"github.com/bilimbaga/bilimbaga/internal/email"
	"github.com/bilimbaga/bilimbaga/internal/exams"
	"github.com/bilimbaga/bilimbaga/internal/portal"
	"github.com/bilimbaga/bilimbaga/internal/questions"
	"github.com/bilimbaga/bilimbaga/internal/rbac"
	"github.com/bilimbaga/bilimbaga/internal/roles"
	"github.com/bilimbaga/bilimbaga/internal/reports"
	"github.com/bilimbaga/bilimbaga/internal/router"
	"github.com/bilimbaga/bilimbaga/internal/sessions"
	"github.com/bilimbaga/bilimbaga/internal/tags"
	"github.com/bilimbaga/bilimbaga/internal/tenant"
	"github.com/bilimbaga/bilimbaga/internal/users"
	"github.com/rs/zerolog"
)

// Version is the Git commit SHA injected at build time via:
//
//	go build -ldflags "-X main.Version=$(git rev-parse --short HEAD)"
//
// It defaults to "dev" when no linker flag is provided.
var Version = "dev"

// initLogger initialises the zerolog global level and returns a configured
// structured JSON logger that writes NDJSON to stdout.  If level is invalid,
// it falls back to InfoLevel and emits a startup warning.
func initLogger(level string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	log := zerolog.New(os.Stdout).With().Timestamp().Str("version", Version).Logger()
	if err != nil {
		log.Warn().Str("log_level", level).Msg("invalid LOG_LEVEL — defaulting to info")
	}
	return log
}

// serve is the normal startup path: migrate, wire dependencies, serve HTTP until signalled.
func serve() {
	// appCtx is cancelled on SIGINT/SIGTERM; used by background jobs for clean shutdown.
	appCtx, stopApp := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopApp()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "startup error: %v\n", err)
		os.Exit(1)
	}

	// zerolog logger for the middleware chain and structured startup messages.
	zlog := initLogger(cfg.LogLevel)

	// No environment marker (e.g. APP_ENV) exists in Config, so warn whenever the
	// localhost default is in use: certificate QR verify links would point at localhost.
	if cfg.PublicAppURLDefaulted {
		zlog.Warn().Str("public_app_url", cfg.PublicAppURL).
			Msg("PUBLIC_APP_URL is not set; defaulting to localhost — certificate verify links/QR codes will be wrong in production")
	}

	// slog.Default() is kept for existing services that were written against *slog.Logger.
	// They are not migrated here to avoid a large, unrelated diff.
	slogger := slog.Default()

	db, err := openDB(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "startup error: %v\n", err)
		os.Exit(1)
	}

	if err := dbpkg.RunMigrations(db, "migrations"); err != nil {
		fmt.Fprintf(os.Stderr, "startup error: run migrations: %v\n", err)
		os.Exit(1)
	}
	zlog.Info().Msg("database migrations applied")

	// Set up structured logger and audit writer.
	auditWriter := audit.NewWriter(db, slogger)

	// Wire up email notification service (FR-BB61).
	emailSvc := email.NewEmailService(email.Config{
		Host:         cfg.SMTPHost,
		Port:         cfg.SMTPPort,
		User:         cfg.SMTPUser,
		Pass:         cfg.SMTPPass,
		TLS:          cfg.SMTPTLS,
		From:         cfg.SMTPFrom,
		APIBaseURL:   cfg.APIBaseURL,
		PublicAppURL: cfg.PublicAppURL,
	}, db, slogger)
	if cfg.SMTPFromDefaulted {
		slogger.Warn("SMTP_FROM is not set; using default sender", "default", cfg.SMTPFrom)
	}
	emailHandler := email.NewHandler(emailSvc)
	go email.StartDeadlineReminderScheduler(appCtx, emailSvc, cfg.TenantTimezone)

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
	zlog.Info().Msg("tenant config cache loaded")

	// Wire up authentication.
	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(auth.ServiceConfig{
		JWTSecret:         cfg.JWTSecret,
		JWTAccessTTLMin:   cfg.JWTAccessTTLMinutes,
		JWTRefreshTTLDays: cfg.JWTRefreshTTLDays,
		BcryptCost:        cfg.BcryptCost,
		CookieDomain:      cfg.CookieDomain,
		CookieSecure:      cfg.CookieSecure,
	}, authRepo, emailSvc)
	authHandler := auth.NewHandler(authSvc, auditWriter)

	// ISS-150/ISS-152: make the seeded super_admin safe. Applies BOOTSTRAP_ADMIN_PASSWORD (or a
	// generated one-time password) while the admin still has the default password, otherwise
	// forces a password change and warns. Secrets are never logged except the generated
	// one-time password, once.
	bootCtx, cancelBoot := context.WithTimeout(appCtx, 30*time.Second)
	bootRes, err := auth.BootstrapAdmin(bootCtx, auth.NewBootstrapStore(db), auth.BootstrapOptions{
		Password: cfg.BootstrapAdminPassword,
		Generate: cfg.BootstrapAdminGenerate,
		Cost:     cfg.BcryptCost,
	})
	cancelBoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "startup error: bootstrap admin: %v\n", err)
		os.Exit(1)
	}
	switch bootRes.Outcome {
	case auth.BootstrapPasswordGenerated:
		zlog.Warn().Str("admin_email", auth.BootstrapAdminEmail).Str("one_time_password", bootRes.GeneratedPassword).
			Msg("generated one-time admin password (shown once; change required at first login)")
	case auth.BootstrapPasswordApplied:
		zlog.Info().Str("admin_email", auth.BootstrapAdminEmail).Msg("admin password set from BOOTSTRAP_ADMIN_PASSWORD")
	}
	if bootRes.EnvPasswordInvalid {
		zlog.Warn().Msg("BOOTSTRAP_ADMIN_PASSWORD is set but invalid (needs 8-72 bytes, upper, lower, digit, not the default) and is ignored: admin password already changed or admin absent")
	}
	if bootRes.StillDefault {
		zlog.Warn().Str("admin_email", auth.BootstrapAdminEmail).
			Msg("SECURITY: seeded admin still has the default password; a password change is forced at first login. Set BOOTSTRAP_ADMIN_PASSWORD or change it now")
	}

	// ISS-181: if migration 035 skipped the lower(email) unique index because of legacy
	// case-insensitive duplicate emails, report the twins (WARN log + audit entry).
	// Never aborts startup.
	users.RunStartupDuplicateEmailCheck(appCtx, db, slogger)

	// Load the RBAC permission cache.
	rbacCache := rbac.NewCache()
	if err := rbacCache.Load(db); err != nil {
		fmt.Fprintf(os.Stderr, "startup error: load rbac cache: %v\n", err)
		os.Exit(1)
	}
	zlog.Info().Msg("rbac permission cache loaded")

	// Wire up department management.
	deptRepo := departments.NewRepository(db)
	deptSvc := departments.NewService(deptRepo)
	deptHandler := departments.NewHandler(deptSvc, auditWriter)

	// Wire up user management.
	usersRepo := users.NewRepository(db)
	usersSvc := users.WithPermissionsLookup(users.WithPermissionChecker(users.NewService(usersRepo, emailSvc), rbacCache.Has), rbacCache.PermissionsFor)
	usersHandler := users.NewHandler(usersSvc, auditWriter).WithPermissionsProvider(rbacCache.PermissionsFor)

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
	examsSvc := exams.NewService(examsRepo, emailSvc)
	examsHandler := exams.NewHandler(examsSvc, auditWriter)

	// Wire up employee exam portal (FR-BB34).
	portalRepo := portal.NewRepository(db)
	portalSvc := portal.NewService(portalRepo)
	portalHandler := portal.NewHandler(portalSvc)

	// Wire up session creation (FR-BB35).
	// If AnthropicAPIKey is configured, use AI-backed grading for short-text questions (FR-BB73).
	var gradingEngine sessions.GradingEngine
	if cfg.AnthropicAPIKey != "" {
		aiGradingClient := ai.NewAnthropicClient(cfg.AnthropicAPIKey, slogger)
		gradingEngine = sessions.NewAIGradingEngine(aiGradingClient, cfg.AnthropicModel, db, slogger)
	} else {
		gradingEngine = sessions.NewGradingEngine()
	}
	sessionsRepo := sessions.NewRepository(db, gradingEngine)
	sessionsSvc := sessions.NewService(sessionsRepo, emailSvc)
	sessionsHandler := sessions.NewHandler(sessionsSvc)

	// Start FR-BB310 auto-submit background job.
	go sessions.AutoSubmitJob(appCtx, db, slogger, gradingEngine)

	// Wire up certificates (FR-BB43).
	certsRepo := certificates.NewRepository(db)
	certsSvc := certificates.NewService(certsRepo, tenantSvc)
	certHandler := certificates.NewHandler(certsSvc, cfg.PublicAppURL)

	// Wire up dashboard metrics (FR-BB51) and export API (FR-BB54).
	reportsRepo := reports.NewRepository(db)
	reportsSvc := reports.NewService(reportsRepo)
	reportsHandler := reports.NewHandler(reportsSvc, tenantSvc)

	// Wire up AI question generation (FR-BB71).
	aiRepo := ai.NewRepository(db)
	aiClient := ai.NewAnthropicClient(cfg.AnthropicAPIKey, slogger)
	aiSvc := ai.NewService(aiRepo, aiClient, cfg.AnthropicModel, slogger, ai.WithInsightsDailyLimit(cfg.AIInsightsDailyLimit))
	aiHandler := ai.NewHandler(aiSvc)

	// Wire up role management (FR-BB117); the cache is rebuilt after every mutation.
	rolesRepo := roles.NewRepository(db)
	rolesSvc := roles.NewService(rolesRepo, func(ctx context.Context) error { return rbacCache.Load(db) })
	rolesHandler := roles.NewHandler(rolesSvc, auditWriter)

	r := router.New(
		tenantHandler, authHandler, deptHandler, usersHandler, auditHandler,
		categoriesHandler, tagsHandler, questionsHandler, translationsHandler,
		examsHandler, portalHandler, sessionsHandler, certHandler, reportsHandler,
		emailHandler, aiHandler, rolesHandler, cfg.JWTSecret, rbacCache, db, Version, zlog,
	)

	srv := &http.Server{
		Addr:         ":" + cfg.APIPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		zlog.Info().Str("addr", ":"+cfg.APIPort).Msg("API listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zlog.Fatal().Err(err).Msg("server error")
		}
	}()

	// Wait for SIGINT / SIGTERM (appCtx is cancelled by signal.NotifyContext).
	<-appCtx.Done()
	stopApp()
	zlog.Info().Msg("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		zlog.Fatal().Err(err).Msg("forced shutdown")
	}

	zlog.Info().Msg("server stopped")
}
