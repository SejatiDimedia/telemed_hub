package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/timurdianradhasejati/telemed_hub/internal/config"
	"github.com/timurdianradhasejati/telemed_hub/internal/admin/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/admin/service"
	"github.com/timurdianradhasejati/telemed_hub/pkg/logger"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type MockAdminService struct {
	mock.Mock
}

func (m *MockAdminService) ListAuditLogs(ctx context.Context, actorID uuid.UUID, roles []string, filter dto.ListAuditLogsFilter) ([]*dto.AuditLogResponse, int, error) {
	args := m.Called(ctx, actorID, roles, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*dto.AuditLogResponse), args.Int(1), args.Error(2)
}

var _ service.AdminService = (*MockAdminService)(nil)

func newTestHandler(svc service.AdminService) (*AdminHandler, chi.Router) {
	cfg := &config.Config{}
	log := logger.Setup("error")
	h := NewAdminHandler(svc, cfg, nil, log)

	r := chi.NewRouter()
	r.Get("/admin/audit-logs", h.ListAuditLogs)

	return h, r
}

func withAuth(req *http.Request, userID uuid.UUID, roles []string) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
	ctx = context.WithValue(ctx, middleware.RolesContextKey, roles)
	return req.WithContext(ctx)
}

func TestAdminHandler_ListAuditLogs_Success(t *testing.T) {
	mockSvc := new(MockAdminService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	filter := dto.ListAuditLogsFilter{
		Page:  1,
		Limit: 10,
	}

	expectedLogs := []*dto.AuditLogResponse{
		{
			ID:         uuid.New().String(),
			ActorID:    userID.String(),
			Action:     "medical_record.viewed",
			TargetType: "medical_records",
			TargetID:   uuid.New().String(),
			IPAddress:  "127.0.0.1",
			CreatedAt:  "2026-07-07T00:00:00Z",
		},
	}

	mockSvc.On("ListAuditLogs", mock.Anything, userID, []string{"admin"}, filter).Return(expectedLogs, 1, nil).Once()

	req, _ := http.NewRequest(http.MethodGet, "/admin/audit-logs?page=1&limit=10", nil)
	req = withAuth(req, userID, []string{"admin"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var envelope struct {
		Success    bool                    `json:"success"`
		Data       []dto.AuditLogResponse `json:"data"`
		Pagination struct {
			Page       int `json:"page"`
			Limit      int `json:"limit"`
			TotalItems int `json:"total_items"`
			TotalPages int `json:"total_pages"`
		} `json:"pagination"`
	}

	require.NoError(t, json.NewDecoder(rec.Body).Decode(&envelope))
	assert.True(t, envelope.Success)
	assert.Len(t, envelope.Data, 1)
	assert.Equal(t, 1, envelope.Pagination.TotalItems)
	assert.Equal(t, 1, envelope.Pagination.TotalPages)
	mockSvc.AssertExpectations(t)
}

func TestAdminHandler_ListAuditLogs_ForbiddenForDoctor(t *testing.T) {
	mockSvc := new(MockAdminService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	filter := dto.ListAuditLogsFilter{
		Page:  1,
		Limit: 10,
	}

	mockSvc.On("ListAuditLogs", mock.Anything, userID, []string{"doctor"}, filter).Return(nil, 0, service.ErrUnauthorized).Once()

	req, _ := http.NewRequest(http.MethodGet, "/admin/audit-logs?page=1&limit=10", nil)
	req = withAuth(req, userID, []string{"doctor"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}
