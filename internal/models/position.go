package models

// Position represents a 2D coordinate for visualization
// Used by Place and Transition
// JSON: { "x": 80, "y": 160 }
type Position struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}
