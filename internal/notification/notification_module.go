package notification

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/handler"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/service"
)

type Module struct {
	Handler *handler.NotificationHandler
	Service NotificationService
	Worker  *NotificationWorker
}

func NewModule(
	db *pgxpool.Pool,
	rdb *redis.Client,
	cfg *config.Config,
	log *slog.Logger,
) *Module {
	repo := repository.NewPostgresRepository(db)
	svc := service.NewNotificationService(repo, rdb, log)
	h := handler.NewNotificationHandler(svc, cfg, rdb)
	worker := service.NewNotificationWorker(repo, rdb, log)

	return &Module{
		Handler: h,
		Service: svc,
		Worker:  worker,
	}
}

func (m *Module) Start() {
	m.Worker.Start()
}

func (m *Module) Stop() {
	m.Worker.Stop()
}
