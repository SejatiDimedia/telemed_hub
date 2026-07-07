package repository

import (
	"context"

	"github.com/timurdianradhasejati/telemed_hub/internal/admin/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/admin/model"
)

type AdminRepository interface {
	ListAuditLogs(ctx context.Context, filter dto.ListAuditLogsFilter) ([]*model.AuditLog, int, error)
}
