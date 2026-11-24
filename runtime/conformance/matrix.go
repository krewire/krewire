package conformance

import (
	"fmt"
	"time"
)

// SupportTier indicates the level of support for a feature or browser platform.
type SupportTier string

const (
	// TierSupported represents full official support with automated test coverage.
	TierSupported SupportTier = "supported"
	// TierProgressive represents progressive enhancement (works with graceful degradation).
	TierProgressive SupportTier = "progressive"
	// TierUnsupported represents explicitly unsupported combinations (fails or rejected).
	TierUnsupported SupportTier = "unsupported"
)

// FeatureCategory groups related DSL or runtime capabilities.
type FeatureCategory string

const (
	CategoryGrammar       FeatureCategory = "grammar"
	CategoryDiagnostics   FeatureCategory = "diagnostics"
	CategorySSR           FeatureCategory = "ssr"
	CategoryHydration     FeatureCategory = "hydration"
	CategoryProgressive   FeatureCategory = "progressive"
	CategoryBrowserTarget FeatureCategory = "browser_target"
)

// FeatureSpec defines one entry in the .kiw Conformance Matrix.
type FeatureSpec struct {
	ID          string          `json:"id"`
	Category    FeatureCategory `json:"category"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Directives  []string        `json:"directives,omitempty"`
	TestFunc    string          `json:"test_func"`
	Tier        SupportTier     `json:"tier"`
	Notes       string          `json:"notes,omitempty"`
}

// BrowserTier defines browser/runtime baseline compatibility.
type BrowserTier struct {
	Platform    string      `json:"platform"`
	VersionMin  string      `json:"version_min"`
	Tier        SupportTier `json:"tier"`
	FeaturesReq []string    `json:"features_req"`
	Notes       string      `json:"notes"`
}

// ConformanceMatrix holds the complete catalog of verified DSL and runtime behaviors.
var ConformanceMatrix = []FeatureSpec{
	// ── 1. Grammar & Parsing ────────────────────────────────────────────────
	{
		ID:          "KIW-GRM-001",
		Category:    CategoryGrammar,
		Name:        "YAML Frontmatter Extraction",
		Description: "Parses leading YAML frontmatter delimited by --- fences into structured metadata and props.",
		Directives:  []string{"---"},
		TestFunc:    "TestConformance_Grammar_Frontmatter",
		Tier:        TierSupported,
	},
	{
		ID:          "KIW-GRM-002",
		Category:    CategoryGrammar,
		Name:        "Type-Safe Props and Defaults",
		Description: "Validates declared props with types (string, number, boolean), default values, and required constraints.",
		Directives:  []string{"props:"},
		TestFunc:    "TestConformance_Grammar_PropsAndDefaults",
		Tier:        TierSupported,
	},
	{
		ID:          "KIW-GRM-003",
		Category:    CategoryGrammar,
		Name:        "Slot Declarations and Named Slots",
		Description: "Supports default slot (<slot />) and named slots (<slot name=\"...\" />, <slot:name>, <template slot=\"...\">).",
		Directives:  []string{"<slot>", "<slot:name>", "slot=\"...\""},
		TestFunc:    "TestConformance_Grammar_Slots",
		Tier:        TierSupported,
	},
	{
		ID:          "KIW-GRM-004",
		Category:    CategoryGrammar,
		Name:        "Conditional Rendering Blocks",
		Description: "Desugars {#if condition}, {:else if condition}, and {:else} into standard Go html/template conditionals.",
		Directives:  []string{"{#if}", "{:else if}", "{:else}", "{/if}"},
		TestFunc:    "TestConformance_Grammar_Conditionals",
		Tier:        TierSupported,
	},
	{
		ID:          "KIW-GRM-005",
		Category:    CategoryGrammar,
		Name:        "Iteration and Loop Blocks",
		Description: "Desugars {#each items as item, index} into Go template range loops with loop variable preservation.",
		Directives:  []string{"{#each ... as ...}", "{/each}"},
		TestFunc:    "TestConformance_Grammar_Loops",
		Tier:        TierSupported,
	},
	{
		ID:          "KIW-GRM-006",
		Category:    CategoryGrammar,
		Name:        "Scoped Styles and Class Directives",
		Description: "Extracts <style scoped> blocks and transforms class:active={true} directives into dynamic HTML classes.",
		Directives:  []string{"<style scoped>", "class:name={...}"},
		TestFunc:    "TestConformance_Grammar_StylesAndClasses",
		Tier:        TierSupported,
	},
	{
		ID:          "KIW-GRM-007",
		Category:    CategoryGrammar,
		Name:        "Compound and Namespaced Components",
		Description: "Resolves compound tags (<Card.Header>) and namespaced components (<ui:Button>) into uniform component calls.",
		Directives:  []string{"<Namespace:Component>", "<Component.Sub>"},
		TestFunc:    "TestConformance_Grammar_CompoundAndNamespaced",
		Tier:        TierSupported,
	},

	// ── 2. Diagnostics & Error Handling ─────────────────────────────────────
	{
		ID:          "KIW-DIA-001",
		Category:    CategoryDiagnostics,
		Name:        "Line and Column Accurate Source Diagnostics",
		Description: "Maps parse and template errors back to 1-based line and column offsets in original .kiw source files.",
		TestFunc:    "TestConformance_Diagnostics_LineColAccurate",
		Tier:        TierSupported,
	},
	{
		ID:          "KIW-DIA-002",
		Category:    CategoryDiagnostics,
		Name:        "Graceful Diagnostics for Malformed Blocks",
		Description: "Provides actionable hints when encountering unclosed conditional or loop tags without compiler panic.",
		TestFunc:    "TestConformance_Diagnostics_MalformedBlocks",
		Tier:        TierSupported,
	},

	// ── 3. SSR / SSG Rendering ──────────────────────────────────────────────
	{
		ID:          "KIW-SSR-001",
		Category:    CategorySSR,
		Name:        "Contextual HTML Auto-Escaping",
		Description: "Prevents XSS by contextually escaping untrusted data in text, attributes, URIs, and embedded script payloads.",
		TestFunc:    "TestConformance_SSR_ContextualEscaping",
		Tier:        TierSupported,
	},
	{
		ID:          "KIW-SSR-002",
		Category:    CategorySSR,
		Name:        "Deterministic Idempotent Output",
		Description: "Guarantees byte-for-byte identical rendered HTML across repeated compilations of identical templates and props.",
		TestFunc:    "TestConformance_SSR_DeterministicRendering",
		Tier:        TierSupported,
	},

	// ── 4. Client Island Hydration ──────────────────────────────────────────
	{
		ID:          "KIW-HYD-001",
		Category:    CategoryHydration,
		Name:        "Progressive Hydration Directives (load, idle, visible)",
		Description: "Emits data-kiw-hydrate attributes governing when client behavior attaches: immediately, on idle, or in viewport.",
		Directives:  []string{"hydrate=\"load\"", "hydrate=\"idle\"", "hydrate=\"visible\""},
		TestFunc:    "TestConformance_Hydration_Directives",
		Tier:        TierSupported,
	},
	{
		ID:          "KIW-HYD-002",
		Category:    CategoryHydration,
		Name:        "Multi-Framework Island Markers",
		Description: "Emits data-kiw-framework markers and serializes component props into HTML-escaped JSON data-kiw-props.",
		Directives:  []string{"data-kiw-framework", "data-kiw-mount", "data-kiw-props"},
		TestFunc:    "TestConformance_Hydration_FrameworkMarkers",
		Tier:        TierSupported,
	},
	{
		ID:          "KIW-HYD-003",
		Category:    CategoryHydration,
		Name:        "Client Island Loader Injection and Asset Deduping",
		Description: "Injects kiw-islands.js module script before </body> on pages with islands and avoids duplicate injections.",
		Directives:  []string{"assets/kiw-islands.js"},
		TestFunc:    "TestConformance_Hydration_LoaderInjection",
		Tier:        TierSupported,
	},

	// ── 5. Progressive Enhancement & Fallback ───────────────────────────────
	{
		ID:          "KIW-PRG-001",
		Category:    CategoryProgressive,
		Name:        "Semantic HTML Fallback",
		Description: "Ensures complete server-rendered semantic HTML and accessible content is displayed when JavaScript is disabled.",
		TestFunc:    "TestConformance_Progressive_SemanticFallback",
		Tier:        TierSupported,
	},
	{
		ID:          "KIW-PRG-002",
		Category:    CategoryProgressive,
		Name:        "Graceful Island Hydration Failure Recovery",
		Description: "Ensures client island instantiation failure leaves server-rendered markup interactive-fallback ready without blanking.",
		TestFunc:    "TestConformance_Progressive_FaultTolerance",
		Tier:        TierSupported,
	},

	// ── 6. Browser Baseline ─────────────────────────────────────────────────
	{
		ID:          "KIW-BRW-001",
		Category:    CategoryBrowserTarget,
		Name:        "Modern Browser Baseline (Tier 1)",
		Description: "Automated headless browser simulation verifying DOM structure, hydration markers, and text visibility.",
		TestFunc:    "TestConformance_Browser_HeadlessSimulation",
		Tier:        TierSupported,
	},
}

// BrowserMatrix documents the explicit compatibility tiers for browser runtimes.
var BrowserMatrix = []BrowserTier{
	{
		Platform:    "Chromium / Google Chrome",
		VersionMin:  "100+",
		Tier:        TierSupported,
		FeaturesReq: []string{"ES6 Modules", "IntersectionObserver", "requestIdleCallback", "CustomElements"},
		Notes:       "Fully supported with all hydration schedules.",
	},
	{
		Platform:    "Mozilla Firefox",
		VersionMin:  "100+",
		Tier:        TierSupported,
		FeaturesReq: []string{"ES6 Modules", "IntersectionObserver", "requestIdleCallback", "CustomElements"},
		Notes:       "Fully supported with all hydration schedules.",
	},
	{
		Platform:    "Apple Safari / WebKit",
		VersionMin:  "15.4+",
		Tier:        TierSupported,
		FeaturesReq: []string{"ES6 Modules", "IntersectionObserver", "CustomElements"},
		Notes:       "Supported; requestIdleCallback polyfilled via setTimeout fallback.",
	},
	{
		Platform:    "Microsoft Edge (Chromium)",
		VersionMin:  "100+",
		Tier:        TierSupported,
		FeaturesReq: []string{"ES6 Modules", "IntersectionObserver", "requestIdleCallback", "CustomElements"},
		Notes:       "Fully supported with all hydration schedules.",
	},
	{
		Platform:    "Mobile WebKit (iOS Safari)",
		VersionMin:  "15.4+",
		Tier:        TierProgressive,
		FeaturesReq: []string{"ES6 Modules", "IntersectionObserver"},
		Notes:       "Progressive hydration active; idle schedules adapt to battery constraints.",
	},
	{
		Platform:    "Legacy Browsers (IE 11 / Opera Mini)",
		VersionMin:  "< All",
		Tier:        TierUnsupported,
		FeaturesReq: []string{"None"},
		Notes:       "Explicitly unsupported for client hydration. Server-rendered semantic HTML remains fully readable.",
	},
}

// ConformanceResult holds the test execution outcome for a feature.
type ConformanceResult struct {
	FeatureID string        `json:"feature_id"`
	Name      string        `json:"name"`
	Passed    bool          `json:"passed"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
}

// ConformanceReport aggregates all conformance verification results.
type ConformanceReport struct {
	Timestamp   time.Time           `json:"timestamp"`
	Total       int                 `json:"total"`
	Passed      int                 `json:"passed"`
	Failed      int                 `json:"failed"`
	CoveragePct float64             `json:"coverage_pct"`
	Results     []ConformanceResult `json:"results"`
}

// GetFeatureByID returns the feature specification matching id, or nil.
func GetFeatureByID(id string) *FeatureSpec {
	for i := range ConformanceMatrix {
		if ConformanceMatrix[i].ID == id {
			return &ConformanceMatrix[i]
		}
	}
	return nil
}

// FormatMatrixSummary produces a human-readable text summary of the conformance catalog.
func FormatMatrixSummary() string {
	var countSup, countProg, countUnsup int
	for _, f := range ConformanceMatrix {
		switch f.Tier {
		case TierSupported:
			countSup++
		case TierProgressive:
			countProg++
		case TierUnsupported:
			countUnsup++
		}
	}
	return fmt.Sprintf("Krewire .kiw Conformance Matrix: %d total features (%d supported, %d progressive, %d unsupported)",
		len(ConformanceMatrix), countSup, countProg, countUnsup)
}
