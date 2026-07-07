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
	"github.com/timurdianradhasejati/telemed_hub/internal/notification/dto"
	"github.com/timurdianradhasejati/telemed_hub/pkg/middleware"
)

type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) PublishNotification(ctx context.Context, userID uuid.UUID, channel string, typeStr string, payload map[string]any) error {
	args := m.Called(ctx, userID, channel, typeStr, payload)
	return args.Error(0)
}

func (m *MockNotificationService) ListNotifications(ctx context.Context, userID uuid.UUID, status *string, page, limit int) ([]*dto.NotificationResponse, int, error) {
	args := m.Called(ctx, userID, status, page, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*dto.NotificationResponse), args.Int(1), args.Error(2)
}

func (m *MockNotificationService) MarkAsRead(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}

func newTestHandler(svc *MockNotificationService) (*NotificationHandler, chi.Router) {
	cfg := &config.Config{}
	h := NewNotificationHandler(svc, cfg, nil)

	r := chi.NewRouter()
	r.Get("/notifications", h.List)
	r.Post("/notifications/{id}/read", h.MarkAsRead)

	return h, r
}

func withAuth(req *http.Request, userID uuid.UUID, roles []string) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
	ctx = context.WithValue(ctx, middleware.RolesContextKey, roles)
	return req.WithContext(ctx)
}

func TestNotificationHandler_List_Success(t *testing.T) {
	mockSvc := new(MockNotificationService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	statusFilter := "sent"

	expectedList := []*dto.NotificationResponse{
		{
			ID:      uuid.New().String(),
			UserID:  userID.String(),
			Channel: "email",
			Type:    "appointment_confirmed",
			Status:  "sent",
			Payload: map[string]any{"appointment_id": "999"},
		},
	}

	mockSvc.On("ListNotifications", mock.Anything, userID, &statusFilter, 1, 10).Return(expectedList, 1, nil).Once()

	req, _ := http.NewRequest(http.MethodGet, "/notifications?status=unread&page=1&limit=10", nil)
	req = withAuth(req, userID, []string{"patient"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var envelope struct {
		Success    bool                          `json:"success"`
		Data       []dto.NotificationResponse    `json:"data"`
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

func TestNotificationHandler_MarkAsRead_Success(t *testing.T) {
	mockSvc := new(MockNotificationService)
	_, r := newTestHandler(mockSvc)

	userID := uuid.New()
	notifID := uuid.New()

	mockSvc.On("MarkAsRead", mock.Anything, userID, notifID).Return(nil).Once()

	req, _ := http.NewRequest(http.MethodPost, "/notifications/"+notifID.String()+"/read", nil)
	req = withAuth(req, userID, []string{"patient"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockSvc.AssertExpectations(t)
}
