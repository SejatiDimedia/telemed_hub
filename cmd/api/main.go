package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"

	"github.com/timurdianradhasejati/telemed_hub/internal/appointment"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor"
	"github.com/timurdianradhasejati/telemed_hub/internal/healthcheck"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription"
	"github.com/timurdianradhasejati/telemed_hub/internal/shared"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet"
	"github.com/timurdianradhasejati/telemed_hub/pkg/logger"
)

func main() {
	// --- Load configuration ---
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// --- Setup logger ---
	log := logger.Setup(cfg.App.LogLevel)
	log.Info("starting TeleMedHub",
		"env", cfg.App.Env,
		"port", cfg.Server.Port,
	)

	// --- Connect to PostgreSQL ---
	ctx := context.Background()

	dbPool, err := pgxpool.New(ctx, cfg.DB.URL)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		log.Error("failed to ping postgres", "error", err)
		os.Exit(1)
	}
	log.Info("connected to postgres")

	// --- Connect to Redis ---
	redisOpts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		log.Error("failed to parse redis URL", "error", err)
		os.Exit(1)
	}

	rdb := redis.NewClient(redisOpts)
	defer func() {
		if err := rdb.Close(); err != nil {
			log.Error("failed to close redis connection", "error", err)
		}
	}()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error("failed to ping redis", "error", err)
		os.Exit(1)
	}
	log.Info("connected to redis")

	// --- Connect to MinIO ---
	minioClient, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		log.Error("failed to create minio client", "error", err)
		os.Exit(1)
	}

	// Ensure the default bucket exists
	bucketExists, err := minioClient.BucketExists(ctx, cfg.MinIO.BucketName)
	if err != nil {
		log.Error("failed to check minio bucket", "error", err)
		os.Exit(1)
	}
	if !bucketExists {
		if err := minioClient.MakeBucket(ctx, cfg.MinIO.BucketName, minio.MakeBucketOptions{}); err != nil {
			log.Error("failed to create minio bucket", "error", err, "bucket", cfg.MinIO.BucketName)
			os.Exit(1)
		}
		log.Info("created minio bucket", "bucket", cfg.MinIO.BucketName)
	}
	log.Info("connected to minio", "endpoint", cfg.MinIO.Endpoint)

	// --- Setup router ---
	r := chi.NewRouter()

	// Base middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middlewareLogger(log))
	r.Use(middleware.Recoverer)

	// Health check endpoints (no auth required)
	healthHandler := healthcheck.NewHandler(dbPool, rdb, minioClient, log)
	r.Get("/healthz", healthHandler.Healthz)
	r.Get("/readyz", healthHandler.Readyz)

	// --- Initialize Shared Services ---
	auditSvc := shared.NewAuditService(dbPool)
	walletSvc := wallet.NewWalletStub()

	// --- Initialize Modules ---
	authMod := auth.NewModule(dbPool, rdb, cfg, log)
	patientMod := patient.NewModule(dbPool, rdb, cfg, log)
	doctorMod := doctor.NewModule(dbPool, rdb, cfg, auditSvc, log)
	appointmentMod := appointment.NewModule(dbPool, cfg, rdb, log, patientMod.Service, doctorMod.Service, walletSvc)
	consultationMod := consultation.NewModule(dbPool, rdb, cfg, log, appointmentMod.Service)
	prescriptionMod := prescription.NewModule(dbPool, rdb, cfg, log, consultationMod.Service, doctorMod.Service, patientMod.Service)

	// Resolve setter DI for circular dependency
	appointmentMod.Service.SetConsultationService(consultationMod.Service)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Mount("/auth", authMod.Handler.Routes())
		r.Mount("/patients", patientMod.Handler.Routes())
		r.Mount("/doctors", doctorMod.Handler.Routes())
		r.Mount("/appointments", appointmentMod.Handler.Routes())
		r.Mount("/consultations", consultationMod.Handler.Routes())
		r.Mount("/prescriptions", prescriptionMod.Handler.Routes())
	})

	// --- Start HTTP server with graceful shutdown ---
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for errors from the server
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("HTTP server listening", "addr", srv.Addr)
		serverErrors <- srv.ListenAndServe()
	}()

	// Listen for shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive a signal or server error
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	case sig := <-quit:
		log.Info("shutting down server", "signal", sig.String())

		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, cfg.Server.ShutdownTimeout)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("forced shutdown", "error", err)
			os.Exit(1)
		}
	}

	log.Info("server stopped gracefully")
}

// middlewareLogger creates a chi middleware that logs each request using slog.
func middlewareLogger(log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap ResponseWriter to capture status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			log.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"latency_ms", time.Since(start).Milliseconds(),
				"bytes", ww.BytesWritten(),
				"remote_addr", r.RemoteAddr,
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
