package admin

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/admin/handler"
	"github.com/timurdianradhasejati/telemed_hub/internal/admin/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/admin/service"
)

type Module struct {
	Handler *handler.AdminHandler
	Service service.AdminService
}

func NewModule(
	db *pgxpool.Pool,
	rdb *redis.Client,
	cfg *config.Config,
	log *slog.Logger,
) *Module {
	repo := repository.NewPostgresRepository(db)
	svc := service.NewAdminService(repo)
	h := handler.NewAdminHandler(svc, cfg, rdb, log)

	return &Module{
		Handler: h,
		Service: svc,
	}
}
