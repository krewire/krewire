package faker

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	mathrand "math/rand"
	"strings"
	"sync"
	"time"
)

// Faker generates realistic pseudo-random fake data for testing and database seeding.
// It is thread-safe and can be deterministically seeded.
type Faker struct {
	mu  sync.Mutex
	rng *mathrand.Rand
}

var (
	defaultMu sync.Mutex
	defaultFk = New(time.Now().UnixNano())
)

// New initializes a new Faker instance with an optional seed.
func New(seed ...int64) *Faker {
	s := time.Now().UnixNano()
	if len(seed) > 0 && seed[0] != 0 {
		s = seed[0]
	}
	return &Faker{
		rng: mathrand.New(mathrand.NewSource(s)),
	}
}

// Seed resets the random seed of the default faker.
func Seed(seed int64) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultFk = New(seed)
}

// SetSeed updates the random seed of this Faker instance.
func (f *Faker) SetSeed(seed int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rng = mathrand.New(mathrand.NewSource(seed))
}

func (f *Faker) intn(n int) int {
	if n <= 0 {
		return 0
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rng.Intn(n)
}

func (f *Faker) float64() float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rng.Float64()
}

// Element returns a random element from a given slice.
func (f *Faker) Element(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[f.intn(len(items))]
}

// Shuffle returns a randomly shuffled copy of a slice.
func Shuffle[T any](f *Faker, items []T) []T {
	out := make([]T, len(items))
	copy(out, items)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rng.Shuffle(len(out), func(i, j int) {
		out[i], out[j] = out[j], out[i]
	})
	return out
}

// RandomElement returns a random element from any slice.
func RandomElement[T any](f *Faker, items []T) T {
	var zero T
	if len(items) == 0 {
		return zero
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return items[f.rng.Intn(len(items))]
}

// --- Person & Identity ---

// FirstName returns a random first name.
func (f *Faker) FirstName() string {
	return f.Element(firstNames)
}

// LastName returns a random last name.
func (f *Faker) LastName() string {
	return f.Element(lastNames)
}

// Name returns a random full name (first + last).
func (f *Faker) Name() string {
	return f.FirstName() + " " + f.LastName()
}

// Username returns a random username suitable for handles or login.
func (f *Faker) Username() string {
	first := strings.ToLower(f.FirstName())
	last := strings.ToLower(f.LastName())
	formats := []string{
		fmt.Sprintf("%s%s", first, last),
		fmt.Sprintf("%s_%s", first, last),
		fmt.Sprintf("%s%d", first, f.IntBetween(10, 999)),
		fmt.Sprintf("%s.%s%d", first, last, f.IntBetween(1, 99)),
	}
	return formats[f.intn(len(formats))]
}

// Email returns a realistic corporate or work email address.
func (f *Faker) Email() string {
	user := strings.ToLower(f.Username())
	domain := f.Element(domains)
	return fmt.Sprintf("%s@%s", user, domain)
}

// SafeEmail returns an email with a common public/free domain.
func (f *Faker) SafeEmail() string {
	user := strings.ToLower(f.Username())
	domain := f.Element(freeEmailDomains)
	return fmt.Sprintf("%s@%s", user, domain)
}

// Phone returns a formatted phone number.
func (f *Faker) Phone() string {
	formats := []string{
		fmt.Sprintf("+62 8%d-%d-%d", f.IntBetween(11, 99), f.IntBetween(1000, 9999), f.IntBetween(1000, 9999)),
		fmt.Sprintf("+1 (%d) %d-%d", f.IntBetween(200, 999), f.IntBetween(100, 999), f.IntBetween(1000, 9999)),
		fmt.Sprintf("08%d%d", f.IntBetween(11, 99), f.IntBetween(10000000, 99999999)),
	}
	return formats[f.intn(len(formats))]
}

// Gender returns a random gender.
func (f *Faker) Gender() string {
	genders := []string{"male", "female", "non-binary", "prefer-not-to-say"}
	return genders[f.intn(len(genders))]
}

// --- Internet & Tech ---

// Domain returns a random domain name.
func (f *Faker) Domain() string {
	return f.Element(domains)
}

// URL returns a full URL with HTTPS scheme.
func (f *Faker) URL() string {
	path := f.Slug()
	return fmt.Sprintf("https://%s/%s", f.Domain(), path)
}

// IPv4 returns a random valid IPv4 address.
func (f *Faker) IPv4() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		f.IntBetween(1, 254),
		f.IntBetween(1, 254),
		f.IntBetween(1, 254),
		f.IntBetween(1, 254),
	)
}

// IPv6 returns a random IPv6 address.
func (f *Faker) IPv6() string {
	parts := make([]string, 8)
	for i := range parts {
		parts[i] = fmt.Sprintf("%04x", f.IntBetween(0, 0xffff))
	}
	return strings.Join(parts, ":")
}

// UserAgent returns a realistic browser User-Agent string.
func (f *Faker) UserAgent() string {
	return f.Element(userAgents)
}

// Slug returns a kebab-cased slug made of 2-4 random words.
func (f *Faker) Slug() string {
	count := f.IntBetween(2, 4)
	words := f.Words(count)
	return strings.ToLower(strings.Join(words, "-"))
}

// UUID returns a UUID v4 string.
func (f *Faker) UUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// fallback to pseudo-random if crypto/rand fails
		f.mu.Lock()
		_, _ = f.rng.Read(b)
		f.mu.Unlock()
	}
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}

// --- Address & Geography ---

// Street returns a street address name with number.
func (f *Faker) Street() string {
	num := f.IntBetween(1, 999)
	name := f.Element(streetNames)
	return fmt.Sprintf("%s No. %d", name, num)
}

// City returns a random city name.
func (f *Faker) City() string {
	return f.Element(cities)
}

// State returns a state or province name.
func (f *Faker) State() string {
	return f.Element(states)
}

// ZipCode returns a 5-digit postal code.
func (f *Faker) ZipCode() string {
	return fmt.Sprintf("%05d", f.IntBetween(10000, 99999))
}

// Country returns a random country name.
func (f *Faker) Country() string {
	return f.Element(countries)
}

// Address returns a full multi-line or single-line address.
func (f *Faker) Address() string {
	return fmt.Sprintf("%s, %s, %s %s, %s",
		f.Street(),
		f.City(),
		f.State(),
		f.ZipCode(),
		f.Country(),
	)
}

// Latitude returns a geographic latitude (-90.0 to 90.0).
func (f *Faker) Latitude() float64 {
	lat := (f.float64() * 180.0) - 90.0
	return math.Round(lat*1000000) / 1000000
}

// Longitude returns a geographic longitude (-180.0 to 180.0).
func (f *Faker) Longitude() float64 {
	lng := (f.float64() * 360.0) - 180.0
	return math.Round(lng*1000000) / 1000000
}

// --- Company & Business ---

// Company returns a realistic company or organization name.
func (f *Faker) Company() string {
	prefix := f.Element(companyPrefixes)
	suffix := f.Element(companySuffixes)
	return prefix + " " + suffix
}

// JobTitle returns a realistic job title.
func (f *Faker) JobTitle() string {
	return f.Element(jobTitles)
}

// Department returns a business department name.
func (f *Faker) Department() string {
	return f.Element(departments)
}

// --- Commerce & Products ---

// ProductName returns a generated commercial product title.
func (f *Faker) ProductName() string {
	adj := f.Element(productAdjectives)
	mat := f.Element(productMaterials)
	typ := f.Element(productTypes)
	return fmt.Sprintf("%s %s %s", adj, mat, typ)
}

// Price returns a random price between min and max rounded to two decimal places.
func (f *Faker) Price(min, max float64) float64 {
	if min > max {
		min, max = max, min
	}
	p := min + f.float64()*(max-min)
	return math.Round(p*100) / 100
}

// CreditCardNumber returns a dummy 16-digit credit card number.
func (f *Faker) CreditCardNumber() string {
	return fmt.Sprintf("4%03d-%04d-%04d-%04d",
		f.IntBetween(100, 999),
		f.IntBetween(1000, 9999),
		f.IntBetween(1000, 9999),
		f.IntBetween(1000, 9999),
	)
}

// CurrencyCode returns a 3-letter currency code.
func (f *Faker) CurrencyCode() string {
	currencies := []string{"USD", "IDR", "EUR", "GBP", "JPY", "SGD", "AUD", "CAD"}
	return currencies[f.intn(len(currencies))]
}

// --- Text & Content ---

// Word returns a single random word.
func (f *Faker) Word() string {
	return f.Element(loremWords)
}

// Words returns n random words.
func (f *Faker) Words(n int) []string {
	if n <= 0 {
		return nil
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = f.Word()
	}
	return out
}

// Sentence returns a capitalized sentence ending with a period.
func (f *Faker) Sentence() string {
	count := f.IntBetween(6, 14)
	words := f.Words(count)
	if len(words) == 0 {
		return "Lorem ipsum dolor sit amet."
	}
	first := strings.ToUpper(string(words[0][0])) + words[0][1:]
	words[0] = first
	return strings.Join(words, " ") + "."
}

// Paragraph returns a multi-sentence paragraph.
func (f *Faker) Paragraph() string {
	count := f.IntBetween(3, 6)
	sentences := make([]string, count)
	for i := 0; i < count; i++ {
		sentences[i] = f.Sentence()
	}
	return strings.Join(sentences, " ")
}

// Paragraphs returns n paragraphs joined by double newlines.
func (f *Faker) Paragraphs(n int) string {
	if n <= 0 {
		return ""
	}
	paras := make([]string, n)
	for i := 0; i < n; i++ {
		paras[i] = f.Paragraph()
	}
	return strings.Join(paras, "\n\n")
}

// --- Numbers & Booleans ---

// IntBetween returns a random integer in [min, max].
func (f *Faker) IntBetween(min, max int) int {
	if min > max {
		min, max = max, min
	}
	diff := max - min + 1
	return min + f.intn(diff)
}

// Int64Between returns a random int64 in [min, max].
func (f *Faker) Int64Between(min, max int64) int64 {
	if min > max {
		min, max = max, min
	}
	diff := max - min + 1
	f.mu.Lock()
	defer f.mu.Unlock()
	return min + f.rng.Int63n(diff)
}

// FloatBetween returns a random float in [min, max].
func (f *Faker) FloatBetween(min, max float64) float64 {
	if min > max {
		min, max = max, min
	}
	return min + f.float64()*(max-min)
}

// Bool returns a random boolean with 50/50 probability.
func (f *Faker) Bool() bool {
	return f.intn(2) == 1
}

// BoolWithChance returns true with probability p (0-100%).
func (f *Faker) BoolWithChance(percent int) bool {
	if percent <= 0 {
		return false
	}
	if percent >= 100 {
		return true
	}
	return f.intn(100) < percent
}

// --- Dates & Times ---

// Date returns a random date within the last 5 years up to today.
func (f *Faker) Date() time.Time {
	now := time.Now()
	start := now.AddDate(-5, 0, 0)
	return f.DateBetween(start, now)
}

// PastDate returns a date between 1 year ago and now.
func (f *Faker) PastDate() time.Time {
	now := time.Now()
	start := now.AddDate(-1, 0, 0)
	return f.DateBetween(start, now)
}

// FutureDate returns a date between tomorrow and 2 years from now.
func (f *Faker) FutureDate() time.Time {
	now := time.Now()
	start := now.AddDate(0, 0, 1)
	end := now.AddDate(2, 0, 0)
	return f.DateBetween(start, end)
}

// DateBetween returns a random time between start and end.
func (f *Faker) DateBetween(start, end time.Time) time.Time {
	if start.After(end) {
		start, end = end, start
	}
	delta := end.Unix() - start.Unix()
	if delta <= 0 {
		return start
	}
	f.mu.Lock()
	sec := f.rng.Int63n(delta)
	f.mu.Unlock()
	return start.Add(time.Duration(sec) * time.Second)
}

// --- Global helper delegation functions ---

func Default() *Faker {
	return defaultFk
}

func Name() string                               { return defaultFk.Name() }
func FirstName() string                          { return defaultFk.FirstName() }
func LastName() string                           { return defaultFk.LastName() }
func Username() string                           { return defaultFk.Username() }
func Email() string                              { return defaultFk.Email() }
func SafeEmail() string                          { return defaultFk.SafeEmail() }
func Phone() string                              { return defaultFk.Phone() }
func Gender() string                             { return defaultFk.Gender() }
func Domain() string                             { return defaultFk.Domain() }
func URL() string                                { return defaultFk.URL() }
func IPv4() string                               { return defaultFk.IPv4() }
func IPv6() string                               { return defaultFk.IPv6() }
func UserAgent() string                          { return defaultFk.UserAgent() }
func Slug() string                               { return defaultFk.Slug() }
func UUID() string                               { return defaultFk.UUID() }
func Street() string                             { return defaultFk.Street() }
func City() string                               { return defaultFk.City() }
func State() string                              { return defaultFk.State() }
func ZipCode() string                            { return defaultFk.ZipCode() }
func Country() string                            { return defaultFk.Country() }
func Address() string                            { return defaultFk.Address() }
func Latitude() float64                          { return defaultFk.Latitude() }
func Longitude() float64                         { return defaultFk.Longitude() }
func Company() string                            { return defaultFk.Company() }
func JobTitle() string                           { return defaultFk.JobTitle() }
func Department() string                         { return defaultFk.Department() }
func ProductName() string                        { return defaultFk.ProductName() }
func Price(min, max float64) float64             { return defaultFk.Price(min, max) }
func CreditCardNumber() string                   { return defaultFk.CreditCardNumber() }
func CurrencyCode() string                       { return defaultFk.CurrencyCode() }
func Word() string                               { return defaultFk.Word() }
func Words(n int) []string                       { return defaultFk.Words(n) }
func Sentence() string                           { return defaultFk.Sentence() }
func Paragraph() string                          { return defaultFk.Paragraph() }
func Paragraphs(n int) string                    { return defaultFk.Paragraphs(n) }
func IntBetween(min, max int) int                { return defaultFk.IntBetween(min, max) }
func FloatBetween(min, max float64) float64      { return defaultFk.FloatBetween(min, max) }
func Bool() bool                                 { return defaultFk.Bool() }
func BoolWithChance(percent int) bool            { return defaultFk.BoolWithChance(percent) }
func Date() time.Time                            { return defaultFk.Date() }
func PastDate() time.Time                        { return defaultFk.PastDate() }
func FutureDate() time.Time                      { return defaultFk.FutureDate() }
func DateBetween(start, end time.Time) time.Time { return defaultFk.DateBetween(start, end) }
