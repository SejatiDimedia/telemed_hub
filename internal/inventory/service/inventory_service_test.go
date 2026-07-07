package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/timurdianradhasejati/telemed_hub/internal/inventory/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/inventory/model"
	"github.com/timurdianradhasejati/telemed_hub/internal/inventory/repository"
)

type MockMedicineRepository struct {
	mock.Mock
}

func (m *MockMedicineRepository) Create(ctx context.Context, med *model.Medicine) error {
	args := m.Called(ctx, med)
	return args.Error(0)
}

func (m *MockMedicineRepository) Update(ctx context.Context, med *model.Medicine) error {
	args := m.Called(ctx, med)
	return args.Error(0)
}

func (m *MockMedicineRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Medicine, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Medicine), args.Error(1)
}

func (m *MockMedicineRepository) GetByIDForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Medicine, error) {
	args := m.Called(ctx, tx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Medicine), args.Error(1)
}

func (m *MockMedicineRepository) List(ctx context.Context, nameFilter *string, reqPrescFilter *bool, page, limit int) ([]*model.Medicine, int, error) {
	args := m.Called(ctx, nameFilter, reqPrescFilter, page, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*model.Medicine), args.Int(1), args.Error(2)
}

func (m *MockMedicineRepository) SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	args := m.Called(ctx, id, deletedBy)
	return args.Error(0)
}

func (m *MockMedicineRepository) UpdateStock(ctx context.Context, tx pgx.Tx, id uuid.UUID, newStock int) error {
	args := m.Called(ctx, tx, id, newStock)
	return args.Error(0)
}

var _ repository.MedicineRepository = (*MockMedicineRepository)(nil)

func TestInventoryService_Create(t *testing.T) {
	mockRepo := new(MockMedicineRepository)
	svc := NewInventoryService(mockRepo)

	actorID := uuid.New()
	req := dto.CreateMedicineRequest{
		Name:                 "Aspirin 100mg",
		UnitPrice:            12000.0,
		StockQuantity:        50,
		RequiresPrescription: false,
	}

	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Medicine")).Run(func(args mock.Arguments) {
		med := args.Get(1).(*model.Medicine)
		med.ID = uuid.New()
		med.CreatedAt = time.Now()
		med.UpdatedAt = time.Now()
	}).Return(nil).Once()

	res, err := svc.Create(context.Background(), actorID, req)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "Aspirin 100mg", res.Name)
	assert.Equal(t, 12000.0, res.UnitPrice)
	assert.Equal(t, 50, res.StockQuantity)
	assert.False(t, res.RequiresPrescription)
	mockRepo.AssertExpectations(t)
}

func TestInventoryService_Update(t *testing.T) {
	mockRepo := new(MockMedicineRepository)
	svc := NewInventoryService(mockRepo)

	actorID := uuid.New()
	medID := uuid.New()
	req := dto.UpdateMedicineRequest{
		Name:                 "Aspirin Updated",
		UnitPrice:            15000.0,
		StockQuantity:        100,
		RequiresPrescription: true,
	}

	existingMed := &model.Medicine{
		ID:                   medID,
		Name:                 "Aspirin 100mg",
		UnitPrice:            12000.0,
		StockQuantity:        50,
		RequiresPrescription: false,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	mockRepo.On("GetByID", mock.Anything, medID).Return(existingMed, nil).Once()
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.Medicine")).Return(nil).Once()

	res, err := svc.Update(context.Background(), actorID, medID, req)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "Aspirin Updated", res.Name)
	assert.Equal(t, 15000.0, res.UnitPrice)
	assert.Equal(t, 100, res.StockQuantity)
	assert.True(t, res.RequiresPrescription)
	mockRepo.AssertExpectations(t)
}

func TestInventoryService_Update_NotFound(t *testing.T) {
	mockRepo := new(MockMedicineRepository)
	svc := NewInventoryService(mockRepo)

	actorID := uuid.New()
	medID := uuid.New()
	req := dto.UpdateMedicineRequest{Name: "Non Existent"}

	mockRepo.On("GetByID", mock.Anything, medID).Return(nil, repository.ErrMedicineNotFound).Once()

	_, err := svc.Update(context.Background(), actorID, medID, req)
	assert.ErrorIs(t, err, ErrMedicineNotFound)
	mockRepo.AssertExpectations(t)
}

func TestInventoryService_GetByID(t *testing.T) {
	mockRepo := new(MockMedicineRepository)
	svc := NewInventoryService(mockRepo)

	medID := uuid.New()
	med := &model.Medicine{
		ID:                   medID,
		Name:                 "Paracetamol",
		UnitPrice:            5000.0,
		StockQuantity:        500,
		RequiresPrescription: false,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	mockRepo.On("GetByID", mock.Anything, medID).Return(med, nil).Once()

	res, err := svc.GetByID(context.Background(), medID)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "Paracetamol", res.Name)
	mockRepo.AssertExpectations(t)
}

func TestInventoryService_GetByID_NotFound(t *testing.T) {
	mockRepo := new(MockMedicineRepository)
	svc := NewInventoryService(mockRepo)

	medID := uuid.New()
	mockRepo.On("GetByID", mock.Anything, medID).Return(nil, repository.ErrMedicineNotFound).Once()

	_, err := svc.GetByID(context.Background(), medID)
	assert.ErrorIs(t, err, ErrMedicineNotFound)
	mockRepo.AssertExpectations(t)
}

func TestInventoryService_List(t *testing.T) {
	mockRepo := new(MockMedicineRepository)
	svc := NewInventoryService(mockRepo)

	filter := "Aspirin"
	reqPresc := false
	medicines := []*model.Medicine{
		{
			ID:                   uuid.New(),
			Name:                 "Aspirin 100mg",
			UnitPrice:            12000.0,
			StockQuantity:        50,
			RequiresPrescription: false,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		},
	}

	mockRepo.On("List", mock.Anything, &filter, &reqPresc, 1, 20).Return(medicines, 1, nil).Once()

	res, total, err := svc.List(context.Background(), &filter, &reqPresc, 1, 20)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, res, 1)
	assert.Equal(t, "Aspirin 100mg", res[0].Name)
	mockRepo.AssertExpectations(t)
}

func TestInventoryService_Delete(t *testing.T) {
	mockRepo := new(MockMedicineRepository)
	svc := NewInventoryService(mockRepo)

	actorID := uuid.New()
	medID := uuid.New()

	mockRepo.On("SoftDelete", mock.Anything, medID, actorID).Return(nil).Once()

	err := svc.Delete(context.Background(), actorID, medID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestInventoryService_Delete_NotFound(t *testing.T) {
	mockRepo := new(MockMedicineRepository)
	svc := NewInventoryService(mockRepo)

	actorID := uuid.New()
	medID := uuid.New()

	mockRepo.On("SoftDelete", mock.Anything, medID, actorID).Return(repository.ErrMedicineNotFound).Once()

	err := svc.Delete(context.Background(), actorID, medID)
	assert.ErrorIs(t, err, ErrMedicineNotFound)
	mockRepo.AssertExpectations(t)
}
