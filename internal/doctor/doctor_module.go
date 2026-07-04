package doctor

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/handler"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/shared"
)

// Module wraps all components of the Doctor domain.
type Module struct {
	Handler *handler.DoctorHandler
	Service service.DoctorService
}

// NewModule constructs the repository, service, and handler layers for Doctor.
func NewModule(db *pgxpool.Pool, rdb *redis.Client, cfg *config.Config, auditService shared.AuditService, logger *slog.Logger) *Module {
	repo := repository.NewPostgresRepository(db)
	svc := service.NewDoctorService(repo, auditService)
	h := handler.NewDoctorHandler(svc, cfg, rdb, logger)

	return &Module{
		Handler: h,
		Service: svc,
	}
}
