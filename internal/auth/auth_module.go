package auth

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/handler"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/auth/service"
)

// Module wraps all components of the authentication domain.
type Module struct {
	Handler *handler.AuthHandler
	Service service.AuthService
}

// NewModule constructs the repository, service, and handler layers for Auth.
func NewModule(db *pgxpool.Pool, rdb *redis.Client, cfg *config.Config, logger *slog.Logger) *Module {
	repo := repository.NewPostgresRepository(db)
	svc := service.NewAuthService(repo, rdb, cfg, logger)
	h := handler.NewAuthHandler(svc, cfg, rdb, logger)

	return &Module{
		Handler: h,
		Service: svc,
	}
}
