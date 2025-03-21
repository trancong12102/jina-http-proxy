package key

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockKeyRepository is a mock implementation of KeyRepository
type MockKeyRepository struct {
	mock.Mock
}

func (m *MockKeyRepository) InsertKey(ctx context.Context, params InsertKeyParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockKeyRepository) UseBestKey(ctx context.Context) (*string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	key := args.String(0)
	return &key, args.Error(1)
}

func (m *MockKeyRepository) GetKeyStats(ctx context.Context) (*KeyStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(*KeyStats), args.Error(1)
}

func TestKeyService_InsertKey(t *testing.T) {
	mockRepo := new(MockKeyRepository)
	service := NewKeyService(mockRepo)
	ctx := context.Background()
	params := InsertKeyParams{Key: "test-key"}

	// Test successful insertion
	mockRepo.On("InsertKey", ctx, params).Return(nil)
	err := service.InsertKey(ctx, params)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Test error handling
	mockRepo = new(MockKeyRepository)
	service = NewKeyService(mockRepo)
	expectedErr := assert.AnError
	mockRepo.On("InsertKey", ctx, params).Return(expectedErr)
	err = service.InsertKey(ctx, params)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockRepo.AssertExpectations(t)
}

func TestKeyService_UseBestKey(t *testing.T) {
	mockRepo := new(MockKeyRepository)
	service := NewKeyService(mockRepo)
	ctx := context.Background()
	expectedKey := "best-key"

	// Test successful retrieval
	mockRepo.On("UseBestKey", ctx).Return(expectedKey, nil)
	key, err := service.UseBestKey(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, key)
	assert.Equal(t, expectedKey, *key)
	mockRepo.AssertExpectations(t)

	// Test error handling
	mockRepo = new(MockKeyRepository)
	service = NewKeyService(mockRepo)
	expectedErr := assert.AnError
	mockRepo.On("UseBestKey", ctx).Return(nil, expectedErr)
	key, err = service.UseBestKey(ctx)
	assert.Error(t, err)
	assert.Nil(t, key)
	assert.Equal(t, expectedErr, err)
	mockRepo.AssertExpectations(t)
}

func TestKeyService_GetKeyStats(t *testing.T) {
	mockRepo := new(MockKeyRepository)
	service := NewKeyService(mockRepo)
	ctx := context.Background()
	expectedStats := &KeyStats{Count: 5, Balance: 10000}

	// Test successful stats retrieval
	mockRepo.On("GetKeyStats", ctx).Return(expectedStats, nil)
	stats, err := service.GetKeyStats(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expectedStats, stats)
	mockRepo.AssertExpectations(t)

	// Test error handling
	mockRepo = new(MockKeyRepository)
	service = NewKeyService(mockRepo)
	expectedErr := assert.AnError
	mockRepo.On("GetKeyStats", ctx).Return(&KeyStats{}, expectedErr)
	_, err = service.GetKeyStats(ctx)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	mockRepo.AssertExpectations(t)
}
