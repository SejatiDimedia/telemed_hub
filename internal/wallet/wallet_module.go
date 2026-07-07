package wallet

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/handler"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet/service"
)

type Module struct {
	Handler *handler.WalletHandler
	Service WalletService
}

func NewModule(
	db *pgxpool.Pool,
	rdb *redis.Client,
	cfg *config.Config,
	log *slog.Logger,
) *Module {
	repo := repository.NewPostgresRepository(db)
	svc := service.NewWalletService(repo, db)
	h := handler.NewWalletHandler(svc, cfg, rdb, log)

	return &Module{
		Handler: h,
		Service: svc,
	}
}
