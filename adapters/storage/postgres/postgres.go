package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	mpg "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/models"
	"go.uber.org/zap"

	"github.com/cockroachdb/errors"
)

const (
	driverName             = "postgres"
	defaultDatabase        = "postgres"
	connectionTimeout      = 10 * time.Second
	defaultMaxOpenConns    = 50
	defaultMaxIdleConns    = 10
	defaultConnMaxLifetime = 5 * time.Minute
)

func InitDB(cfg *models.Config, logger *zap.Logger) (*sql.DB, error) {
	dsn := cfg.Database.DSN

	if dsn == "" {
		dsn = cfg.Database.URL
	}

	if dsn == "" {
		return nil, errs.ErrDatabaseDSNRequired
	}

	if err := ensureDatabaseExists(dsn, logger); err != nil {
		return nil, errs.Wrapf(errs.ErrEnsureDatabaseExists, err)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, errs.Wrapf(errs.ErrOpenPostgresConnection, err)
	}

	setConnectionPoolConfig(db, cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, errs.Wrapf(errs.ErrPingPostgres, err)
	}

	logger.Info("postgres connection established")
	if cfg.Database.MigrationPath != "" {
		if err := RunMigrations(db, cfg.Database.MigrationPath, logger); err != nil {
			_ = db.Close()
			return nil, errs.Wrapf(errs.ErrRunMigrations, err)
		}
	}
	return db, nil
}

func BuildLogger(env string) (*zap.Logger, error) {
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

func RunMigrations(db *sql.DB, migrationPath string, logger *zap.Logger) error {
	logger.Info("running database migrations")

	driver, err := mpg.WithInstance(db, &mpg.Config{})
	if err != nil {
		return errs.Wrapf(errs.ErrCreateMigrationDriver, err)
	}

	m, err := migrate.NewWithDatabaseInstance(fmt.Sprintf("file://%s", migrationPath), "postgres", driver)
	if err != nil {
		return errs.Wrapf(errs.ErrCreateMigrationInstance, err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return errs.Wrapf(errs.ErrRunMigrations, err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		logger.Info("migrations: no changes")
	} else {
		logger.Info("migrations: completed successfully")
	}

	return nil
}

func setConnectionPoolConfig(db *sql.DB, cfg *models.Config, logger *zap.Logger) {
	maxOpen := cfg.Database.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = defaultMaxOpenConns
	}

	maxIdle := cfg.Database.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = defaultMaxIdleConns
	}
	if maxIdle > maxOpen {
		logger.Warn("max_idle_conns exceeds max_open_conns, clamping",
			zap.Int("max_idle", maxIdle), zap.Int("max_open", maxOpen))
		maxIdle = maxOpen
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(defaultConnMaxLifetime)
	db.SetConnMaxIdleTime(5 * time.Minute)

	logger.Info(
		"postgres pool configured",
		zap.Int("max_open", maxOpen),
		zap.Int("max_idle", maxIdle),
		zap.Duration("conn_max_lifetime", defaultConnMaxLifetime),
	)
}

func ensureDatabaseExists(dsn string, logger *zap.Logger) error {
	dbName, err := databaseNameFromDSN(dsn)
	if err != nil {
		return errs.Wrapf(errs.ErrParseDatabaseName, err)
	}

	if dbName == "" {
		return errs.ErrDatabaseNameEmpty
	}

	defaultDSN, err := replaceDatabaseInDSN(dsn, defaultDatabase)
	if err != nil {
		return errs.Wrapf(errs.ErrCreateDefaultDatabaseDSN, err)
	}

	db, err := sql.Open(driverName, defaultDSN)
	if err != nil {
		return errs.Wrapf(errs.ErrConnectDefaultPostgres, err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return errs.Wrapf(errs.ErrPingDefaultPostgres, err)
	}

	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
	err = db.QueryRowContext(ctx, query, dbName).Scan(&exists)
	if err != nil {
		return errs.Wrapf(errs.ErrCheckDatabaseExists, err)
	}

	if exists {
		log.Printf("database '%s' already exists\n", dbName)
		return nil
	}

	log.Printf("database '%s' does not exist, creating it...\n", dbName)

	createQuery := fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(dbName))
	_, err = db.ExecContext(ctx, createQuery)
	if err != nil {
		return errs.Wrapf(errs.ErrCreateDatabase, err, dbName)
	}

	log.Printf("database '%s' created successfully\n", dbName)
	return nil
}

func databaseNameFromDSN(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", errs.Wrapf(errs.ErrParseDatabaseName, err)
	}

	databaseName := strings.TrimPrefix(u.Path, "/")

	if databaseName == "" {
		return "", errs.ErrDatabaseNameEmpty
	}

	return databaseName, nil
}

func replaceDatabaseInDSN(dsn, newDB string) (string, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "", err
		}
		u.Path = "/" + newDB
		return u.String(), nil
	}

	parts := strings.Fields(dsn)
	var result []string
	dbNameReplaced := false

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && kv[0] == "dbname" {
			result = append(result, fmt.Sprintf("dbname=%s", newDB))
			dbNameReplaced = true
		} else {
			result = append(result, part)
		}
	}

	if !dbNameReplaced {
		result = append(result, fmt.Sprintf("dbname=%s", newDB))
	}

	return strings.Join(result, " "), nil
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
