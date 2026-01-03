package storage

import (
	"context"
	"fmt"

	"proxypool/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(dbURL string) (*PostgresRepository, error) {
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}
	
	// Fix for Supabase Transaction Pooler (PgBouncer) "prepared statement already exists" error
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	return &PostgresRepository{pool: pool}, nil
}

func (r *PostgresRepository) Close() {
	r.pool.Close()
}

// SaveBatch inserts new proxies. Duplicates (ip, port) are ignored.
func (r *PostgresRepository) SaveBatch(ctx context.Context, proxies []*model.Proxy) error {
	if len(proxies) == 0 {
		return nil
	}

	// We use COPY for high performance, but COPY doesn't support ON CONFLICT natively in a simple way 
	// without using a temp table. 
	// For "World Scale", thousands of proxies, INSERT ... ON CONFLICT is safer and "fast enough" if batched correctly.
	// But building a huge generic INSERT string is messy. 
	// Let's use pgx.Batch.

	batch := &pgx.Batch{}
	for _, p := range proxies {
		batch.Queue(`
			INSERT INTO proxies (ip, port, protocol, created_at)
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT (ip, port) DO NOTHING
		`, p.IP, p.Port, p.Protocol)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	// We must execute the batch results to ensure it actually happened and check errors
	for i := 0; i < len(proxies); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to insert batch item %d: %w", i, err)
		}
	}

	return nil
}

// GetProxiesToCheck returns proxies that haven't been checked recently.
// Uses FOR UPDATE SKIP LOCKED to allow multiple concurrent consumers.
func (r *PostgresRepository) GetProxiesToCheck(ctx context.Context, limit int) ([]*model.Proxy, error) {
	query := `
		SELECT id, ip::TEXT, port, COALESCE(protocol, ''), COALESCE(country, ''), COALESCE(anonymity, ''), COALESCE(latency_ms, 0), last_checked_at, created_at
		FROM proxies
		ORDER BY last_checked_at ASC NULLS FIRST
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var result []*model.Proxy
	for rows.Next() {
		p := &model.Proxy{}
		var ipStr string
		err := rows.Scan(
			&p.ID,
			&ipStr,
			&p.Port,
			&p.Protocol,
			&p.Country,
			&p.Anonymity,
			&p.LatencyMS,
			&p.LastCheckedAt,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		p.IP = ipStr
		result = append(result, p)
	}
	return result, nil
}

// Update updates a single proxy's status.
func (r *PostgresRepository) Update(ctx context.Context, p *model.Proxy) error {
	query := `
		UPDATE proxies 
		SET latency_ms = $1, last_checked_at = $2, country = $3
		WHERE id = $4
	`
	_, err := r.pool.Exec(ctx, query, p.LatencyMS, p.LastCheckedAt, p.Country, p.ID)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	return nil
}

// UpdateBatch updates multiple proxies efficiently using a batch.
func (r *PostgresRepository) UpdateBatch(ctx context.Context, proxies []*model.Proxy) error {
	batch := &pgx.Batch{}
	for _, p := range proxies {
		batch.Queue(`
			UPDATE proxies 
			SET latency_ms = $1, last_checked_at = $2, country = $3, protocol = $4
			WHERE id = $5
		`, p.LatencyMS, p.LastCheckedAt, p.Country, p.Protocol, p.ID)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(proxies); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to update batch item %d: %w", i, err)
		}
	}
	return nil
}

// Count returns the total number of proxies.
func (r *PostgresRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM proxies").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
