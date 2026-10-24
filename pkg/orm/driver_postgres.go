package orm

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// PostgresConfig contains connection parameters for PostgreSQL engines.
type PostgresConfig struct {
	Host            string
	Port            int // default: 5432
	Database        string
	User            string
	Password        string
	SSLMode         string // default: "disable" (options: "require", "verify-ca", "verify-full")
	TimeZone        string // default: "UTC"
	DriverName      string // default: "postgres" (or "pgx")
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// FormatDSN generates a standard PostgreSQL URL connection string.
func (c PostgresConfig) FormatDSN() string {
	host := c.Host
	if host == "" {
		host = "localhost"
	}

	port := c.Port
	if port <= 0 {
		port = 5432
	}

	user := c.User
	if user == "" {
		user = "postgres"
	}

	dbName := c.Database
	if dbName == "" {
		dbName = "postgres"
	}

	sslMode := strings.ToLower(strings.TrimSpace(c.SSLMode))
	if sslMode == "" {
		sslMode = "disable"
	}

	tz := c.TimeZone
	if tz == "" {
		tz = "UTC"
	}

	var userPass string
	if c.Password != "" {
		userPass = fmt.Sprintf("%s:%s@", url.QueryEscape(user), url.QueryEscape(c.Password))
	} else {
		userPass = fmt.Sprintf("%s@", url.QueryEscape(user))
	}

	return fmt.Sprintf(
		"postgres://%s%s:%d/%s?sslmode=%s&timezone=%s",
		userPass,
		host,
		port,
		dbName,
		sslMode,
		url.QueryEscape(tz),
	)
}

// NewPostgres creates and opens a DB connection pool configured for PostgreSQL.
func NewPostgres(cfg PostgresConfig) (*DB, error) {
	driver := strings.TrimSpace(cfg.DriverName)
	if driver == "" {
		driver = "postgres"
	}

	return OpenWithConfig(Config{
		DriverName:      driver,
		DSN:             cfg.FormatDSN(),
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
	})
}
