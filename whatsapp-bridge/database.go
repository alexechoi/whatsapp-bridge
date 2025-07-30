package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// init function runs before main() and loads environment variables
func init() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// Don't fail if .env file doesn't exist, just log it
		log.Printf("No .env file found or error loading it: %v", err)
	}
}

// DatabaseAdapter handles connections to either PostgreSQL or SQLite
type DatabaseAdapter struct {
	db     *sql.DB
	dbURL  string
	logger waLog.Logger
}

// NewDatabaseAdapter creates a new database adapter
func NewDatabaseAdapter(logger waLog.Logger) *DatabaseAdapter {
	return &DatabaseAdapter{
		logger: logger,
	}
}

// Initialize sets up the database connection
func (a *DatabaseAdapter) Initialize() (*sqlstore.Container, error) {
	// Try to connect to PostgreSQL first
	container, err := a.connectPostgreSQL()
	if err != nil {
		a.logger.Warnf("Failed to connect to PostgreSQL: %v", err)
		a.logger.Infof("Falling back to SQLite")
		
		// Fall back to SQLite
		return a.connectSQLite()
	}
	
	return container, nil
}

// connectPostgreSQL attempts to connect to PostgreSQL using environment variables
func (a *DatabaseAdapter) connectPostgreSQL() (*sqlstore.Container, error) {
	// Get database URL from environment variable
	dbURL := os.Getenv("DATABASE_URL")
	
	// Store the connection URL
	a.dbURL = dbURL
	
	// Test the connection
	err := a.TestConnection()
	if err != nil {
		return nil, fmt.Errorf("PostgreSQL connection test failed: %v", err)
	}
	
	// Connect to the database
	a.logger.Infof("Connecting to PostgreSQL at %s", sanitizeConnectionURL(dbURL))
	
	// Open a direct connection to the database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	
	// Check if we need to add the facebook_uuid column
	err = a.checkAndUpdateSchema(db)
	if err != nil {
		a.logger.Warnf("Failed to update schema: %v", err)
		// Continue anyway, as this is not critical
	}
	
	// Create a custom container with the database
	container := sqlstore.NewWithDB(db, "postgres", a.logger)
	
	// Skip the upgrade since tables already exist
	a.logger.Infof("Tables already exist, skipping upgrade")
	
	return container, nil
}

// checkAndUpdateSchema checks if the database schema needs updates and applies them
func (a *DatabaseAdapter) checkAndUpdateSchema(db *sql.DB) error {
	// Check if facebook_uuid column exists in the whatsmeow_device table
	var columnExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_name = 'whatsmeow_device' 
			AND column_name = 'facebook_uuid'
		)
	`).Scan(&columnExists)
	
	if err != nil {
		return fmt.Errorf("failed to check if facebook_uuid column exists: %v", err)
	}
	
	// If the column doesn't exist, add it
	if !columnExists {
		a.logger.Infof("Adding facebook_uuid column to whatsmeow_device table")
		
		_, err := db.Exec(`
			ALTER TABLE whatsmeow_device 
			ADD COLUMN facebook_uuid TEXT
		`)
		
		if err != nil {
			return fmt.Errorf("failed to add facebook_uuid column: %v", err)
		}
		
		a.logger.Infof("Successfully added facebook_uuid column")
	}
	
	// Check if lid_migration_ts column exists in the whatsmeow_device table
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_name = 'whatsmeow_device' 
			AND column_name = 'lid_migration_ts'
		)
	`).Scan(&columnExists)
	
	if err != nil {
		return fmt.Errorf("failed to check if lid_migration_ts column exists: %v", err)
	}
	
	// If the column doesn't exist, add it
	if !columnExists {
		a.logger.Infof("Adding lid_migration_ts column to whatsmeow_device table")
		
		_, err := db.Exec(`
			ALTER TABLE whatsmeow_device 
			ADD COLUMN lid_migration_ts BIGINT DEFAULT 0
		`)
		
		if err != nil {
			return fmt.Errorf("failed to add lid_migration_ts column: %v", err)
		}
		
		a.logger.Infof("Successfully added lid_migration_ts column")
	}
	
	return nil
}

// connectSQLite creates a SQLite connection as fallback
func (a *DatabaseAdapter) connectSQLite() (*sqlstore.Container, error) {
	// Create directory for SQLite database if it doesn't exist
	if err := os.MkdirAll("store", 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %v", err)
	}
	
	// Connect to SQLite database
	a.logger.Infof("Connecting to SQLite database")
	
	// Create a new container with the SQLite connection
	container, err := sqlstore.New(context.Background(), "sqlite3", "file:store/whatsmeow.db?_foreign_keys=on", a.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQLite database container: %v", err)
	}
	
	// Reset the PostgreSQL URL since we're using SQLite
	a.dbURL = ""
	
	return container, nil
}

// TestConnection tests the PostgreSQL connection
func (a *DatabaseAdapter) TestConnection() error {
	if a.dbURL == "" {
		return fmt.Errorf("database URL is not set")
	}
	
	db, err := sql.Open("postgres", a.dbURL)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()
	
	// Set connection parameters
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	
	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}
	
	// Check if whatsmeow tables exist
	var tableCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'whatsmeow_device'").Scan(&tableCount)
	if err != nil {
		return fmt.Errorf("failed to check for whatsmeow tables: %v", err)
	}
	
	if tableCount == 0 {
		return fmt.Errorf("whatsmeow tables not found in database, please run migrations first")
	}
	
	return nil
}

// GetConnectionInfo returns information about the current database connection
func (a *DatabaseAdapter) GetConnectionInfo() map[string]string {
	info := make(map[string]string)
	
	if a.dbURL != "" {
		// PostgreSQL connection
		info["type"] = "PostgreSQL"
		info["url"] = sanitizeConnectionURL(a.dbURL)
		
		// Add parsed details
		info["host"] = extractHost(a.dbURL)
		info["port"] = extractPort(a.dbURL)
		info["user"] = extractUser(a.dbURL)
		info["database"] = extractDatabase(a.dbURL)
	} else {
		// SQLite connection
		info["type"] = "SQLite"
		info["path"] = "store/whatsmeow.db"
	}
	
	return info
}

// sanitizeConnectionURL removes sensitive information from a connection URL
func sanitizeConnectionURL(url string) string {
	// Hide password
	parts := strings.Split(url, "@")
	if len(parts) < 2 {
		return url
	}
	
	credParts := strings.Split(parts[0], ":")
	if len(credParts) < 3 {
		return url
	}
	
	// Replace password with asterisks
	maskedURL := fmt.Sprintf("%s:***@%s", credParts[0], parts[1])
	return maskedURL
}

// Helper functions to extract connection details from DATABASE_URL

// extractHost extracts the host from a connection string
func extractHost(connStr string) string {
	// Remove protocol prefix
	connStr = strings.TrimPrefix(connStr, "postgresql://")
	
	// Split by @ to get the server part
	parts := strings.Split(connStr, "@")
	if len(parts) < 2 {
		return "localhost"
	}
	
	// Get the host:port part
	serverParts := strings.Split(parts[1], ":")
	if len(serverParts) == 0 {
		return "localhost"
	}
	
	// Extract host
	hostPart := serverParts[0]
	
	// If there's a / in the host, take only what's before it
	if strings.Contains(hostPart, "/") {
		hostPart = strings.Split(hostPart, "/")[0]
	}
	
	return hostPart
}

// extractPort extracts the port from a connection string
func extractPort(connStr string) string {
	// Remove protocol prefix
	connStr = strings.TrimPrefix(connStr, "postgresql://")
	
	// Split by @ to get the server part
	parts := strings.Split(connStr, "@")
	if len(parts) < 2 {
		return "5432" // Default PostgreSQL port
	}
	
	// Get the host:port part
	serverParts := strings.Split(parts[1], ":")
	if len(serverParts) < 2 {
		return "5432" // Default PostgreSQL port
	}
	
	// Extract port
	portPart := serverParts[1]
	
	// If there's a / in the port, take only what's before it
	if strings.Contains(portPart, "/") {
		portPart = strings.Split(portPart, "/")[0]
	}
	
	return portPart
}

// extractUser extracts the username from a connection string
func extractUser(connStr string) string {
	// Remove protocol prefix
	connStr = strings.TrimPrefix(connStr, "postgresql://")
	
	// Split by @ to get the credentials part
	parts := strings.Split(connStr, "@")
	if len(parts) == 0 {
		return "postgres"
	}
	
	// Get the username:password part
	credParts := strings.Split(parts[0], ":")
	if len(credParts) == 0 {
		return "postgres"
	}
	
	return credParts[0]
}

// extractPassword extracts the password from a connection string
func extractPassword(connStr string) string {
	// Remove protocol prefix
	connStr = strings.TrimPrefix(connStr, "postgresql://")
	
	// Split by @ to get the credentials part
	parts := strings.Split(connStr, "@")
	if len(parts) == 0 {
		return ""
	}
	
	// Get the username:password part
	credParts := strings.Split(parts[0], ":")
	if len(credParts) < 2 {
		return ""
	}
	
	return credParts[1]
}

// extractDatabase extracts the database name from a connection string
func extractDatabase(connStr string) string {
	// Look for the database name at the end of the string
	parts := strings.Split(connStr, "/")
	if len(parts) < 2 {
		return "postgres" // Default database name
	}
	
	dbName := parts[len(parts)-1]
	
	// Remove any query parameters
	if strings.Contains(dbName, "?") {
		dbName = strings.Split(dbName, "?")[0]
	}
	
	return dbName
}

// GetDB returns a database connection
func (a *DatabaseAdapter) GetDB() (*sql.DB, error) {
	if a.dbURL == "" {
		return nil, fmt.Errorf("database URL is not set")
	}
	
	db, err := sql.Open("postgres", a.dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	
	return db, nil
}

// QRWebServer provides a simple web server to display QR codes
type QRWebServer struct {
	currentQR   string
	isConnected bool
}

// NewQRWebServer creates a new QR web server
func NewQRWebServer() *QRWebServer {
	return &QRWebServer{
		currentQR:   "",
		isConnected: false,
	}
}

// UpdateQRCode updates the current QR code
func (s *QRWebServer) UpdateQRCode(qrCode string) {
	s.currentQR = qrCode
	s.isConnected = false
}

// SetConnected marks the connection as established
func (s *QRWebServer) SetConnected() {
	s.isConnected = true
	s.currentQR = ""
}

// StartQRWebServer starts a web server to display QR codes
func (s *QRWebServer) StartQRWebServer(port int) {
	// Create a simple HTTP server to display the QR code
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		
		if s.isConnected {
			// Show connected message
			fmt.Fprintf(w, `
				<!DOCTYPE html>
				<html>
				<head>
					<title>WhatsApp Connection</title>
					<meta name="viewport" content="width=device-width, initial-scale=1">
					<style>
						body { font-family: Arial, sans-serif; text-align: center; margin-top: 50px; }
						.success { color: green; font-size: 24px; }
					</style>
				</head>
				<body>
					<h1>WhatsApp Connection</h1>
					<p class="success">✓ Connected to WhatsApp!</p>
					<p>You can close this window now.</p>
				</body>
				</html>
			`)
		} else if s.currentQR != "" {
			// Show QR code
			fmt.Fprintf(w, `
				<!DOCTYPE html>
				<html>
				<head>
					<title>WhatsApp QR Code</title>
					<meta name="viewport" content="width=device-width, initial-scale=1">
					<script src="https://cdn.jsdelivr.net/npm/qrcode-generator@1.4.4/qrcode.min.js"></script>
					<style>
						body { font-family: Arial, sans-serif; text-align: center; margin-top: 50px; }
						#qrcode { display: inline-block; margin: 20px; }
						.instructions { max-width: 500px; margin: 20px auto; text-align: left; }
					</style>
				</head>
				<body>
					<h1>WhatsApp QR Code</h1>
					<div id="qrcode"></div>
					<div class="instructions">
						<h3>Instructions:</h3>
						<ol>
							<li>Open WhatsApp on your phone</li>
							<li>Tap Menu or Settings and select WhatsApp Web</li>
							<li>Scan the QR code with your phone</li>
							<li>This page will automatically update when connected</li>
						</ol>
					</div>
					<p><small>This page will automatically refresh every 30 seconds</small></p>
					<script>
						// Generate QR code
						var qr = qrcode(0, 'L');
						qr.addData('%s');
						qr.make();
						document.getElementById('qrcode').innerHTML = qr.createImgTag(8);
						
						// Refresh page every 30 seconds to check for new QR code
						setTimeout(function() { window.location.reload(); }, 30000);
					</script>
				</body>
				</html>
			`, s.currentQR)
		} else {
			// Show waiting message
			fmt.Fprintf(w, `
				<!DOCTYPE html>
				<html>
				<head>
					<title>WhatsApp Connection</title>
					<meta name="viewport" content="width=device-width, initial-scale=1">
					<style>
						body { font-family: Arial, sans-serif; text-align: center; margin-top: 50px; }
						.waiting { color: orange; }
					</style>
					<meta http-equiv="refresh" content="5">
				</head>
				<body>
					<h1>WhatsApp Connection</h1>
					<p class="waiting">Waiting for QR code...</p>
					<p>This page will refresh automatically.</p>
				</body>
				</html>
			`)
		}
	})
	
	// Start the server in a goroutine
	go func() {
		addr := fmt.Sprintf(":%d", port)
		fmt.Printf("Starting QR web server on http://localhost%s\n", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Printf("QR web server error: %v\n", err)
		}
	}()
} 