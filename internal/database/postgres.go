package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// Postgres represents a PostgreSQL database connection
type Postgres struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewPostgres creates a new PostgreSQL connection
func NewPostgres(host string, port int, database, user, password, sslMode string,
	maxOpenConns, maxIdleConns int, connMaxLifetime, connMaxIdleTime time.Duration,
	logger *zap.Logger) (*Postgres, error) {

	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		host, port, database, user, password, sslMode)

	// If password is empty, don't include it in DSN
	if password == "" {
		dsn = fmt.Sprintf("host=%s port=%d dbname=%s user=%s sslmode=%s",
			host, port, database, user, sslMode)
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connected successfully",
		zap.String("host", host),
		zap.Int("port", port),
		zap.String("database", database))

	return &Postgres{
		db:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection
func (p *Postgres) Close() error {
	return p.db.Close()
}

// GetDB returns the underlying sqlx.DB
func (p *Postgres) GetDB() *sqlx.DB {
	return p.db
}

// GetTx returns a new transaction
func (p *Postgres) GetTx(ctx context.Context) (*sqlx.Tx, error) {
	return p.db.BeginTxx(ctx, nil)
}

// Health checks the database health
func (p *Postgres) Health(ctx context.Context) error {
	return p.db.PingContext(ctx)
}
