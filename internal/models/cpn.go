package models

import (
	"fmt"
	"strings"
)

// CPN represents a Colored Petri Net
type CPN struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	Places         []*Place            `json:"places"`
	Transitions    []*Transition       `json:"transitions"`
	Arcs           []*Arc              `json:"arcs"`
	InitialMarking map[string][]*Token `json:"initialMarking"` // Initial tokens by place ID
	EndPlaces      []string            `json:"endPlaces"`      // Places that signify case completion (still by name for UX)
}

// NewCPN creates a new CPN with the given ID, name, and description
func NewCPN(id, name, description string) *CPN {
	return &CPN{
		ID:             id,
		Name:           name,
		Description:    description,
		Places:         []*Place{},
		Transitions:    []*Transition{},
		Arcs:           []*Arc{},
		InitialMarking: make(map[string][]*Token),
		EndPlaces:      []string{},
	}
}

// AddPlace adds a place to the CPN
func (cpn *CPN) AddPlace(place *Place) {
	cpn.Places = append(cpn.Places, place)
}

// AddTransition adds a transition to the CPN
func (cpn *CPN) AddTransition(transition *Transition) {
	cpn.Transitions = append(cpn.Transitions, transition)
}

// AddArc adds an arc to the CPN
func (cpn *CPN) AddArc(arc *Arc) {
	cpn.Arcs = append(cpn.Arcs, arc)
}

// SetInitialMarking sets the initial marking for a place
func (cpn *CPN) SetInitialMarking(placeID string, tokens []*Token) {
	cpn.InitialMarking[placeID] = tokens
}

// AddInitialToken adds a token to the initial marking of a place
func (cpn *CPN) AddInitialToken(placeID string, token *Token) {
	cpn.InitialMarking[placeID] = append(cpn.InitialMarking[placeID], token)
}

// SetEndPlaces sets the end places for the CPN
func (cpn *CPN) SetEndPlaces(endPlaces []string) {
	cpn.EndPlaces = endPlaces
}

// GetPlace returns the place with the given ID
func (cpn *CPN) GetPlace(id string) *Place {
	for _, place := range cpn.Places {
		if place.ID == id {
			return place
		}
	}
	return nil
}

// GetPlaceByName returns the place with the given name
func (cpn *CPN) GetPlaceByName(name string) *Place {
	for _, place := range cpn.Places {
		if place.Name == name {
			return place
		}
	}
	return nil
}

// GetTransition returns the transition with the given ID
func (cpn *CPN) GetTransition(id string) *Transition {
	for _, transition := range cpn.Transitions {
		if transition.ID == id {
			return transition
		}
	}
	return nil
}

// GetTransitionByName returns the transition with the given name
func (cpn *CPN) GetTransitionByName(name string) *Transition {
	for _, transition := range cpn.Transitions {
		if transition.Name == name {
			return transition
		}
	}
	return nil
}

// GetArc returns the arc with the given ID
func (cpn *CPN) GetArc(id string) *Arc {
	for _, arc := range cpn.Arcs {
		if arc.ID == id {
			return arc
		}
	}
	return nil
}

// GetInputArcs returns all input arcs for the given transition
func (cpn *CPN) GetInputArcs(transitionID string) []*Arc {
	var inputArcs []*Arc
	for _, arc := range cpn.Arcs {
		if arc.IsInputArc() && arc.GetTransitionID() == transitionID {
			inputArcs = append(inputArcs, arc)
		}
	}
	return inputArcs
}

// GetOutputArcs returns all output arcs for the given transition
func (cpn *CPN) GetOutputArcs(transitionID string) []*Arc {
	var outputArcs []*Arc
	for _, arc := range cpn.Arcs {
		if arc.IsOutputArc() && arc.GetTransitionID() == transitionID {
			outputArcs = append(outputArcs, arc)
		}
	}
	return outputArcs
}

// GetArcsForPlace returns all arcs connected to the given place
func (cpn *CPN) GetArcsForPlace(placeID string) []*Arc {
	var arcs []*Arc
	for _, arc := range cpn.Arcs {
		if arc.GetPlaceID() == placeID {
			arcs = append(arcs, arc)
		}
	}
	return arcs
}

// CreateInitialMarking creates a marking based on the initial marking definition
func (cpn *CPN) CreateInitialMarking() *Marking {
	marking := NewMarking()
	for placeID, tokens := range cpn.InitialMarking {
		for _, token := range tokens {
			marking.AddToken(placeID, token.Clone())
		}
	}
	return marking
}

// ValidateStructure performs basic structural validation of the CPN
func (cpn *CPN) ValidateStructure() []error {
	var errors []error

	// Check for duplicate place IDs
	placeIDs := make(map[string]bool)
	for _, place := range cpn.Places {
		if placeIDs[place.ID] {
			errors = append(errors, fmt.Errorf("duplicate place ID: %s", place.ID))
		}
		placeIDs[place.ID] = true

		// Check if place has a color set
		if place.ColorSet == nil {
			errors = append(errors, fmt.Errorf("place %s has no color set", place.Name))
		}
	}

	// Check for duplicate transition IDs
	transitionIDs := make(map[string]bool)
	for _, transition := range cpn.Transitions {
		if transitionIDs[transition.ID] {
			errors = append(errors, fmt.Errorf("duplicate transition ID: %s", transition.ID))
		}
		transitionIDs[transition.ID] = true
	}

	// Check for duplicate arc IDs
	arcIDs := make(map[string]bool)
	for _, arc := range cpn.Arcs {
		if arcIDs[arc.ID] {
			errors = append(errors, fmt.Errorf("duplicate arc ID: %s", arc.ID))
		}
		arcIDs[arc.ID] = true

		// Check if arc references valid places and transitions
		if arc.IsInputArc() {
			if cpn.GetPlace(arc.SourceID) == nil {
				errors = append(errors, fmt.Errorf("arc %s references non-existent place: %s", arc.ID, arc.SourceID))
			}
			if cpn.GetTransition(arc.TargetID) == nil {
				errors = append(errors, fmt.Errorf("arc %s references non-existent transition: %s", arc.ID, arc.TargetID))
			}
		} else {
			if cpn.GetTransition(arc.SourceID) == nil {
				errors = append(errors, fmt.Errorf("arc %s references non-existent transition: %s", arc.ID, arc.SourceID))
			}
			if cpn.GetPlace(arc.TargetID) == nil {
				errors = append(errors, fmt.Errorf("arc %s references non-existent place: %s", arc.ID, arc.TargetID))
			}
		}
	}

	// Check if initial marking references valid places
	for placeID := range cpn.InitialMarking {
		if cpn.GetPlace(placeID) == nil {
			errors = append(errors, fmt.Errorf("initial marking references non-existent place ID: %s", placeID))
		}
	}

	// Check if end places reference valid places
	for _, endPlace := range cpn.EndPlaces {
		if cpn.GetPlaceByName(endPlace) == nil {
			errors = append(errors, fmt.Errorf("end place references non-existent place name: %s", endPlace))
		}
	}

	return errors
}

// IsCompleted checks if the CPN is in a completed state based on end places
func (cpn *CPN) IsCompleted(marking *Marking) bool {
	if len(cpn.EndPlaces) == 0 {
		return false
	}

	// Check if all end places have tokens
	for _, endPlace := range cpn.EndPlaces {
		pl := cpn.GetPlaceByName(endPlace)
		if pl == nil || !marking.HasTokens(pl.ID) {
			return false
		}
	}

	return true
}

// String returns a string representation of the CPN
func (cpn *CPN) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("CPN{ID: %s, Name: %s}", cpn.ID, cpn.Name))
	parts = append(parts, fmt.Sprintf("  Places: %d", len(cpn.Places)))
	parts = append(parts, fmt.Sprintf("  Transitions: %d", len(cpn.Transitions)))
	parts = append(parts, fmt.Sprintf("  Arcs: %d", len(cpn.Arcs)))

	if len(cpn.EndPlaces) > 0 {
		parts = append(parts, fmt.Sprintf("  End Places: [%s]", strings.Join(cpn.EndPlaces, ", ")))
	}

	return strings.Join(parts, "\n")
}

// Clone creates a deep copy of the CPN
func (cpn *CPN) Clone() *CPN {
	clone := &CPN{
		ID:             cpn.ID,
		Name:           cpn.Name,
		Description:    cpn.Description,
		Places:         make([]*Place, len(cpn.Places)),
		Transitions:    make([]*Transition, len(cpn.Transitions)),
		Arcs:           make([]*Arc, len(cpn.Arcs)),
		InitialMarking: make(map[string][]*Token),
		EndPlaces:      make([]string, len(cpn.EndPlaces)),
	}

	// Clone places
	for i, place := range cpn.Places {
		clone.Places[i] = place.Clone()
	}

	// Clone transitions
	for i, transition := range cpn.Transitions {
		clone.Transitions[i] = transition.Clone()
	}

	// Clone arcs
	for i, arc := range cpn.Arcs {
		clone.Arcs[i] = arc.Clone()
	}

	// Clone initial marking
	for placeID, tokens := range cpn.InitialMarking {
		clonedTokens := make([]*Token, len(tokens))
		for i, token := range tokens {
			clonedTokens[i] = token.Clone()
		}
		clone.InitialMarking[placeID] = clonedTokens
	}

	// Clone end places
	copy(clone.EndPlaces, cpn.EndPlaces)

	return clone
}
