package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CostRecord represents a single persisted cost entry for an API request.
type CostRecord struct {
	ID                  int64     `json:"id"`
	Provider            string    `json:"provider"`
	Model               string    `json:"model"`
	InputTokens         int       `json:"input_tokens"`
	OutputTokens        int       `json:"output_tokens"`
	CacheReadTokens     int       `json:"cache_read_tokens"`
	CacheCreationTokens int       `json:"cache_creation_tokens"`
	CostUSD             float64   `json:"cost_usd"`
	LatencyMs           int       `json:"latency_ms"`
	CreatedAt           time.Time `json:"created_at"`
}

// UsageStats holds aggregated usage statistics for a provider/model pair.
type UsageStats struct {
	Provider           string  `json:"provider"`
	Model              string  `json:"model"`
	RequestCount       int     `json:"request_count"`
	TotalInputTokens   int     `json:"total_input_tokens"`
	TotalOutputTokens  int     `json:"total_output_tokens"`
	TotalCacheRead     int     `json:"total_cache_read_tokens"`
	TotalCacheCreation int     `json:"total_cache_creation_tokens"`
	ErrorCount         int     `json:"error_count"`
	ErrorRate          float64 `json:"error_rate"`
	AvgLatencyMs       float64 `json:"avg_latency_ms"`
	TotalCostUSD       float64 `json:"total_cost_usd"`
}

// ProviderHealthRecord holds per-provider health data from the nodes table.
type ProviderHealthRecord struct {
	Provider      string    `json:"provider"`
	RequestCount  int       `json:"request_count"`
	ErrorCount    int       `json:"error_count"`
	ErrorRate     float64   `json:"error_rate"`
	AvgLatencyMs  float64   `json:"avg_latency_ms"`
	P50LatencyMs  float64   `json:"p50_latency_ms"`
	P95LatencyMs  float64   `json:"p95_latency_ms"`
	P99LatencyMs  float64   `json:"p99_latency_ms"`
	LastRequestAt time.Time `json:"last_request_at"`
}

// AnalyticsStore provides persistent cost tracking and usage analytics.
// It extends the base Store with analytics-specific tables and queries.
type AnalyticsStore interface {
	// RecordCost persists a cost record for a completed request.
	RecordCost(ctx context.Context, rec *CostRecord) error
	// GetUsageStats returns aggregated usage stats grouped by provider/model,
	// filtered to records created after the given time.
	GetUsageStats(ctx context.Context, since time.Time) ([]UsageStats, error)
	// GetCostSummary returns total cost grouped by provider/model since the given time.
	GetCostSummary(ctx context.Context, since time.Time) ([]UsageStats, error)
	// GetProviderHealth returns health metrics derived from actual request data.
	GetProviderHealth(ctx context.Context, since time.Time) ([]ProviderHealthRecord, error)
}

// SQLiteAnalyticsStore implements AnalyticsStore backed by SQLite.
type SQLiteAnalyticsStore struct {
	db *sql.DB
}

// NewSQLiteAnalyticsStore wraps an existing SQLiteStore's database connection
// for analytics operations. The caller must ensure the underlying DB has been
// migrated (i.e. the SQLiteStore was opened via Open).
func NewSQLiteAnalyticsStore(s *SQLiteStore) *SQLiteAnalyticsStore {
	return &SQLiteAnalyticsStore{db: s.db}
}

// NewSQLiteAnalyticsStoreFromDB creates an analytics store from a raw DB handle.
func NewSQLiteAnalyticsStoreFromDB(db *sql.DB) *SQLiteAnalyticsStore {
	return &SQLiteAnalyticsStore{db: db}
}

// MigrateAnalytics creates the analytics-specific tables. It is safe to call
// multiple times (uses IF NOT EXISTS).
func (a *SQLiteAnalyticsStore) MigrateAnalytics() error {
	_, err := a.db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS cost_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			cache_read_tokens INTEGER DEFAULT 0,
			cache_creation_tokens INTEGER DEFAULT 0,
			cost_usd REAL DEFAULT 0,
			latency_ms INTEGER DEFAULT 0,
			created_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_cost_records_provider ON cost_records(provider, model);
		CREATE INDEX IF NOT EXISTS idx_cost_records_created ON cost_records(created_at);
	`)
	return err
}

// RecordCost persists a cost record.
func (a *SQLiteAnalyticsStore) RecordCost(ctx context.Context, rec *CostRecord) error {
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	_, err := a.db.ExecContext(
		ctx,
		`INSERT INTO cost_records (provider, model, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, cost_usd, latency_ms, created_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		rec.Provider, rec.Model, rec.InputTokens, rec.OutputTokens,
		rec.CacheReadTokens, rec.CacheCreationTokens, rec.CostUSD,
		rec.LatencyMs, rec.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("analytics: record cost: %w", err)
	}
	return nil
}

// GetUsageStats returns aggregated usage statistics from the nodes table,
// grouped by provider and model, for nodes created after 'since'.
func (a *SQLiteAnalyticsStore) GetUsageStats(ctx context.Context, since time.Time) ([]UsageStats, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT
			provider,
			model,
			COUNT(*) AS request_count,
			COALESCE(SUM(tokens_in), 0) AS total_input,
			COALESCE(SUM(tokens_out), 0) AS total_output,
			COALESCE(SUM(tokens_cache_read), 0) AS total_cache_read,
			COALESCE(SUM(tokens_cache_creation), 0) AS total_cache_creation,
			COALESCE(SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END), 0) AS error_count,
			COALESCE(AVG(latency_ms), 0) AS avg_latency
		FROM nodes
		WHERE created_at >= ? AND provider != ''
		GROUP BY provider, model
		ORDER BY request_count DESC
	`, since.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("analytics: usage stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var stats []UsageStats
	for rows.Next() {
		var s UsageStats
		if err := rows.Scan(&s.Provider, &s.Model, &s.RequestCount,
			&s.TotalInputTokens, &s.TotalOutputTokens,
			&s.TotalCacheRead, &s.TotalCacheCreation,
			&s.ErrorCount, &s.AvgLatencyMs); err != nil {
			return nil, fmt.Errorf("analytics: scan usage: %w", err)
		}
		if s.RequestCount > 0 {
			s.ErrorRate = float64(s.ErrorCount) / float64(s.RequestCount)
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetCostSummary returns total cost grouped by provider/model from cost_records.
func (a *SQLiteAnalyticsStore) GetCostSummary(ctx context.Context, since time.Time) ([]UsageStats, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT
			provider,
			model,
			COUNT(*) AS request_count,
			COALESCE(SUM(input_tokens), 0) AS total_input,
			COALESCE(SUM(output_tokens), 0) AS total_output,
			COALESCE(SUM(cache_read_tokens), 0) AS total_cache_read,
			COALESCE(SUM(cache_creation_tokens), 0) AS total_cache_creation,
			COALESCE(SUM(cost_usd), 0) AS total_cost,
			COALESCE(AVG(latency_ms), 0) AS avg_latency
		FROM cost_records
		WHERE created_at >= ?
		GROUP BY provider, model
		ORDER BY total_cost DESC
	`, since.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("analytics: cost summary: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var stats []UsageStats
	for rows.Next() {
		var s UsageStats
		if err := rows.Scan(&s.Provider, &s.Model, &s.RequestCount,
			&s.TotalInputTokens, &s.TotalOutputTokens,
			&s.TotalCacheRead, &s.TotalCacheCreation,
			&s.TotalCostUSD, &s.AvgLatencyMs); err != nil {
			return nil, fmt.Errorf("analytics: scan cost: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetProviderHealth returns health metrics derived from actual request data in
// the nodes table, computed over the given time window.
func (a *SQLiteAnalyticsStore) GetProviderHealth(ctx context.Context, since time.Time) ([]ProviderHealthRecord, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT
			provider,
			COUNT(*) AS request_count,
			COALESCE(SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END), 0) AS error_count,
			COALESCE(AVG(latency_ms), 0) AS avg_latency,
			MAX(created_at) AS last_request
		FROM nodes
		WHERE created_at >= ? AND provider != '' AND node_type = 'assistant'
		GROUP BY provider
		ORDER BY request_count DESC
	`, since.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("analytics: provider health: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []ProviderHealthRecord
	for rows.Next() {
		var r ProviderHealthRecord
		var lastReq string
		if err := rows.Scan(&r.Provider, &r.RequestCount, &r.ErrorCount,
			&r.AvgLatencyMs, &lastReq); err != nil {
			return nil, fmt.Errorf("analytics: scan health: %w", err)
		}
		if r.RequestCount > 0 {
			r.ErrorRate = float64(r.ErrorCount) / float64(r.RequestCount)
		}
		r.LastRequestAt, _ = time.Parse(time.RFC3339Nano, lastReq)
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Compute latency percentiles per provider.
	for i := range records {
		r := &records[i]
		p50, p95, p99, err := a.latencyPercentiles(ctx, r.Provider, since)
		if err != nil {
			continue // non-fatal
		}
		r.P50LatencyMs = p50
		r.P95LatencyMs = p95
		r.P99LatencyMs = p99
	}

	return records, nil
}

// latencyPercentiles returns P50, P95, P99 latency for a provider from nodes.
func (a *SQLiteAnalyticsStore) latencyPercentiles(ctx context.Context, provider string, since time.Time) (p50, p95, p99 float64, err error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT latency_ms FROM nodes
		WHERE provider = ? AND created_at >= ? AND node_type = 'assistant' AND latency_ms > 0
		ORDER BY latency_ms ASC
	`, provider, since.Format(time.RFC3339Nano))
	if err != nil {
		return 0, 0, 0, err
	}
	defer func() { _ = rows.Close() }()

	var latencies []float64
	for rows.Next() {
		var l float64
		if err := rows.Scan(&l); err != nil {
			return 0, 0, 0, err
		}
		latencies = append(latencies, l)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, 0, err
	}

	n := len(latencies)
	if n == 0 {
		return 0, 0, 0, nil
	}

	p50 = latencies[int(float64(n-1)*0.50)]
	p95 = latencies[int(float64(n-1)*0.95)]
	p99 = latencies[int(float64(n-1)*0.99)]
	return p50, p95, p99, nil
}
