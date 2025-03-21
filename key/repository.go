package key

import (
	"context"
	"database/sql"
)

type KeyDBRepository struct {
	db *sql.DB
}

// Check if KeyDBRepository implements KeyRepository
var _ KeyRepository = &KeyDBRepository{}

// InsertKey inserts a new key into the database
// Skip if the key already exists
func (r *KeyDBRepository) InsertKey(ctx context.Context, params InsertKeyParams) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO keys (key, balance) VALUES ($1, $2) ON CONFLICT DO NOTHING", params.Key, 1000000)
	return err
}

// UseBestKey returns the best key from the database.
// Best key is the key with latest created_at, then most old used_at, then most balance.
// Use SELECT FOR UPDATE SKIP LOCKED to lock the key, update used_at and return the key.
func (r *KeyDBRepository) UseBestKey(ctx context.Context) (*string, error) {
	// Create a transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback() // Intentionally ignore error from rollback as it's called from defer
		}
	}()

	// Select the best key and lock it
	var key string
	err = tx.QueryRowContext(ctx, "SELECT key FROM keys ORDER BY created_at DESC, used_at ASC, balance DESC LIMIT 1 FOR UPDATE SKIP LOCKED").Scan(&key)
	if err != nil {
		return nil, err
	}

	// Update used_at
	_, err = tx.ExecContext(ctx, "UPDATE keys SET used_at = now() WHERE key = $1", key)
	if err != nil {
		return nil, err
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &key, nil
}

// GetKeyStats returns the stats of the keys
func (r *KeyDBRepository) GetKeyStats(ctx context.Context) (*KeyStats, error) {
	var stats KeyStats
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*), SUM(balance) FROM keys").Scan(&stats.Count, &stats.Balance)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func NewKeyDBRepository(db *sql.DB) KeyRepository {
	return &KeyDBRepository{db: db}
}
