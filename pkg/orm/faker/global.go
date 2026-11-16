package faker

import "time"

// Default returns the package-level default Faker instance.
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
