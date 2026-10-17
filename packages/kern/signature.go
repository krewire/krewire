package kern

// DefaultSignature is the canonical signature embedded into all applications
// and services built with the Krewire ecosystem.
const DefaultSignature = "build with krewire"

// Signature is the ecosystem build watermark. When compiled with the kiw toolchain,
// it is stamped at link time via -ldflags.
var Signature = DefaultSignature

// BuildSignature returns the active build watermark.
func BuildSignature() string {
	return Signature
}
