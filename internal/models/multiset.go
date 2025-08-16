package models

import (
	"fmt"
	"sort"
	"strings"
)

// Multiset represents a collection of tokens in a place
// Key: token value string, Value: list of tokens (for timestamps)
type Multiset map[string][]*Token

// NewMultiset creates a new empty multiset
func NewMultiset() Multiset {
	return make(Multiset)
}

// Add adds a token to the multiset
func (ms Multiset) Add(token *Token) {
	valueStr := token.ValueString()
	ms[valueStr] = append(ms[valueStr], token)
}

// Remove removes a token from the multiset
// Returns true if the token was found and removed, false otherwise
func (ms Multiset) Remove(token *Token) bool {
	valueStr := token.ValueString()
	tokens, exists := ms[valueStr]
	if !exists {
		return false
	}

	// Find and remove the first matching token
	for i, t := range tokens {
		if t.Equals(token) {
			// Remove the token from the slice
			ms[valueStr] = append(tokens[:i], tokens[i+1:]...)
			// If no more tokens with this value, remove the key
			if len(ms[valueStr]) == 0 {
				delete(ms, valueStr)
			}
			return true
		}
	}
	return false
}

// RemoveByValue removes the first token with the given value
// Returns the removed token if found, nil otherwise
func (ms Multiset) RemoveByValue(value interface{}) *Token {
	valueStr := tokenValueToString(value)
	tokens, exists := ms[valueStr]
	if !exists || len(tokens) == 0 {
		return nil
	}

	// Remove the first token
	token := tokens[0]
	ms[valueStr] = tokens[1:]
	
	// If no more tokens with this value, remove the key
	if len(ms[valueStr]) == 0 {
		delete(ms, valueStr)
	}
	
	return token
}

// Contains checks if the multiset contains a token with the given value
func (ms Multiset) Contains(value interface{}) bool {
	valueStr := tokenValueToString(value)
	tokens, exists := ms[valueStr]
	return exists && len(tokens) > 0
}

// Count returns the number of tokens with the given value
func (ms Multiset) Count(value interface{}) int {
	valueStr := tokenValueToString(value)
	if tokens, exists := ms[valueStr]; exists {
		return len(tokens)
	}
	return 0
}

// Size returns the total number of tokens in the multiset
func (ms Multiset) Size() int {
	total := 0
	for _, tokens := range ms {
		total += len(tokens)
	}
	return total
}

// IsEmpty returns true if the multiset is empty
func (ms Multiset) IsEmpty() bool {
	return ms.Size() == 0
}

// GetTokens returns all tokens with the given value
func (ms Multiset) GetTokens(value interface{}) []*Token {
	valueStr := tokenValueToString(value)
	if tokens, exists := ms[valueStr]; exists {
		// Return a copy to prevent external modification
		result := make([]*Token, len(tokens))
		copy(result, tokens)
		return result
	}
	return []*Token{}
}

// GetAllTokens returns all tokens in the multiset
func (ms Multiset) GetAllTokens() []*Token {
	var result []*Token
	for _, tokens := range ms {
		result = append(result, tokens...)
	}
	return result
}

// GetValues returns all unique values in the multiset
func (ms Multiset) GetValues() []interface{} {
	var values []interface{}
	for _, tokens := range ms {
		if len(tokens) > 0 {
			values = append(values, tokens[0].Value)
		}
	}
	return values
}

// Clear removes all tokens from the multiset
func (ms Multiset) Clear() {
	for k := range ms {
		delete(ms, k)
	}
}

// Clone creates a deep copy of the multiset
func (ms Multiset) Clone() Multiset {
	clone := NewMultiset()
	for valueStr, tokens := range ms {
		clonedTokens := make([]*Token, len(tokens))
		for i, token := range tokens {
			clonedTokens[i] = token.Clone()
		}
		clone[valueStr] = clonedTokens
	}
	return clone
}

// String returns a string representation of the multiset
func (ms Multiset) String() string {
	if ms.IsEmpty() {
		return "∅"
	}

	var parts []string
	
	// Sort keys for consistent output
	keys := make([]string, 0, len(ms))
	for k := range ms {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		tokens := ms[key]
		if len(tokens) == 1 {
			parts = append(parts, key)
		} else {
			parts = append(parts, fmt.Sprintf("%d`%s", len(tokens), key))
		}
	}

	return "{" + strings.Join(parts, ", ") + "}"
}

// tokenValueToString converts a token value to its string representation
func tokenValueToString(value interface{}) string {
	token := &Token{Value: value}
	return token.ValueString()
}

