package consultation

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	appointmentService "github.com/timurdianradhasejati/telemed_hub/internal/appointment/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/handler"
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/consultation/service"
)

type Module struct {
	Handler *handler.ConsultationHandler
	Service service.ConsultationService
}

func NewModule(
	db *pgxpool.Pool,
	rdb *redis.Client,
	cfg *config.Config,
	log *slog.Logger,
	appointmentSvc appointmentService.AppointmentService,
) *Module {
	repo := repository.NewPostgresRepository(db)
	svc := service.NewConsultationService(repo, appointmentSvc)
	h := handler.NewConsultationHandler(svc, cfg, rdb, log)

	return &Module{
		Handler: h,
		Service: svc,
	}
}
