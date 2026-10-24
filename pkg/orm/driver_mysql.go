package orm

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// MySQLConfig contains connection parameters for MySQL and MariaDB engines.
type MySQLConfig struct {
	Host            string
	Port            int // default: 3306
	Database        string
	User            string
	Password        string
	Charset         string // default: "utf8mb4"
	Collation       string // default: "utf8mb4_unicode_ci"
	ParseTime       bool   // default: true
	Loc             string // default: "Local"
	DriverName      string // default: "mysql"
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// FormatDSN generates a standard MySQL connection DSN:
// [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...]
func (c MySQLConfig) FormatDSN() string {
	host := c.Host
	if host == "" {
		host = "127.0.0.1"
	}

	port := c.Port
	if port <= 0 {
		port = 3306
	}

	user := c.User
	if user == "" {
		user = "root"
	}

	charset := c.Charset
	if charset == "" {
		charset = "utf8mb4"
	}

	loc := c.Loc
	if loc == "" {
		loc = "Local"
	}

	params := url.Values{}
	params.Set("charset", charset)
	params.Set("parseTime", "true")
	params.Set("loc", loc)

	if c.Collation != "" {
		params.Set("collation", c.Collation)
	}

	var authPart string
	if c.Password != "" {
		authPart = fmt.Sprintf("%s:%s", c.User, c.Password)
	} else {
		authPart = c.User
	}

	return fmt.Sprintf(
		"%s@tcp(%s:%d)/%s?%s",
		authPart,
		host,
		port,
		c.Database,
		params.Encode(),
	)
}

// NewMySQL creates and opens a DB connection pool configured for MySQL.
func NewMySQL(cfg MySQLConfig) (*DB, error) {
	driver := strings.TrimSpace(cfg.DriverName)
	if driver == "" {
		driver = "mysql"
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
