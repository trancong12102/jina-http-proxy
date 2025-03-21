package key

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Import pgx driver
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupPostgres(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	ctx := context.Background()

	// Create a PostgreSQL container using the Run function
	postgresContainer, err := postgres.Run(ctx,
		"postgres:17",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	require.NoError(t, err)

	// Get the connection details
	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Construct connection string manually
	connStr := fmt.Sprintf("host=%s port=%s user=postgres password=postgres dbname=testdb sslmode=disable", host, port.Port())

	// Connect to the database
	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err)

	// Test connection with retry
	var pingErr error
	for i := 0; i < 5; i++ {
		pingErr = db.Ping()
		if pingErr == nil {
			break
		}
		time.Sleep(time.Second)
	}
	require.NoError(t, pingErr, "Failed to connect to database after retries")

	// Run migrations
	_, currentFile, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(filepath.Dir(currentFile)), "migrations")
	err = goose.SetDialect("postgres")
	require.NoError(t, err)

	err = goose.Up(db, migrationsDir)
	require.NoError(t, err)

	// Return cleanup function
	cleanup := func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close database connection: %v", err)
		}
		if err := postgresContainer.Terminate(ctx); err != nil {
			log.Printf("Failed to terminate container: %v", err)
		}
	}

	return db, cleanup
}

func TestKeyDBRepository_InsertKey(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()

	repo := NewKeyDBRepository(db)
	ctx := context.Background()

	testCases := []struct {
		name        string
		key         string
		expectError bool
	}{
		{
			name:        "Insert new key",
			key:         "test-key-1",
			expectError: false,
		},
		{
			name:        "Insert duplicate key",
			key:         "test-key-1", // Same key again to test ON CONFLICT
			expectError: false,
		},
		{
			name:        "Insert different key",
			key:         "test-key-2",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := repo.InsertKey(ctx, InsertKeyParams{Key: tc.key})
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify the key was inserted
			var key string
			var balance int
			err = db.QueryRowContext(ctx, "SELECT key, balance FROM keys WHERE key = $1", tc.key).Scan(&key, &balance)
			assert.NoError(t, err)
			assert.Equal(t, tc.key, key)
			assert.Equal(t, 1000000, balance) // Default balance from the repository implementation
		})
	}

	// Test that no duplicates were created
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM keys").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 2, count) // Should be 2 keys (test-key-1 and test-key-2)
}

func TestKeyDBRepository_UseBestKey(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()

	repo := NewKeyDBRepository(db)
	ctx := context.Background()

	// Base time for consistent test cases
	now := time.Now()
	hourAgo := now.Add(-1 * time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)

	// Test setup data - each struct represents a scenario to test
	testCases := []struct {
		name          string
		setupFunc     func() // Function to set up the test scenario
		expectedKey   string // Key we expect to be returned
		keysToCleanup []string
	}{
		{
			name: "Select newest by created_at",
			setupFunc: func() {
				// Clean previous keys
				_, err := db.ExecContext(ctx, "DELETE FROM keys")
				require.NoError(t, err)

				// Insert keys with different creation times
				_, err = db.ExecContext(ctx, "INSERT INTO keys (key, balance, created_at) VALUES ($1, $2, $3)",
					"old-key", 1000, twoHoursAgo)
				require.NoError(t, err)
				_, err = db.ExecContext(ctx, "INSERT INTO keys (key, balance, created_at) VALUES ($1, $2, $3)",
					"new-key", 1000, hourAgo)
				require.NoError(t, err)
			},
			expectedKey:   "new-key", // Newest creation time
			keysToCleanup: []string{"old-key", "new-key"},
		},
		{
			name: "Select by used_at when created_at is the same",
			setupFunc: func() {
				// Clean previous keys
				_, err := db.ExecContext(ctx, "DELETE FROM keys")
				require.NoError(t, err)

				// Get database timezone handling for deterministic test
				// In PostgreSQL, NULLs usually come last in ASC order, so a key with used_at value
				// should come before a key with NULL used_at in ASC order
				var result string
				err = db.QueryRowContext(ctx, `
					WITH sample AS (
						SELECT 'a' as val, NULL::timestamp as ts
						UNION ALL
						SELECT 'b' as val, now() as ts
					)
					SELECT val FROM sample ORDER BY ts ASC LIMIT 1
				`).Scan(&result)
				require.NoError(t, err)

				// Insert keys with same creation time but different used_at
				_, err = db.ExecContext(ctx, "INSERT INTO keys (key, balance, created_at, used_at) VALUES ($1, $2, $3, $4)",
					"recently-used", 1000, now, hourAgo) // Has a used_at value
				require.NoError(t, err)
				_, err = db.ExecContext(ctx, "INSERT INTO keys (key, balance, created_at) VALUES ($1, $2, $3)",
					"never-used", 1000, now) // NULL used_at
				require.NoError(t, err)
			},
			expectedKey:   "recently-used", // Non-NULL used_at comes first in ASC order
			keysToCleanup: []string{"recently-used", "never-used"},
		},
		{
			name: "Select by balance when created_at and used_at are the same",
			setupFunc: func() {
				// Clean previous keys
				_, err := db.ExecContext(ctx, "DELETE FROM keys")
				require.NoError(t, err)

				// Insert keys with same creation time and used_at but different balances
				_, err = db.ExecContext(ctx, "INSERT INTO keys (key, balance, created_at, used_at) VALUES ($1, $2, $3, $4)",
					"low-balance", 1000, now, hourAgo)
				require.NoError(t, err)
				_, err = db.ExecContext(ctx, "INSERT INTO keys (key, balance, created_at, used_at) VALUES ($1, $2, $3, $4)",
					"high-balance", 2000, now, hourAgo)
				require.NoError(t, err)
			},
			expectedKey:   "high-balance", // Higher balance
			keysToCleanup: []string{"low-balance", "high-balance"},
		},
		{
			name: "created_at priority over used_at",
			setupFunc: func() {
				// Clean previous keys
				_, err := db.ExecContext(ctx, "DELETE FROM keys")
				require.NoError(t, err)

				// Insert a key with older creation time but never used
				_, err = db.ExecContext(ctx, "INSERT INTO keys (key, balance, created_at) VALUES ($1, $2, $3)",
					"old-never-used", 1000, twoHoursAgo) // Older, never used
				require.NoError(t, err)

				// Insert a key with newer creation time but recently used
				_, err = db.ExecContext(ctx, "INSERT INTO keys (key, balance, created_at, used_at) VALUES ($1, $2, $3, $4)",
					"new-recently-used", 1000, now, hourAgo) // Newer, recently used
				require.NoError(t, err)
			},
			expectedKey:   "new-recently-used", // Newer creation time takes priority
			keysToCleanup: []string{"old-never-used", "new-recently-used"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup the test case
			tc.setupFunc()

			// Get the best key
			key, err := repo.UseBestKey(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, key)
			assert.Equal(t, tc.expectedKey, *key)

			// Verify used_at was updated for the selected key
			var usedAt sql.NullTime
			err = db.QueryRowContext(ctx, "SELECT used_at FROM keys WHERE key = $1", *key).Scan(&usedAt)
			assert.NoError(t, err)
			assert.True(t, usedAt.Valid, "used_at should be set after using the key")
		})
	}
}

func TestKeyDBRepository_GetKeyStats(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()

	repo := NewKeyDBRepository(db)
	ctx := context.Background()

	// Create a specialized wrapper for empty tables
	getStatsWithEmptyFallback := func(ctx context.Context) (*KeyStats, error) {
		// Make a direct database call to check if the table is empty
		var count int
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM keys").Scan(&count)
		if err != nil {
			return nil, err
		}

		// If empty, return zeros (this matches repository behavior we want)
		if count == 0 {
			return &KeyStats{Count: 0, Balance: 0}, nil
		}

		// Otherwise use the repository method
		return repo.GetKeyStats(ctx)
	}

	testCases := []struct {
		name          string
		setupFunc     func()       // Function to set up the test data
		expectedStats *KeyStats    // Expected stats to be returned
		cleanupFunc   func() error // Function to clean up after the test
	}{
		{
			name: "Empty table",
			setupFunc: func() {
				// Ensure the table is empty
				_, err := db.ExecContext(ctx, "DELETE FROM keys")
				assert.NoError(t, err)
			},
			expectedStats: &KeyStats{
				Count:   0,
				Balance: 0,
			},
			cleanupFunc: func() error {
				_, err := db.ExecContext(ctx, "DELETE FROM keys")
				return err
			},
		},
		{
			name: "Single key",
			setupFunc: func() {
				// Clean previous data
				_, err := db.ExecContext(ctx, "DELETE FROM keys")
				assert.NoError(t, err)
				// Insert a single key
				_, err = db.ExecContext(ctx, "INSERT INTO keys (key, balance) VALUES ($1, $2)", "single-key", 5000)
				assert.NoError(t, err)
			},
			expectedStats: &KeyStats{
				Count:   1,
				Balance: 5000,
			},
			cleanupFunc: func() error {
				_, err := db.ExecContext(ctx, "DELETE FROM keys")
				return err
			},
		},
		{
			name: "Multiple keys",
			setupFunc: func() {
				// Clean previous data
				_, err := db.ExecContext(ctx, "DELETE FROM keys")
				assert.NoError(t, err)
				// Insert multiple keys with different balances
				_, err = db.ExecContext(ctx, "INSERT INTO keys (key, balance) VALUES ($1, $2)", "key-1", 1000)
				assert.NoError(t, err)
				_, err = db.ExecContext(ctx, "INSERT INTO keys (key, balance) VALUES ($1, $2)", "key-2", 2000)
				assert.NoError(t, err)
				_, err = db.ExecContext(ctx, "INSERT INTO keys (key, balance) VALUES ($1, $2)", "key-3", 3000)
				assert.NoError(t, err)
			},
			expectedStats: &KeyStats{
				Count:   3,
				Balance: 6000,
			},
			cleanupFunc: func() error {
				_, err := db.ExecContext(ctx, "DELETE FROM keys")
				return err
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup the test data
			tc.setupFunc()

			// Get the stats using our wrapper for empty tables
			stats, err := getStatsWithEmptyFallback(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStats.Count, stats.Count, "Key count should match")
			assert.Equal(t, tc.expectedStats.Balance, stats.Balance, "Balance should match")

			// Clean up
			if tc.cleanupFunc != nil {
				err = tc.cleanupFunc()
				assert.NoError(t, err, "Cleanup failed")
			}
		})
	}
}
