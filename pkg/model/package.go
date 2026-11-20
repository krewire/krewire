package model

// StabilityTier defines the maturity, stability guarantees, and support window of a Go package.
type StabilityTier string

const (
	// StabilityStable indicates production-ready packages with strict backwards compatibility guarantees.
	StabilityStable StabilityTier = "STABLE"

	// StabilityBeta indicates feature-complete packages undergoing API stabilization before general release.
	StabilityBeta StabilityTier = "BETA"

	// StabilityExperimental indicates early-stage packages whose APIs may evolve without deprecation windows.
	StabilityExperimental StabilityTier = "EXPERIMENTAL"

	// StabilityDeprecated indicates legacy packages slated for removal in the next major version.
	StabilityDeprecated StabilityTier = "DEPRECATED"

	// StabilityUnsupported indicates unmaintained or out-of-scope packages.
	StabilityUnsupported StabilityTier = "UNSUPPORTED"
)

// PackageMetadata describes the operational health, licensing, layering, and compatibility
// expectations of a first-party Krewire package.
type PackageMetadata struct {
	Name                 string        `json:"name"`
	ImportPath           string        `json:"import_path"`
	Layer                int           `json:"layer"`
	Purpose              string        `json:"purpose"`
	License              string        `json:"license"`
	Stability            StabilityTier `json:"stability"`
	MinGoVersion         string        `json:"min_go_version"`
	ThirdPartyDeps       []string      `json:"third_party_deps"`
	BreakingChangePolicy string        `json:"breaking_change_policy"`
	MigrationGuide       string        `json:"migration_guide,omitempty"`
}

// IsValid reports whether the stability tier is recognized.
func (s StabilityTier) IsValid() bool {
	switch s {
	case StabilityStable, StabilityBeta, StabilityExperimental, StabilityDeprecated, StabilityUnsupported:
		return true
	default:
		return false
	}
}
