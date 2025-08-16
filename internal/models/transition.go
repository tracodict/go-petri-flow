package models

import "fmt"

// TransitionKind represents the type of transition
type TransitionKind string

const (
	TransitionKindAuto   TransitionKind = "Auto"
	TransitionKindManual TransitionKind = "Manual"
)

// Transition represents a transition in the CPN
type Transition struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	GuardExpression string         `json:"guardExpression"` // Lua expression
	Variables       []string       `json:"variables"`       // Variables used in guard/arc expressions
	TransitionDelay int            `json:"transitionDelay"` // Delay in time units
	Kind            TransitionKind `json:"kind"`            // Auto or Manual
}

// NewTransition creates a new transition with the given parameters
func NewTransition(id, name string) *Transition {
	return &Transition{
		ID:              id,
		Name:            name,
		GuardExpression: "",
		Variables:       []string{},
		TransitionDelay: 0,
		Kind:            TransitionKindAuto,
	}
}

// NewTransitionWithGuard creates a new transition with a guard expression
func NewTransitionWithGuard(id, name, guardExpression string, variables []string) *Transition {
	return &Transition{
		ID:              id,
		Name:            name,
		GuardExpression: guardExpression,
		Variables:       variables,
		TransitionDelay: 0,
		Kind:            TransitionKindAuto,
	}
}

// SetGuard sets the guard expression and variables for the transition
func (t *Transition) SetGuard(guardExpression string, variables []string) {
	t.GuardExpression = guardExpression
	t.Variables = variables
}

// SetDelay sets the transition delay
func (t *Transition) SetDelay(delay int) {
	t.TransitionDelay = delay
}

// SetKind sets the transition kind (Auto or Manual)
func (t *Transition) SetKind(kind TransitionKind) {
	t.Kind = kind
}

// IsAuto returns true if the transition is automatic
func (t *Transition) IsAuto() bool {
	return t.Kind == TransitionKindAuto
}

// IsManual returns true if the transition is manual
func (t *Transition) IsManual() bool {
	return t.Kind == TransitionKindManual
}

// HasGuard returns true if the transition has a guard expression
func (t *Transition) HasGuard() bool {
	return t.GuardExpression != ""
}

// String returns a string representation of the transition
func (t *Transition) String() string {
	guard := "none"
	if t.HasGuard() {
		guard = t.GuardExpression
	}
	return fmt.Sprintf("Transition{ID: %s, Name: %s, Guard: %s, Delay: %d, Kind: %s}",
		t.ID, t.Name, guard, t.TransitionDelay, t.Kind)
}

// Clone creates a copy of the transition
func (t *Transition) Clone() *Transition {
	variables := make([]string, len(t.Variables))
	copy(variables, t.Variables)

	return &Transition{
		ID:              t.ID,
		Name:            t.Name,
		GuardExpression: t.GuardExpression,
		Variables:       variables,
		TransitionDelay: t.TransitionDelay,
		Kind:            t.Kind,
	}
}
