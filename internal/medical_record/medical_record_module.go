package medical_record

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	patientSvc "github.com/timurdianradhasejati/telemed_hub/internal/patient/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/shared"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/handler"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/medical_record/service"
)

type Module struct {
	Handler *handler.MedicalRecordHandler
	Service service.MedicalRecordService
}

func NewModule(
	db *pgxpool.Pool,
	rdb *redis.Client,
	cfg *config.Config,
	log *slog.Logger,
	patientSvc patientSvc.PatientService,
	auditSvc shared.AuditService,
) *Module {
	repo := repository.NewPostgresRepository(db)
	svc := service.NewMedicalRecordService(repo, patientSvc, auditSvc)
	h := handler.NewMedicalRecordHandler(svc, cfg, rdb, log)

	return &Module{
		Handler: h,
		Service: svc,
	}
}
