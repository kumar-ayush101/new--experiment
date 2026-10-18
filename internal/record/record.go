// Package record defines the data model that flows through a LogQL pipeline: a
// Record is a set of named fields, and a Value is a dynamically-typed scalar
// (string, number, or bool). Keeping values typed lets comparisons like
// `status >= 500` work numerically while string fields compare lexically.
package record

import "strconv"

// Kind is the dynamic type of a Value.
type Kind uint8

const (
	Null Kind = iota
	String
	Number
	Bool
)

// Value is a dynamically-typed scalar field value.
type Value struct {
	Kind Kind
	Str  string
	Num  float64
	B    bool
}

// Record is a single parsed log line as an ordered-insensitive field map.
type Record map[string]Value

// Null value singleton helper.
func Nullv() Value { return Value{Kind: Null} }

// Str builds a string value, auto-promoting to a number when the text is
// numeric. This is what makes `where status >= 500` work on logfmt input where
// everything starts as text.
func Str(s string) Value {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return Value{Kind: Number, Num: f, Str: s}
	}
	switch s {
	case "true":
		return Value{Kind: Bool, B: true, Str: s}
	case "false":
		return Value{Kind: Bool, B: false, Str: s}
	}
	return Value{Kind: String, Str: s}
}

// RawStr builds a string value without numeric promotion.
func RawStr(s string) Value { return Value{Kind: String, Str: s} }

// Numv builds a number value.
func Numv(f float64) Value {
	return Value{Kind: Number, Num: f, Str: strconv.FormatFloat(f, 'g', -1, 64)}
}

// Boolv builds a bool value.
func Boolv(b bool) Value {
	return Value{Kind: Bool, B: b, Str: strconv.FormatBool(b)}
}

// String renders a value as text for output.
func (v Value) String() string {
	switch v.Kind {
	case Null:
		return ""
	case Number:
		if v.Str != "" {
			return v.Str
		}
		return strconv.FormatFloat(v.Num, 'g', -1, 64)
	case Bool:
		return strconv.FormatBool(v.B)
	default:
		return v.Str
	}
}

// AsNumber returns the numeric value and whether it is numeric.
func (v Value) AsNumber() (float64, bool) {
	switch v.Kind {
	case Number:
		return v.Num, true
	case Bool:
		if v.B {
			return 1, true
		}
		return 0, true
	default:
		if f, err := strconv.ParseFloat(v.Str, 64); err == nil {
			return f, true
		}
		return 0, false
	}
}

// Truthy reports whether the value is logically true (for predicate results).
func (v Value) Truthy() bool {
	switch v.Kind {
	case Bool:
		return v.B
	case Number:
		return v.Num != 0
	case String:
		return v.Str != ""
	default:
		return false
	}
}
