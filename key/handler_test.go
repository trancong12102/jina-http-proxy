package key

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockKeyService is a mock implementation of KeyBiz
type MockKeyService struct {
	mock.Mock
}

func (m *MockKeyService) InsertKey(ctx context.Context, params InsertKeyParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

func (m *MockKeyService) UseBestKey(ctx context.Context) (*string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	key := args.String(0)
	return &key, args.Error(1)
}

func (m *MockKeyService) GetKeyStats(ctx context.Context) (*KeyStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*KeyStats), args.Error(1)
}

func TestKeyHandler_InsertKey(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockKeyService)
		expectedStatus int
	}{
		{
			name:        "Valid request",
			requestBody: InsertKeyRequest{Key: "test-key"},
			setupMock: func(m *MockKeyService) {
				m.On("InsertKey", mock.Anything, InsertKeyParams{Key: "test-key"}).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:        "Service error",
			requestBody: InsertKeyRequest{Key: "test-key"},
			setupMock: func(m *MockKeyService) {
				m.On("InsertKey", mock.Anything, InsertKeyParams{Key: "test-key"}).Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Invalid request body",
			requestBody:    "invalid-json", // Will cause JSON decode error
			setupMock:      func(m *MockKeyService) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockKeyService)
			tc.setupMock(mockService)
			handler := NewKeyHandler(mockService)

			// Create request
			var reqBody []byte
			var err error
			if str, ok := tc.requestBody.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, err = json.Marshal(tc.requestBody)
				assert.NoError(t, err)
			}

			req, err := http.NewRequest("POST", "/keys", bytes.NewBuffer(reqBody))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handler.InsertKey(rr, req)

			// Check status code
			assert.Equal(t, tc.expectedStatus, rr.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestKeyHandler_GetKeyStats(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockKeyService)
		expectedStatus int
		expectedBody   *KeyStats
	}{
		{
			name: "Success",
			setupMock: func(m *MockKeyService) {
				m.On("GetKeyStats", mock.Anything).Return(&KeyStats{Count: 5, Balance: 10000}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   &KeyStats{Count: 5, Balance: 10000},
		},
		{
			name: "Service error",
			setupMock: func(m *MockKeyService) {
				m.On("GetKeyStats", mock.Anything).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockKeyService)
			tc.setupMock(mockService)
			handler := NewKeyHandler(mockService)

			// Create request
			req, err := http.NewRequest("GET", "/stats", nil)
			assert.NoError(t, err)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handler.GetKeyStats(rr, req)

			// Check status code
			assert.Equal(t, tc.expectedStatus, rr.Code)

			// Check response body for successful cases
			if tc.expectedStatus == http.StatusOK {
				var response KeyStats
				err = json.NewDecoder(rr.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedBody.Count, response.Count)
				assert.Equal(t, tc.expectedBody.Balance, response.Balance)
			}

			mockService.AssertExpectations(t)
		})
	}
}
