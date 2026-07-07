package pharmacy

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	inventorySvc "github.com/timurdianradhasejati/telemed_hub/internal/inventory/service"
	patientSvc "github.com/timurdianradhasejati/telemed_hub/internal/patient/service"
	prescriptionSvc "github.com/timurdianradhasejati/telemed_hub/internal/prescription/service"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/handler"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/repository"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/service"
	walletSvc "github.com/timurdianradhasejati/telemed_hub/internal/wallet"
	"github.com/timurdianradhasejati/telemed_hub/internal/notification"
)

type Module struct {
	Handler *handler.OrderHandler
	Service service.OrderService
}

func NewModule(
	db *pgxpool.Pool,
	rdb *redis.Client,
	cfg *config.Config,
	log *slog.Logger,
	prescriptionSvc prescriptionSvc.PrescriptionService,
	inventorySvc inventorySvc.InventoryService,
	patientSvc patientSvc.PatientService,
	walletSvc walletSvc.WalletService,
	notificationSvc notification.NotificationService,
) *Module {
	repo := repository.NewPostgresRepository(db)
	svc := service.NewOrderService(repo, db, prescriptionSvc, inventorySvc, patientSvc, walletSvc, notificationSvc)
	h := handler.NewOrderHandler(svc, cfg, rdb, log)

	return &Module{
		Handler: h,
		Service: svc,
	}
}
