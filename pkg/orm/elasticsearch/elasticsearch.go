// Package elasticsearch bundles a high-performance pure-Go Elasticsearch adapter for pkg/orm.
//
// Offers Open and Config helpers with production-friendly defaults:
// multi-node round-robin balancing, automatic retries with exponential backoff,
// bulk indexing, search query DSL, index management, and cluster health diagnostics.
//
// Built entirely on the Go standard library (net/http), with zero external
// dependencies, perfectly aligning with Pillar 1 (Secure) and Pillar 2 (Sustainable).
package elasticsearch

import (
	"time"

	"github.com/krewire/krewire/pkg/orm"
)

// DefaultAddress is the default cluster URL.
const DefaultAddress = "http://localhost:9200"

// DefaultTimeout is the standard execution timeout for Elasticsearch requests.
const DefaultTimeout = 15 * time.Second

// Config returns a baseline orm.ElasticsearchConfig for cluster addresses.
func Config(addresses ...string) orm.ElasticsearchConfig {
	if len(addresses) == 0 {
		addresses = []string{DefaultAddress}
	}

	return orm.ElasticsearchConfig{
		Addresses:    addresses,
		Timeout:      DefaultTimeout,
		MaxRetries:   3,
		RetryBackoff: 100 * time.Millisecond,
	}
}

// ProductionConfig returns an enterprise-tuned Elasticsearch configuration:
// cluster load-balancing, 20s query timeout, 3 retries with backoff.
func ProductionConfig(addresses ...string) orm.ElasticsearchConfig {
	cfg := Config(addresses...)
	cfg.Timeout = 20 * time.Second
	cfg.MaxRetries = 3
	cfg.RetryBackoff = 150 * time.Millisecond
	return cfg
}

// Open initializes and opens an Elasticsearch adapter with baseline Config defaults.
func Open(addresses ...string) (*orm.ElasticsearchAdapter, error) {
	return OpenConfig(Config(addresses...))
}

// OpenProduction initializes and opens an Elasticsearch adapter tuned for production workloads.
func OpenProduction(addresses ...string) (*orm.ElasticsearchAdapter, error) {
	return OpenConfig(ProductionConfig(addresses...))
}

// OpenConfig initializes an Elasticsearch adapter from cfg.
func OpenConfig(cfg orm.ElasticsearchConfig) (*orm.ElasticsearchAdapter, error) {
	return orm.NewElasticsearch(cfg)
}
