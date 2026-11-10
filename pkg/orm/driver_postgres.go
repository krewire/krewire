package orm

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultPostgresHost is the fallback host for PostgreSQL.
	DefaultPostgresHost = "localhost"

	// DefaultPostgresPort is the standard PostgreSQL port.
	DefaultPostgresPort = 5432

	// DefaultPostgresUser is the default superuser name.
	DefaultPostgresUser = "postgres"

	// DefaultPostgresDatabase is the default maintenance database name.
	DefaultPostgresDatabase = "postgres"

	// DefaultPostgresSSLMode is the default SSL connection mode.
	DefaultPostgresSSLMode = "disable"

	// DefaultPostgresTimeZone is the default session time zone.
	DefaultPostgresTimeZone = "UTC"

	// DefaultPostgresDriverName is the default SQL driver name.
	DefaultPostgresDriverName = "pgx"
)

// PostgresConfig contains connection parameters for PostgreSQL engines.
type PostgresConfig struct {
	Host                            string
	Port                            int // default: DefaultPostgresPort (5432)
	Database                        string
	User                            string
	Password                        string
	SSLMode                         string        // default: DefaultPostgresSSLMode ("disable")
	TimeZone                        string        // default: DefaultPostgresTimeZone ("UTC")
	DriverName                      string        // default: DefaultPostgresDriverName ("pgx")
	SearchPath                      string        // default: "public"
	ApplicationName                 string        // default: "krewire"
	SSLRootCert                     string        // Path to SSL root certificate authority file
	SSLCert                         string        // Path to SSL client certificate file
	SSLKey                          string        // Path to SSL client private key file
	ConnectTimeout                  time.Duration // Dial timeout
	StatementTimeout                time.Duration // Query execution timeout before cancellation
	IdleInTransactionSessionTimeout time.Duration // Transaction idle leak timeout
	MaxOpenConns                    int
	MaxIdleConns                    int
	ConnMaxLifetime                 time.Duration
	ConnMaxIdleTime                 time.Duration
}

// FormatDSN generates a standard PostgreSQL URL connection string.
func (c PostgresConfig) FormatDSN() string {
	host := c.Host
	if host == "" {
		host = DefaultPostgresHost
	}

	port := c.Port
	if port <= 0 {
		port = DefaultPostgresPort
	}

	user := c.User
	if user == "" {
		user = DefaultPostgresUser
	}

	dbName := c.Database
	if dbName == "" {
		dbName = DefaultPostgresDatabase
	}

	sslMode := strings.ToLower(strings.TrimSpace(c.SSLMode))
	if sslMode == "" {
		sslMode = DefaultPostgresSSLMode
	}

	tz := c.TimeZone
	if tz == "" {
		tz = DefaultPostgresTimeZone
	}

	params := url.Values{}
	params.Set("sslmode", sslMode)
	params.Set("timezone", tz)

	if c.SearchPath != "" {
		params.Set("search_path", c.SearchPath)
	}
	if c.ApplicationName != "" {
		params.Set("application_name", c.ApplicationName)
	}
	if c.SSLRootCert != "" {
		params.Set("sslrootcert", c.SSLRootCert)
	}
	if c.SSLCert != "" {
		params.Set("sslcert", c.SSLCert)
	}
	if c.SSLKey != "" {
		params.Set("sslkey", c.SSLKey)
	}

	if c.ConnectTimeout > 0 {
		timeoutSec := int(c.ConnectTimeout.Seconds())
		if timeoutSec < 1 {
			timeoutSec = 1
		}
		params.Set("connect_timeout", fmt.Sprintf("%d", timeoutSec))
	}
	if c.StatementTimeout > 0 {
		params.Set("statement_timeout", fmt.Sprintf("%d", c.StatementTimeout.Milliseconds()))
	}
	if c.IdleInTransactionSessionTimeout > 0 {
		params.Set("idle_in_transaction_session_timeout", fmt.Sprintf("%d", c.IdleInTransactionSessionTimeout.Milliseconds()))
	}

	var userPass string
	if c.Password != "" {
		userPass = fmt.Sprintf("%s:%s@", url.QueryEscape(user), url.QueryEscape(c.Password))
	} else {
		userPass = fmt.Sprintf("%s@", url.QueryEscape(user))
	}

	return fmt.Sprintf(
		"postgres://%s%s:%d/%s?%s",
		userPass,
		host,
		port,
		dbName,
		params.Encode(),
	)
}

// NewPostgres creates and opens a DB connection pool configured for PostgreSQL.
func NewPostgres(cfg PostgresConfig) (*DB, error) {
	driver := strings.TrimSpace(cfg.DriverName)
	if driver == "" {
		driver = DefaultPostgresDriverName
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
