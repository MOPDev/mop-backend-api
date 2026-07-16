package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Money represents monetary amounts in the smallest subunit (øre/cents).
// E.g., 150.95 is represented internally as 15095.
type Money int64

// UnmarshalJSON handles incoming values from Gin.
// It parses raw floats (150.95), JSON numbers, or strings containing decimals.
// Since you are handling Danish values, it automatically replaces commas with dots.
func (m *Money) UnmarshalJSON(data []byte) error {
	// Remove outer quotes if the JSON sent it as a string
	str := strings.Trim(string(data), `"`)

	if str == "null" || str == "" {
		*m = 0
		return nil
	}

	// Danish-locale safeguard: replace comma with dot (e.g. "150,95" -> "150.95")
	str = strings.ReplaceAll(str, ",", ".")

	// Parse it as a float first
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return fmt.Errorf("invalid money format: %w", err)
	}

	// Convert float to integer øre/cents safely.
	// Using math.Round is crucial to avoid IEEE 754 precision truncations
	// (e.g., 150.95 * 100 yielding 15094.99999...)
	*m = Money(math.Round(val * 100))
	return nil
}

// MarshalJSON converts the internal integer back to a standard decimal float for APIs.
func (m Money) MarshalJSON() ([]byte, error) {
	floatVal := float64(m) / 100.0
	return json.Marshal(floatVal)
}

// Value implements driver.Valuer (writes to SQLite)
func (m Money) Value() (driver.Value, error) {
	return int64(m), nil
}

// Scan implements sql.Scanner (reads from SQLite)
func (m *Money) Scan(value interface{}) error {
	if value == nil {
		*m = 0 // Or handle null as zero safely
		return nil
	}
	switch v := value.(type) {
	case int64:
		*m = Money(v)
	case []byte:
		// SQLite sometimes returns numbers as text bytes
		val, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return err
		}
		*m = Money(val)
	default:
		return fmt.Errorf("cannot scan %T into Money", value)
	}
	return nil
}

// FormatDK returns a pretty-printed Danish string (e.g., "150,95 kr.")
func (m Money) FormatDK() string {
	floatVal := float64(m) / 100.0
	res := fmt.Sprintf("%.2f", floatVal)
	return strings.ReplaceAll(res, ".", ",") + " kr."
}
