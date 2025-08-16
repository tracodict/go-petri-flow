package models

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Token represents a colored token with a value and timestamp
type Token struct {
	Value     interface{} `json:"value"`
	Timestamp int         `json:"timestamp"`
}

// NewToken creates a new token with the given value and timestamp
func NewToken(value interface{}, timestamp int) *Token {
	return &Token{
		Value:     value,
		Timestamp: timestamp,
	}
}

// String returns a string representation of the token
func (t *Token) String() string {
	return fmt.Sprintf("Token{Value: %v, Timestamp: %d}", t.Value, t.Timestamp)
}

// Equals checks if two tokens are equal (same value and timestamp)
func (t *Token) Equals(other *Token) bool {
	if other == nil {
		return false
	}
	return t.Value == other.Value && t.Timestamp == other.Timestamp
}

// ValueString returns the string representation of the token's value
func (t *Token) ValueString() string {
	switch v := t.Value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		// For complex types, use JSON marshaling
		if bytes, err := json.Marshal(v); err == nil {
			return string(bytes)
		}
		return fmt.Sprintf("%v", v)
	}
}

// Clone creates a deep copy of the token
func (t *Token) Clone() *Token {
	return &Token{
		Value:     t.Value, // Note: This is a shallow copy of the value
		Timestamp: t.Timestamp,
	}
}

