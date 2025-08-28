package models

import (
	"fmt"
	"sort"
	"strings"
)

// Marking represents the state of a CPN instance (case)
// It maps place IDs to their multisets and includes a global clock
type Marking struct {
	Places      map[string]Multiset `json:"places"` // Key: Place ID
	GlobalClock int                 `json:"globalClock"`
	StepCounter int                 `json:"currentStep"`
}

// NewMarking creates a new empty marking
func NewMarking() *Marking {
	return &Marking{
		Places:      make(map[string]Multiset),
		GlobalClock: 0,
		StepCounter: 0,
	}
}

// NewMarkingWithClock creates a new empty marking with the specified global clock
func NewMarkingWithClock(globalClock int) *Marking {
	return &Marking{
		Places:      make(map[string]Multiset),
		GlobalClock: globalClock,
		StepCounter: 0,
	}
}

// AddToken adds a token to the specified place
func (m *Marking) AddToken(placeID string, token *Token) {
	if m.Places[placeID] == nil {
		m.Places[placeID] = NewMultiset()
	}
	m.Places[placeID].Add(token)
}

// RemoveToken removes a token from the specified place
// Returns true if the token was found and removed, false otherwise

func (m *Marking) RemoveToken(placeID string, token *Token) bool {
	if multiset, exists := m.Places[placeID]; exists {
		removed := multiset.Remove(token)
		// If the place becomes empty, we can optionally remove it from the map
		if multiset.IsEmpty() {
			delete(m.Places, placeID)
		}
		return removed
	}
	return false
}

// RemoveTokenByValue removes the first token with the given value from the specified place
// Returns the removed token if found, nil otherwise

func (m *Marking) RemoveTokenByValue(placeID string, value interface{}) *Token {
	if multiset, exists := m.Places[placeID]; exists {
		token := multiset.RemoveByValue(value)
		// If the place becomes empty, we can optionally remove it from the map
		if multiset.IsEmpty() {
			delete(m.Places, placeID)
		}
		return token
	}
	return nil
}

// GetMultiset returns the multiset for the specified place
// Returns an empty multiset if the place doesn't exist
func (m *Marking) GetMultiset(placeID string) Multiset {
	if multiset, exists := m.Places[placeID]; exists {
		return multiset
	}
	return NewMultiset()
}

// HasTokens checks if the specified place has any tokens
func (m *Marking) HasTokens(placeID string) bool {
	if multiset, exists := m.Places[placeID]; exists {
		return !multiset.IsEmpty()
	}
	return false
}

// HasTokenWithValue checks if the specified place has a token with the given value
func (m *Marking) HasTokenWithValue(placeID string, value interface{}) bool {
	if multiset, exists := m.Places[placeID]; exists {
		return multiset.Contains(value)
	}
	return false
}

// CountTokens returns the total number of tokens in the specified place
func (m *Marking) CountTokens(placeID string) int {
	if multiset, exists := m.Places[placeID]; exists {
		return multiset.Size()
	}
	return 0
}

// CountTokensWithValue returns the number of tokens with the given value in the specified place
func (m *Marking) CountTokensWithValue(placeID string, value interface{}) int {
	if multiset, exists := m.Places[placeID]; exists {
		return multiset.Count(value)
	}
	return 0
}

// GetTokens returns all tokens in the specified place
func (m *Marking) GetTokens(placeID string) []*Token {
	if multiset, exists := m.Places[placeID]; exists {
		return multiset.GetAllTokens()
	}
	return []*Token{}
}

// GetTokensWithValue returns all tokens with the given value in the specified place
func (m *Marking) GetTokensWithValue(placeID string, value interface{}) []*Token {
	if multiset, exists := m.Places[placeID]; exists {
		return multiset.GetTokens(value)
	}
	return []*Token{}
}

// GetPlaceNames returns all place names that have tokens
// GetPlaceIDs returns all place IDs that have tokens
func (m *Marking) GetPlaceIDs() []string {
	ids := make([]string, 0, len(m.Places))
	for id := range m.Places {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
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

	for placeID, multiset := range m.Places {
		clone.Places[placeID] = multiset.Clone()
	}

	return clone
}

// String returns a string representation of the marking
func (m *Marking) String() string {
	if m.IsEmpty() {
		return fmt.Sprintf("Marking{GlobalClock: %d, Places: ∅}", m.GlobalClock)
	}

	var parts []string

	// Sort place IDs for consistent output
	placeIDs := m.GetPlaceIDs()

	for _, placeID := range placeIDs {
		multiset := m.Places[placeID]
		if !multiset.IsEmpty() {
			parts = append(parts, fmt.Sprintf("%s: %s", placeID, multiset.String()))
		}
	}

	placesStr := "{" + strings.Join(parts, ", ") + "}"
	return fmt.Sprintf("Marking{GlobalClock: %d, Places: %s}", m.GlobalClock, placesStr)
}
