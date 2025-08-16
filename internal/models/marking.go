package models

import (
	"fmt"
	"sort"
	"strings"
)

// Marking represents the state of a CPN instance (case)
// It maps place names to their multisets and includes a global clock
type Marking struct {
	Places      map[string]Multiset `json:"places"`      // Key: Place Name
	GlobalClock int                 `json:"globalClock"`
}

// NewMarking creates a new empty marking
func NewMarking() *Marking {
	return &Marking{
		Places:      make(map[string]Multiset),
		GlobalClock: 0,
	}
}

// NewMarkingWithClock creates a new empty marking with the specified global clock
func NewMarkingWithClock(globalClock int) *Marking {
	return &Marking{
		Places:      make(map[string]Multiset),
		GlobalClock: globalClock,
	}
}

// AddToken adds a token to the specified place
func (m *Marking) AddToken(placeName string, token *Token) {
	if m.Places[placeName] == nil {
		m.Places[placeName] = NewMultiset()
	}
	m.Places[placeName].Add(token)
}

// RemoveToken removes a token from the specified place
// Returns true if the token was found and removed, false otherwise
func (m *Marking) RemoveToken(placeName string, token *Token) bool {
	if multiset, exists := m.Places[placeName]; exists {
		removed := multiset.Remove(token)
		// If the place becomes empty, we can optionally remove it from the map
		if multiset.IsEmpty() {
			delete(m.Places, placeName)
		}
		return removed
	}
	return false
}

// RemoveTokenByValue removes the first token with the given value from the specified place
// Returns the removed token if found, nil otherwise
func (m *Marking) RemoveTokenByValue(placeName string, value interface{}) *Token {
	if multiset, exists := m.Places[placeName]; exists {
		token := multiset.RemoveByValue(value)
		// If the place becomes empty, we can optionally remove it from the map
		if multiset.IsEmpty() {
			delete(m.Places, placeName)
		}
		return token
	}
	return nil
}

// GetMultiset returns the multiset for the specified place
// Returns an empty multiset if the place doesn't exist
func (m *Marking) GetMultiset(placeName string) Multiset {
	if multiset, exists := m.Places[placeName]; exists {
		return multiset
	}
	return NewMultiset()
}

// HasTokens checks if the specified place has any tokens
func (m *Marking) HasTokens(placeName string) bool {
	if multiset, exists := m.Places[placeName]; exists {
		return !multiset.IsEmpty()
	}
	return false
}

// HasTokenWithValue checks if the specified place has a token with the given value
func (m *Marking) HasTokenWithValue(placeName string, value interface{}) bool {
	if multiset, exists := m.Places[placeName]; exists {
		return multiset.Contains(value)
	}
	return false
}

// CountTokens returns the total number of tokens in the specified place
func (m *Marking) CountTokens(placeName string) int {
	if multiset, exists := m.Places[placeName]; exists {
		return multiset.Size()
	}
	return 0
}

// CountTokensWithValue returns the number of tokens with the given value in the specified place
func (m *Marking) CountTokensWithValue(placeName string, value interface{}) int {
	if multiset, exists := m.Places[placeName]; exists {
		return multiset.Count(value)
	}
	return 0
}

// GetTokens returns all tokens in the specified place
func (m *Marking) GetTokens(placeName string) []*Token {
	if multiset, exists := m.Places[placeName]; exists {
		return multiset.GetAllTokens()
	}
	return []*Token{}
}

// GetTokensWithValue returns all tokens with the given value in the specified place
func (m *Marking) GetTokensWithValue(placeName string, value interface{}) []*Token {
	if multiset, exists := m.Places[placeName]; exists {
		return multiset.GetTokens(value)
	}
	return []*Token{}
}

// GetPlaceNames returns all place names that have tokens
func (m *Marking) GetPlaceNames() []string {
	names := make([]string, 0, len(m.Places))
	for name := range m.Places {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// IsEmpty returns true if the marking has no tokens in any place
func (m *Marking) IsEmpty() bool {
	for _, multiset := range m.Places {
		if !multiset.IsEmpty() {
			return false
		}
	}
	return true
}

// Clear removes all tokens from all places
func (m *Marking) Clear() {
	for name := range m.Places {
		delete(m.Places, name)
	}
}

// AdvanceGlobalClock advances the global clock to the specified time
func (m *Marking) AdvanceGlobalClock(newTime int) {
	if newTime > m.GlobalClock {
		m.GlobalClock = newTime
	}
}

// GetEarliestTimestamp returns the earliest timestamp among all tokens
// Returns -1 if no tokens exist
func (m *Marking) GetEarliestTimestamp() int {
	earliest := -1
	for _, multiset := range m.Places {
		for _, tokens := range multiset {
			for _, token := range tokens {
				if earliest == -1 || token.Timestamp < earliest {
					earliest = token.Timestamp
				}
			}
		}
	}
	return earliest
}

// GetAvailableTokensAtTime returns all tokens that are available at the given time
// (i.e., tokens with timestamp <= time)
func (m *Marking) GetAvailableTokensAtTime(placeName string, time int) []*Token {
	var availableTokens []*Token
	if multiset, exists := m.Places[placeName]; exists {
		for _, tokens := range multiset {
			for _, token := range tokens {
				if token.Timestamp <= time {
					availableTokens = append(availableTokens, token)
				}
			}
		}
	}
	return availableTokens
}

// Clone creates a deep copy of the marking
func (m *Marking) Clone() *Marking {
	clone := &Marking{
		Places:      make(map[string]Multiset),
		GlobalClock: m.GlobalClock,
	}
	
	for placeName, multiset := range m.Places {
		clone.Places[placeName] = multiset.Clone()
	}
	
	return clone
}

// String returns a string representation of the marking
func (m *Marking) String() string {
	if m.IsEmpty() {
		return fmt.Sprintf("Marking{GlobalClock: %d, Places: ∅}", m.GlobalClock)
	}

	var parts []string
	
	// Sort place names for consistent output
	placeNames := m.GetPlaceNames()
	
	for _, placeName := range placeNames {
		multiset := m.Places[placeName]
		if !multiset.IsEmpty() {
			parts = append(parts, fmt.Sprintf("%s: %s", placeName, multiset.String()))
		}
	}

	placesStr := "{" + strings.Join(parts, ", ") + "}"
	return fmt.Sprintf("Marking{GlobalClock: %d, Places: %s}", m.GlobalClock, placesStr)
}

