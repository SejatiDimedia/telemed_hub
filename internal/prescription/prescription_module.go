package prescription

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	consultationSvc "github.com/timurdianradhasejati/telemed_hub/internal/consultation/service"
	doctorSvc "github.com/timurdianradhasejati/telemed_hub/internal/doctor/service"
	patientSvc "github.com/timurdianradhasejati/telemed_hub/internal/patient/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/handler"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/service"
)

// Module wires up all prescription dependencies and exposes the HTTP handler and service.
type Module struct {
	Handler *handler.PrescriptionHandler
	Service service.PrescriptionService
}

// NewModule creates the prescription module, wiring all its dependencies.
func NewModule(
	db *pgxpool.Pool,
	rdb *redis.Client,
	cfg *config.Config,
	log *slog.Logger,
	consultationService consultationSvc.ConsultationService,
	doctorService doctorSvc.DoctorService,
	patientService patientSvc.PatientService,
) *Module {
	repo := repository.NewPostgresRepository(db)
	svc := service.NewPrescriptionService(repo, consultationService, doctorService, patientService)
	h := handler.NewPrescriptionHandler(svc, cfg, rdb, log)

	return &Module{
		Handler: h,
		Service: svc,
	}
}
