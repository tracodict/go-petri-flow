package models

import "fmt"

// ArcDirection represents the direction of an arc
type ArcDirection string

const (
	ArcDirectionIn  ArcDirection = "IN"  // From Place to Transition
	ArcDirectionOut ArcDirection = "OUT" // From Transition to Place
)

// Arc represents an arc connecting places and transitions
type Arc struct {
	ID           string       `json:"id"`
	SourceID     string       `json:"sourceId"`               // Can be Place or Transition ID
	TargetID     string       `json:"targetId"`               // Can be Place or Transition ID
	Expression   string       `json:"expression"`             // Lua expression
	Direction    ArcDirection `json:"direction"`              // IN or OUT
	Multiplicity int          `json:"multiplicity,omitempty"` // Optional multiplicity (default 1)
}

// NewArc creates a new arc with the given parameters
func NewArc(id, sourceID, targetID, expression string, direction ArcDirection) *Arc {
	return &Arc{
		ID:           id,
		SourceID:     sourceID,
		TargetID:     targetID,
		Expression:   expression,
		Direction:    direction,
		Multiplicity: 1,
	}
}

// NewInputArc creates a new input arc (from place to transition)
func NewInputArc(id, placeID, transitionID, expression string) *Arc {
	return &Arc{
		ID:           id,
		SourceID:     placeID,
		TargetID:     transitionID,
		Expression:   expression,
		Direction:    ArcDirectionIn,
		Multiplicity: 1,
	}
}

// NewOutputArc creates a new output arc (from transition to place)
func NewOutputArc(id, transitionID, placeID, expression string) *Arc {
	return &Arc{
		ID:           id,
		SourceID:     transitionID,
		TargetID:     placeID,
		Expression:   expression,
		Direction:    ArcDirectionOut,
		Multiplicity: 1,
	}
}

// IsInputArc returns true if this is an input arc (place to transition)
func (a *Arc) IsInputArc() bool {
	return a.Direction == ArcDirectionIn
}

// IsOutputArc returns true if this is an output arc (transition to place)
func (a *Arc) IsOutputArc() bool {
	return a.Direction == ArcDirectionOut
}

// GetPlaceID returns the place ID for this arc
func (a *Arc) GetPlaceID() string {
	if a.IsInputArc() {
		return a.SourceID
	}
	return a.TargetID
}

// GetTransitionID returns the transition ID for this arc
func (a *Arc) GetTransitionID() string {
	if a.IsInputArc() {
		return a.TargetID
	}
	return a.SourceID
}

// String returns a string representation of the arc
func (a *Arc) String() string {
	return fmt.Sprintf("Arc{ID: %s, %s -> %s, Expression: %s, Direction: %s}",
		a.ID, a.SourceID, a.TargetID, a.Expression, a.Direction)
}

// Clone creates a copy of the arc
func (a *Arc) Clone() *Arc {
	return &Arc{
		ID:           a.ID,
		SourceID:     a.SourceID,
		TargetID:     a.TargetID,
		Expression:   a.Expression,
		Direction:    a.Direction,
		Multiplicity: a.Multiplicity,
	}
}
