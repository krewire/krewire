package model

import (
	"fmt"
	"strings"

	"github.com/krewire/krewire/packages/kern/errs"
)

// Tier represents an ecosystem tier ladder rung: free -> pro -> team -> enterprise.
// In Krewire, the codebase remains 100% open source; tiers distinguish operational
// scope, support readiness, governance, and extended capabilities.
type Tier string

const (
	TierFree       Tier = "free"
	TierPro        Tier = "pro"
	TierTeam       Tier = "team"
	TierEnterprise Tier = "enterprise"
)

// AllTiers lists every valid Tier in ascending order.
var AllTiers = []Tier{TierFree, TierPro, TierTeam, TierEnterprise}

var tierRank = map[Tier]int{
	TierFree:       0,
	TierPro:        1,
	TierTeam:       2,
	TierEnterprise: 3,
}

// IsValid reports whether t is a valid ecosystem Tier.
func (t Tier) IsValid() bool {
	_, ok := tierRank[t]
	return ok
}

// Rank returns the numeric ordering of the tier (0 = free, 1 = pro, 2 = team, 3 = enterprise).
// Invalid tiers return -1.
func (t Tier) Rank() int {
	if r, ok := tierRank[t]; ok {
		return r
	}
	return -1
}

// Less reports whether t is ranked lower than other.
func (t Tier) Less(other Tier) bool {
	return t.Rank() < other.Rank()
}

// AtLeast reports whether t satisfies or exceeds required.
func (t Tier) AtLeast(required Tier) bool {
	r1, r2 := t.Rank(), required.Rank()
	if r1 < 0 || r2 < 0 {
		return false
	}
	return r1 >= r2
}

// String returns the string representation of the tier.
func (t Tier) String() string {
	return string(t)
}

// ParseTier parses a string into a valid Tier. Empty string defaults to TierFree.
func ParseTier(s string) (Tier, error) {
	v := strings.ToLower(strings.TrimSpace(s))
	if v == "" || v == "free" {
		return TierFree, nil
	}
	t := Tier(v)
	if t.IsValid() {
		return t, nil
	}
	return "", errs.NewError(
		fmt.Sprintf("unknown tier %q: want free, pro, team, or enterprise", s),
		errs.ExitCodeUsage,
	)
}
