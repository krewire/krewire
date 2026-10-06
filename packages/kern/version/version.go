package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/krewire/krewire/packages/kern/errs"
)

// Version is a semantic version per https://semver.org/.
// Build metadata is retained but ignored for precedence.
type Version struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease string
	Build      string
}

var versionRe = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?$`)

// ParseVersion parses s as a semantic version. Leading "v" is optional.
func ParseVersion(s string) (Version, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Version{}, errs.UsageError("version is required")
	}
	m := versionRe.FindStringSubmatch(s)
	if m == nil {
		return Version{}, errs.UsageError(fmt.Sprintf("invalid version %q: want MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]", s))
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return Version{Major: major, Minor: minor, Patch: patch, PreRelease: m[4], Build: m[5]}, nil
}

// MustParseVersion parses s or panics. Use for constants.
func MustParseVersion(s string) Version {
	v, err := ParseVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the canonical string form without leading "v".
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		s += "-" + v.PreRelease
	}
	if v.Build != "" {
		s += "+" + v.Build
	}
	return s
}

// Compare returns -1 if v < other, 0 if equal, 1 if v > other per semver precedence.
// Build metadata is ignored.
func (v Version) Compare(other Version) int {
	if d := cmpInt(v.Major, other.Major); d != 0 {
		return d
	}
	if d := cmpInt(v.Minor, other.Minor); d != 0 {
		return d
	}
	if d := cmpInt(v.Patch, other.Patch); d != 0 {
		return d
	}
	if v.PreRelease == "" && other.PreRelease != "" {
		return 1
	}
	if v.PreRelease != "" && other.PreRelease == "" {
		return -1
	}
	if v.PreRelease != other.PreRelease {
		return comparePreRelease(v.PreRelease, other.PreRelease)
	}
	return 0
}

// comparePreRelease compares two (non-empty) prerelease strings per semver.org
// §11: identifiers are split on ".", a numeric identifier always has lower
// precedence than an alphanumeric one, two numeric identifiers compare
// numerically, two alphanumeric identifiers compare in ASCII order, and when
// every shared identifier is equal the longer identifier list wins.
func comparePreRelease(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		if d := comparePreReleaseIdent(as[i], bs[i]); d != 0 {
			return d
		}
	}
	return cmpInt(len(as), len(bs))
}

func comparePreReleaseIdent(a, b string) int {
	aNum, bNum := isNumericIdent(a), isNumericIdent(b)
	switch {
	case aNum && bNum:
		return compareNumericIdent(a, b)
	case aNum:
		return -1
	case bNum:
		return 1
	default:
		return cmpString(a, b)
	}
}

// isNumericIdent reports whether s is a numeric prerelease identifier (one or
// more ASCII digits). Semver forbids leading zeros; they are tolerated here
// because ParseVersion does not reject them.
func isNumericIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// compareNumericIdent compares two digit-only identifiers numerically without
// overflow: after trimming leading zeros the longer value is larger, and equal
// lengths compare lexicographically.
func compareNumericIdent(a, b string) int {
	a, b = strings.TrimLeft(a, "0"), strings.TrimLeft(b, "0")
	if len(a) != len(b) {
		return cmpInt(len(a), len(b))
	}
	return cmpString(a, b)
}

func cmpString(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// Less reports whether v < other.
func (v Version) Less(other Version) bool { return v.Compare(other) < 0 }

// Equal reports whether v == other (ignoring build).
func (v Version) Equal(other Version) bool { return v.Compare(other) == 0 }

// IsCompatible reports whether actual satisfies required per semver caret semantics
// for the Krewire ecosystem: for 0.y.z, minor must match; for >=1.0.0, major must match and actual >= required.
func (v Version) IsCompatible(required Version) bool {
	if required.Major == 0 {
		// 0.y.z: minor must match, patch >= required
		if v.Major != 0 || v.Minor != required.Minor {
			return false
		}
		return v.Compare(required) >= 0
	}
	if v.Major != required.Major {
		return false
	}
	return v.Compare(required) >= 0
}

// CurrentVersion is the kern module's own version. Bump per release.
var CurrentVersion = MustParseVersion("0.1.0")
