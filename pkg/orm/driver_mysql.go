package orm

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultMySQLHost is the fallback host for MySQL.
	DefaultMySQLHost = "127.0.0.1"

	// DefaultMySQLPort is the standard MySQL port.
	DefaultMySQLPort = 3306

	// DefaultMySQLUser is the default username.
	DefaultMySQLUser = "root"

	// DefaultMySQLCharset is the default character encoding.
	DefaultMySQLCharset = "utf8mb4"

	// DefaultMySQLLoc is the default timezone location.
	DefaultMySQLLoc = "Local"

	// DefaultMySQLDriverName is the default SQL driver name.
	DefaultMySQLDriverName = "mysql"
)

// MySQLConfig contains connection parameters for MySQL and MariaDB engines.
type MySQLConfig struct {
	Host                 string
	Port                 int // default: DefaultMySQLPort (3306)
	Database             string
	User                 string
	Password             string
	Charset              string        // default: DefaultMySQLCharset ("utf8mb4")
	Collation            string        // default: "utf8mb4_unicode_ci"
	ParseTime            bool          // default: true
	Loc                  string        // default: DefaultMySQLLoc ("Local")
	DriverName           string        // default: DefaultMySQLDriverName ("mysql")
	TLS                  string        // e.g. "false", "true", "skip-verify", "preferred"
	Timeout              time.Duration // Dial timeout
	ReadTimeout          time.Duration // I/O read timeout
	WriteTimeout         time.Duration // I/O write timeout
	InterpolateParams    bool          // Client-side statement interpolation (default: true in production)
	MultiStatements      bool          // Allow multiple statements per query (default: false for security)
	AllowNativePasswords bool          // Allow native password authentication
	CheckConnLiveness    bool          // Check connection liveness before reuse
	MaxOpenConns         int
	MaxIdleConns         int
	ConnMaxLifetime      time.Duration
	ConnMaxIdleTime      time.Duration
}

// FormatDSN generates a standard MySQL connection DSN:
// [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...]
func (c MySQLConfig) FormatDSN() string {
	host := c.Host
	if host == "" {
		host = DefaultMySQLHost
	}

	port := c.Port
	if port <= 0 {
		port = DefaultMySQLPort
	}

	user := c.User
	if user == "" {
		user = DefaultMySQLUser
	}

	charset := c.Charset
	if charset == "" {
		charset = DefaultMySQLCharset
	}

	loc := c.Loc
	if loc == "" {
		loc = DefaultMySQLLoc
	}

	params := url.Values{}
	params.Set("charset", charset)
	params.Set("parseTime", "true")
	params.Set("loc", loc)

	if c.Collation != "" {
		params.Set("collation", c.Collation)
	}

	if c.TLS != "" {
		params.Set("tls", c.TLS)
	}

	if c.Timeout > 0 {
		params.Set("timeout", c.Timeout.String())
	}
	if c.ReadTimeout > 0 {
		params.Set("readTimeout", c.ReadTimeout.String())
	}
	if c.WriteTimeout > 0 {
		params.Set("writeTimeout", c.WriteTimeout.String())
	}

	if c.InterpolateParams {
		params.Set("interpolateParams", "true")
	}
	if c.MultiStatements {
		params.Set("multiStatements", "true")
	}
	if c.AllowNativePasswords {
		params.Set("allowNativePasswords", "true")
	}
	if c.CheckConnLiveness {
		params.Set("checkConnLiveness", "true")
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
		driver = DefaultMySQLDriverName
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
