// Package mongodb bundles a pure-Go MongoDB document driver adapter for pkg/orm.
//
// Offers Open and Config helpers with production-friendly defaults:
// connection pooling, query timeouts, and both high-performance in-memory
// hermetic execution (for testing and embedded setups) and remote cluster support.
//
// The driver is opt-in: pkg/orm itself never imports it, preserving independent
// modularity (Pillar 2: Sustainable).
package mongodb

import (
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/orm"
)

// DefaultDatabase is the fallback database name.
const DefaultDatabase = "krewire"

// DefaultTimeout is the standard execution timeout for commands.
const DefaultTimeout = 10 * time.Second

// Config returns a baseline orm.MongoConfig for uri and database:
// standard timeouts, connection pool sizing, and auto-generated database if empty.
func Config(uri, database string) orm.MongoConfig {
	if strings.TrimSpace(database) == "" {
		database = DefaultDatabase
	}
	if strings.TrimSpace(uri) == "" {
		uri = orm.DefaultMongoMemoryURI
	}

	return orm.MongoConfig{
		URI:                    uri,
		Database:               database,
		Timeout:                DefaultTimeout,
		MinPoolSize:            5,
		MaxPoolSize:            50,
		MaxConnIdleTime:        5 * time.Minute,
		ServerSelectionTimeout: 5 * time.Second,
		AppName:                "krewire-mongodb",
	}
}

// ProductionConfig returns an enterprise-tuned MongoDB configuration:
// high-throughput connection pool, 15s operation timeout, and keepalive reaps.
func ProductionConfig(uri, database string) orm.MongoConfig {
	cfg := Config(uri, database)
	cfg.Timeout = 15 * time.Second
	cfg.MinPoolSize = 10
	cfg.MaxPoolSize = 100
	cfg.MaxConnIdleTime = 10 * time.Minute
	cfg.ServerSelectionTimeout = 10 * time.Second
	cfg.AppName = "krewire-production"
	return cfg
}

// Open initializes and opens a MongoDB adapter with baseline Config defaults.
func Open(uri, database string) (*orm.MongoAdapter, error) {
	return OpenConfig(Config(uri, database))
}

// OpenProduction initializes and opens a MongoDB adapter tuned for production workloads.
func OpenProduction(uri, database string) (*orm.MongoAdapter, error) {
	return OpenConfig(ProductionConfig(uri, database))
}

// OpenConfig initializes a MongoDB adapter from cfg.
func OpenConfig(cfg orm.MongoConfig) (*orm.MongoAdapter, error) {
	return orm.NewMongoDB(cfg)
}
