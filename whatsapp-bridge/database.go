package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Driver      string
	URL         string
	IsRemote    bool
	MigrationsPath string
}

// DatabaseAdapter handles database connections and migrations
type DatabaseAdapter struct {
	config *DatabaseConfig
	logger waLog.Logger
}

// NewDatabaseAdapter creates a new database adapter
func NewDatabaseAdapter(logger waLog.Logger) *DatabaseAdapter {
	return &DatabaseAdapter{
		logger: logger,
	}
}

// GetDatabaseConfig determines the database configuration based on environment
func (da *DatabaseAdapter) GetDatabaseConfig() (*DatabaseConfig, error) {
	config := &DatabaseConfig{
		MigrationsPath: "supabase/migrations",
	}

	// Check for DATABASE_URL environment variable (Supabase/PostgreSQL)
	databaseURL := os.Getenv("DATABASE_URL")
	
	if databaseURL != "" {
		// Using remote PostgreSQL (Supabase)
		if strings.HasPrefix(databaseURL, "postgres://") || strings.HasPrefix(databaseURL, "postgresql://") {
			config.Driver = "postgres"
			config.URL = databaseURL
			config.IsRemote = true
			da.logger.Infof("Using remote PostgreSQL database (Supabase)")
		} else {
			return nil, fmt.Errorf("unsupported database URL format: %s", databaseURL)
		}
	} else {
		// Fallback to local SQLite
		config.Driver = "sqlite3"
		config.URL = "file:store/whatsapp.db?_foreign_keys=on"
		config.IsRemote = false
		config.MigrationsPath = "migrations" // Use local migrations for SQLite
		da.logger.Infof("DATABASE_URL not set, using local SQLite database")
		
		// Create directory for SQLite database if it doesn't exist
		if err := os.MkdirAll("store", 0755); err != nil {
			return nil, fmt.Errorf("failed to create store directory: %w", err)
		}
	}

	da.config = config
	return config, nil
}

// TestConnection tests the database connection
func (da *DatabaseAdapter) TestConnection(config *DatabaseConfig) error {
	db, err := sql.Open(config.Driver, config.URL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	// Test the connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	da.logger.Infof("Database connection test successful")
	return nil
}

// RunMigrations runs database migrations if needed
func (da *DatabaseAdapter) RunMigrations(config *DatabaseConfig) error {
	// Only run migrations for PostgreSQL (Supabase)
	// SQLite uses the built-in whatsmeow schema creation
	if !config.IsRemote {
		da.logger.Infof("Using SQLite - skipping custom migrations (whatsmeow will handle schema)")
		return nil
	}

	da.logger.Infof("Checking database migrations from %s...", config.MigrationsPath)
	
	// For Supabase, migrations are handled via Supabase CLI
	// We assume they are already applied
	da.logger.Infof("Database migrations assumed to be handled by Supabase CLI")
	return nil
}

// CreateSQLStore creates a whatsmeow SQL store with the configured database
func (da *DatabaseAdapter) CreateSQLStore(config *DatabaseConfig) (*sqlstore.Container, error) {
	dbLog := waLog.Stdout("Database", "INFO", true)
	
	container, err := sqlstore.New(context.Background(), config.Driver, config.URL, dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQL store: %w", err)
	}

	da.logger.Infof("SQL store created successfully using %s", config.Driver)
	return container, nil
}

// Initialize sets up the database connection with migrations and returns a SQL store
func (da *DatabaseAdapter) Initialize() (*sqlstore.Container, error) {
	// Get database configuration
	config, err := da.GetDatabaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get database config: %w", err)
	}

	// Test the connection first
	if err := da.TestConnection(config); err != nil {
		if config.IsRemote {
			// If remote connection fails, try to fallback to local SQLite
			da.logger.Warnf("Remote database connection failed: %v", err)
			da.logger.Infof("Falling back to local SQLite database...")
			
			// Reconfigure for SQLite
			config.Driver = "sqlite3"
			config.URL = "file:store/whatsapp.db?_foreign_keys=on"
			config.IsRemote = false
			config.MigrationsPath = "migrations"
			
			// Create directory for SQLite database if it doesn't exist
			if err := os.MkdirAll("store", 0755); err != nil {
				return nil, fmt.Errorf("failed to create store directory for fallback: %w", err)
			}
			
			// Test SQLite connection
			if err := da.TestConnection(config); err != nil {
				return nil, fmt.Errorf("both remote and local database connections failed: %w", err)
			}
		} else {
			return nil, err
		}
	}

	// Run migrations
	if err := da.RunMigrations(config); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create and return SQL store
	container, err := da.CreateSQLStore(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQL store: %w", err)
	}

	return container, nil
}

// GetConnectionInfo returns information about the current database connection
func (da *DatabaseAdapter) GetConnectionInfo() map[string]interface{} {
	if da.config == nil {
		return map[string]interface{}{
			"status": "not_initialized",
		}
	}

	info := map[string]interface{}{
		"driver":     da.config.Driver,
		"is_remote":  da.config.IsRemote,
		"migrations_path": da.config.MigrationsPath,
	}

	// Don't expose the full URL for security, just indicate the type
	if da.config.IsRemote {
		info["type"] = "supabase_postgresql"
	} else {
		info["type"] = "local_sqlite"
		info["path"] = "store/whatsapp.db"
	}

	return info
}
