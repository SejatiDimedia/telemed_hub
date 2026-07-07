package appointment

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/handler"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	doctorSvc "github.com/timurdianradhasejati/telemed_hub/internal/doctor/service"
	patientSvc "github.com/timurdianradhasejati/telemed_hub/internal/patient/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/wallet"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification"
)

type Module struct {
	Handler *handler.AppointmentHandler
	Service service.AppointmentService
}

func NewModule(
	db *pgxpool.Pool,
	cfg *config.Config,
	rdb *redis.Client,
	log *slog.Logger,
	patientSvc patientSvc.PatientService,
	doctorSvc doctorSvc.DoctorService,
	walletSvc wallet.WalletService,
	notificationSvc notification.NotificationService,
) *Module {
	repo := repository.NewPostgresRepository(db)
	svc := service.NewAppointmentService(repo, patientSvc, doctorSvc, walletSvc, notificationSvc, cfg.AppointmentCancelCutoffMinutes)
	h := handler.NewAppointmentHandler(svc, cfg, rdb, log)

	return &Module{
		Handler: h,
		Service: svc,
	}
}
