package models

import "fmt"

// Place represents a place in the CPN
type Place struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	ColorSet ColorSet  `json:"colorSet"`
	Position *Position `json:"position,omitempty"`
}

// NewPlace creates a new place with the given ID, name, and color set
func NewPlace(id, name string, colorSet ColorSet) *Place {
	return &Place{
		ID:       id,
		Name:     name,
		ColorSet: colorSet,
		Position: nil,
	}
}

// ValidateToken checks if a token's value is valid for this place's color set
func (p *Place) ValidateToken(token *Token) error {
	if p.ColorSet == nil {
		return fmt.Errorf("place %s has no color set defined", p.Name)
	}

	if !p.ColorSet.IsMember(token.Value) {
		return fmt.Errorf("token value %v is not a member of color set %s for place %s",
			token.Value, p.ColorSet.Name(), p.Name)
	}

	return nil
}

// String returns a string representation of the place
func (p *Place) String() string {
	colorSetName := "nil"
	if p.ColorSet != nil {
		colorSetName = p.ColorSet.Name()
	}
	return fmt.Sprintf("Place{ID: %s, Name: %s, ColorSet: %s}", p.ID, p.Name, colorSetName)
}

// Clone creates a copy of the place
func (p *Place) Clone() *Place {
	return &Place{
		ID:       p.ID,
		Name:     p.Name,
		ColorSet: p.ColorSet, // ColorSet is typically immutable, so shallow copy is fine
	}
}
