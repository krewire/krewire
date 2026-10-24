package assets

import (
	"strings"
	"testing"
)

func TestMinifyCSS(t *testing.T) {
	input := `
		/* Main stylesheet header comment */
		:root {
			--primary-color: #3b82f6;
			--spacing-large: 24px;
		}

		/* Empty rule that should be stripped */
		.empty-class {
		}

		.container {
			display: flex;
			flex-direction: column;
			gap: 16px;
			padding: 0 1rem;
		}

		/* Multi-line comment inside selector */
		.card,
		.panel {
			background: #ffffff;
			border: 1px solid #e5e7eb;
			content: "hello   world   spaces";
		}

		@media (min-width: 768px) and (max-width: 1024px) {
			.container {
				grid-template-columns: repeat(2, 1fr);
			}
		}
	`

	min := MinifyCSS(input)

	// 1. Comments must be gone
	if strings.Contains(min, "Main stylesheet header comment") || strings.Contains(min, "/*") {
		t.Errorf("expected comments to be stripped, got: %s", min)
	}

	// 2. Empty class should be removed
	if strings.Contains(min, ".empty-class") {
		t.Errorf("expected empty rule .empty-class to be removed, got: %s", min)
	}

	// 3. String literal with spaces must be preserved verbatim
	if !strings.Contains(min, `"hello   world   spaces"`) {
		t.Errorf("expected string literal with spaces to be preserved, got: %s", min)
	}

	// 4. CSS variables must be preserved
	if !strings.Contains(min, "--primary-color:#3b82f6") && !strings.Contains(min, "--primary-color: #3b82f6") {
		t.Errorf("expected CSS variable to be preserved, got: %s", min)
	}

	// 5. Media query syntax must remain intact
	if !strings.Contains(min, "@media (min-width: 768px) and (max-width: 1024px)") &&
		!strings.Contains(min, "@media (min-width:768px) and (max-width:1024px)") {
		t.Errorf("expected media query to be preserved intact, got: %s", min)
	}

	// 6. Significant size reduction (>= 25%)
	origLen := len(input)
	minLen := len(min)
	reduction := float64(origLen-minLen) / float64(origLen) * 100
	if reduction < 25.0 {
		t.Errorf("expected at least 25%% reduction, got %.1f%% (%d -> %d bytes)", reduction, origLen, minLen)
	}
}

func TestMinifyJS(t *testing.T) {
	input := `
		// Top-level script comment
		const apiEndpoint = "https://krewire.com/api/v1";

		/* Multi-line
		   block comment
		*/
		function calculateTotal(price, qty) {
			// Inline calculation comment
			const tax = 0.11;
			return price * qty * (1 + tax);
		}

		const template = ` + "`" + `
			<div>
				<h1>Title</h1>
			</div>
		` + "`" + `;
	`

	min := MinifyJS(input)

	// 1. Comments stripped
	if strings.Contains(min, "Top-level script comment") || strings.Contains(min, "Inline calculation comment") {
		t.Errorf("expected comments to be stripped, got: %s", min)
	}

	// 2. URLs with // preserved inside strings
	if !strings.Contains(min, "https://krewire.com/api/v1") {
		t.Errorf("expected URL to be preserved in string, got: %s", min)
	}

	// 3. Template literal preserved
	if !strings.Contains(min, "<h1>Title</h1>") {
		t.Errorf("expected template literal content preserved, got: %s", min)
	}

	// 4. Size reduction
	if len(min) >= len(input) {
		t.Errorf("expected minified JS size to be smaller than input")
	}
}

func BenchmarkMinifyCSS(b *testing.B) {
	sample := strings.Repeat(`
		.btn-primary {
			background-color: #2563eb;
			color: #ffffff;
			padding: 0.5rem 1rem;
			border-radius: 0.375rem;
		}
		/* comment */
		.btn-primary:hover {
			background-color: #1d4ed8;
		}
	`, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MinifyCSS(sample)
	}
}

func BenchmarkMinifyJS(b *testing.B) {
	sample := strings.Repeat(`
		// calculate item
		function getItem(id) {
			/* fetch from api */
			const url = "https://krewire.com/items/" + id;
			return fetch(url).then(r => r.json());
		}
	`, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MinifyJS(sample)
	}
}
