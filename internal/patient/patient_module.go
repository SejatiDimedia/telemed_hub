package patient

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/handler"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/service"
)

// Module wraps all components of the Patient domain.
type Module struct {
	Handler *handler.PatientHandler
	Service service.PatientService
}

// NewModule constructs the repository, service, and handler layers for Patient.
func NewModule(db *pgxpool.Pool, rdb *redis.Client, cfg *config.Config, logger *slog.Logger) *Module {
	repo := repository.NewPostgresRepository(db)
	svc := service.NewPatientService(repo)
	h := handler.NewPatientHandler(svc, cfg, rdb, logger)

	return &Module{
		Handler: h,
		Service: svc,
	}
}
