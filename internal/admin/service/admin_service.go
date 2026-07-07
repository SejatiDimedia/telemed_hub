package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/admin/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/admin/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/admin/repository"
)

var (
	ErrUnauthorized = errors.New("unauthorized: only admins can view audit logs")
)

type AdminService interface {
	ListAuditLogs(ctx context.Context, actorID uuid.UUID, roles []string, filter dto.ListAuditLogsFilter) ([]*dto.AuditLogResponse, int, error)
}

type AdminServiceImpl struct {
	repo repository.AdminRepository
}

func NewAdminService(repo repository.AdminRepository) *AdminServiceImpl {
	return &AdminServiceImpl{repo: repo}
}

func (s *AdminServiceImpl) ListAuditLogs(ctx context.Context, actorID uuid.UUID, roles []string, filter dto.ListAuditLogsFilter) ([]*dto.AuditLogResponse, int, error) {
	isAdmin := false
	for _, r := range roles {
		if r == "admin" {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		return nil, 0, ErrUnauthorized
	}

	list, total, err := s.repo.ListAuditLogs(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	respList := make([]*dto.AuditLogResponse, 0, len(list))
	for _, l := range list {
		respList = append(respList, toResponse(l))
	}
	return respList, total, nil
}

func toResponse(l *model.AuditLog) *dto.AuditLogResponse {
	return &dto.AuditLogResponse{
		ID:         l.ID.String(),
		ActorID:    l.ActorID.String(),
		Action:     l.Action,
		TargetType: l.TargetType,
		TargetID:   l.TargetID.String(),
		IPAddress:  l.IPAddress,
		UserAgent:  l.UserAgent,
		Metadata:   l.Metadata,
		CreatedAt:  l.CreatedAt.Format(time.RFC3339),
	}
}
