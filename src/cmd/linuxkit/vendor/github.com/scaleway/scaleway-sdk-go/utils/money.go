package utils

// Money represents an amount of money with its currency type.
type Money struct {
	// CurrencyCode is the 3-letter currency code defined in ISO 4217.
	CurrencyCode string `json:"currency_code,omitempty"`

	// Units is the whole units of the amount.
	// For example if `currencyCode` is `"USD"`, then 1 unit is one US dollar.
	Units int64 `json:"units,omitempty"`

	// Nanos is the number of nano (10^-9) units of the amount.
	// The value must be between -999,999,999 and +999,999,999 inclusive.
	// If `units` is positive, `nanos` must be positive or zero.
	// If `units` is zero, `nanos` can be positive, zero, or negative.
	// If `units` is negative, `nanos` must be negative or zero.
	// For example $-1.75 is represented as `units`=-1 and `nanos`=-750,000,000.
	Nanos int32 `json:"nanos,omitempty"`
}

// NewMoneyFromFloat conerts a float with currency to a Money object.
func NewMoneyFromFloat(value float64, currency string) *Money {
	return &Money{
		CurrencyCode: currency,
		Units:        int64(value),
		Nanos:        int32((value - float64(int64(value))) * 1000000000),
	}
}

// ToFloat converts a Money object to a float.
func (m *Money) ToFloat() float64 {
	return float64(m.Units) + float64(m.Nanos)/1000000000
}
