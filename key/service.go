package key

import "context"

type KeyRepository interface {
	InsertKey(ctx context.Context, params InsertKeyParams) error
	UseBestKey(ctx context.Context) (*string, error)
	GetKeyStats(ctx context.Context) (*KeyStats, error)
}

type KeyService struct {
	repo KeyRepository
}

func (s *KeyService) InsertKey(ctx context.Context, params InsertKeyParams) error {
	return s.repo.InsertKey(ctx, params)
}

func (s *KeyService) UseBestKey(ctx context.Context) (*string, error) {
	return s.repo.UseBestKey(ctx)
}

func (s *KeyService) GetKeyStats(ctx context.Context) (*KeyStats, error) {
	return s.repo.GetKeyStats(ctx)
}

func NewKeyService(repo KeyRepository) *KeyService {
	return &KeyService{repo: repo}
}
