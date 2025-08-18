package models

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// CPNParser handles parsing of CPN definitions from JSON format
type CPNParser struct {
	colorSetParser *ColorSetParser
}

// NewCPNParser creates a new CPN parser
func NewCPNParser() *CPNParser {
	return &CPNParser{
		colorSetParser: NewColorSetParser(),
	}
}

// CPNDefinitionJSON represents the JSON structure for CPN definitions
type CPNDefinitionJSON struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	ColorSets      []string               `json:"colorSets,omitempty"`   // Color set definitions
	JsonSchemas    []JsonSchemaDef        `json:"jsonSchemas,omitempty"` // Optional JSON Schemas
	Places         []PlaceJSON            `json:"places"`
	Transitions    []TransitionJSON       `json:"transitions"`
	Arcs           []ArcJSON              `json:"arcs"`
	InitialMarking map[string][]TokenJSON `json:"initialMarking,omitempty"`
	EndPlaces      []string               `json:"endPlaces,omitempty"`
}

// JsonSchemaDef represents a named JSON Schema definition
type JsonSchemaDef struct {
	Name   string      `json:"name"`
	Schema interface{} `json:"schema"`
}

// parseJsonSchemas compiles and registers JSON Schemas for later json<SchemaName> color set references
func (p *CPNParser) parseJsonSchemas(defs []JsonSchemaDef) error {
	if len(defs) == 0 {
		return nil
	}
	compiler := jsonschema.NewCompiler()
	for _, d := range defs {
		if d.Name == "" || d.Schema == nil {
			return fmt.Errorf("invalid json schema definition (missing name or schema)")
		}
		// Marshal schema object back to JSON bytes for compiler
		data, err := json.Marshal(d.Schema)
		if err != nil {
			return fmt.Errorf("failed to marshal schema %s: %v", d.Name, err)
		}
		// Use synthetic URL id
		url := "mem://schemas/" + d.Name + ".json"
		if err := compiler.AddResource(url, bytes.NewReader(data)); err != nil {
			return fmt.Errorf("failed to add schema resource %s: %v", d.Name, err)
		}
		compiled, err := compiler.Compile(url)
		if err != nil {
			return fmt.Errorf("failed to compile schema %s: %v", d.Name, err)
		}
		p.colorSetParser.jsonSchemas[d.Name] = compiled
	}
	return nil
}

// PlaceJSON represents the JSON structure for places
type PlaceJSON struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	ColorSet string    `json:"colorSet"` // Name of the color set
	Position *Position `json:"position,omitempty"`
}

// TransitionJSON represents the JSON structure for transitions
type TransitionJSON struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	GuardExpression string    `json:"guardExpression,omitempty"`
	Variables       []string  `json:"variables,omitempty"`
	TransitionDelay int       `json:"transitionDelay,omitempty"`
	Kind            string    `json:"kind,omitempty"` // "Auto" or "Manual"
	Position        *Position `json:"position,omitempty"`
}

// ArcJSON represents the JSON structure for arcs
type ArcJSON struct {
	ID         string `json:"id"`
	SourceID   string `json:"sourceId"`
	TargetID   string `json:"targetId"`
	Expression string `json:"expression"`
	Direction  string `json:"direction"` // "IN" or "OUT"
}

// TokenJSON represents the JSON structure for tokens
type TokenJSON struct {
	Value     interface{} `json:"value"`
	Timestamp int         `json:"timestamp"`
}

// ParseCPNFromJSON parses a CPN definition from JSON
func (p *CPNParser) ParseCPNFromJSON(jsonData []byte) (*CPN, error) {
	var cpnDef CPNDefinitionJSON
	if err := json.Unmarshal(jsonData, &cpnDef); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return p.ParseCPNFromDefinition(&cpnDef)
}

// ParseCPNFromDefinition parses a CPN from a CPNDefinitionJSON structure
func (p *CPNParser) ParseCPNFromDefinition(cpnDef *CPNDefinitionJSON) (*CPN, error) {
	// Create the CPN
	cpn := NewCPN(cpnDef.ID, cpnDef.Name, cpnDef.Description)

	// Load JSON Schemas first so color sets can reference them
	if err := p.parseJsonSchemas(cpnDef.JsonSchemas); err != nil {
		return nil, fmt.Errorf("failed to parse json schemas: %v", err)
	}

	// Parse color sets first
	if err := p.parseColorSets(cpnDef.ColorSets); err != nil {
		return nil, fmt.Errorf("failed to parse color sets: %v", err)
	}

	// Parse places
	if err := p.parsePlaces(cpn, cpnDef.Places); err != nil {
		return nil, fmt.Errorf("failed to parse places: %v", err)
	}

	// Parse transitions
	if err := p.parseTransitions(cpn, cpnDef.Transitions); err != nil {
		return nil, fmt.Errorf("failed to parse transitions: %v", err)
	}

	// Parse arcs
	if err := p.parseArcs(cpn, cpnDef.Arcs); err != nil {
		return nil, fmt.Errorf("failed to parse arcs: %v", err)
	}

	// Parse initial marking
	if err := p.parseInitialMarking(cpn, cpnDef.InitialMarking); err != nil {
		return nil, fmt.Errorf("failed to parse initial marking: %v", err)
	}

	// Set end places
	cpn.SetEndPlaces(cpnDef.EndPlaces)

	// Validate the CPN structure
	if errors := cpn.ValidateStructure(); len(errors) > 0 {
		return nil, fmt.Errorf("CPN validation failed: %v", errors)
	}

	return cpn, nil
}

// parseColorSets parses color set definitions
func (p *CPNParser) parseColorSets(colorSetDefs []string) error {
	for _, def := range colorSetDefs {
		if _, err := p.colorSetParser.ParseColorSetDefinition(def); err != nil {
			return fmt.Errorf("failed to parse color set definition '%s': %v", def, err)
		}
	}
	return nil
}

// parsePlaces parses place definitions
func (p *CPNParser) parsePlaces(cpn *CPN, placeDefs []PlaceJSON) error {
	for _, placeDef := range placeDefs {
		// Get the color set
		colorSet, exists := p.colorSetParser.GetColorSet(placeDef.ColorSet)
		if !exists {
			return fmt.Errorf("unknown color set '%s' for place '%s'", placeDef.ColorSet, placeDef.Name)
		}

		// Create the place
		place := NewPlace(placeDef.ID, placeDef.Name, colorSet)
		if placeDef.Position != nil {
			place.Position = &Position{X: placeDef.Position.X, Y: placeDef.Position.Y}
		}
		cpn.AddPlace(place)
	}
	return nil
}

// parseTransitions parses transition definitions
func (p *CPNParser) parseTransitions(cpn *CPN, transitionDefs []TransitionJSON) error {
	for _, transitionDef := range transitionDefs {
		// Create the transition
		transition := NewTransition(transitionDef.ID, transitionDef.Name)

		// Set guard if provided
		if transitionDef.GuardExpression != "" {
			transition.SetGuard(transitionDef.GuardExpression, transitionDef.Variables)
		}

		// Set delay if provided
		if transitionDef.TransitionDelay > 0 {
			transition.SetDelay(transitionDef.TransitionDelay)
		}

		// Set kind if provided
		if transitionDef.Kind != "" {
			switch transitionDef.Kind {
			case "Auto":
				transition.SetKind(TransitionKindAuto)
			case "Manual":
				transition.SetKind(TransitionKindManual)
			default:
				return fmt.Errorf("unknown transition kind '%s' for transition '%s'", transitionDef.Kind, transitionDef.Name)
			}
		}

		if transitionDef.Position != nil {
			transition.Position = &Position{X: transitionDef.Position.X, Y: transitionDef.Position.Y}
		}

		cpn.AddTransition(transition)
	}
	return nil
}

// parseArcs parses arc definitions
func (p *CPNParser) parseArcs(cpn *CPN, arcDefs []ArcJSON) error {
	for _, arcDef := range arcDefs {
		// Determine direction
		var direction ArcDirection
		switch arcDef.Direction {
		case "IN":
			direction = ArcDirectionIn
		case "OUT":
			direction = ArcDirectionOut
		default:
			return fmt.Errorf("unknown arc direction '%s' for arc '%s'", arcDef.Direction, arcDef.ID)
		}

		// Create the arc
		arc := NewArc(arcDef.ID, arcDef.SourceID, arcDef.TargetID, arcDef.Expression, direction)
		cpn.AddArc(arc)
	}
	return nil
}

// parseInitialMarking parses initial marking definitions
func (p *CPNParser) parseInitialMarking(cpn *CPN, initialMarkingDef map[string][]TokenJSON) error {
	for placeName, tokenDefs := range initialMarkingDef {
		// Find the place to get its color set
		place := cpn.GetPlaceByName(placeName)
		if place == nil {
			return fmt.Errorf("unknown place '%s' in initial marking", placeName)
		}

		// Parse tokens
		var tokens []*Token
		for _, tokenDef := range tokenDefs {
			// Validate token value against place's color set
			if !place.ColorSet.IsMember(tokenDef.Value) {
				return fmt.Errorf("token value %v is not valid for color set %s in place %s",
					tokenDef.Value, place.ColorSet.Name(), placeName)
			}

			token := NewToken(tokenDef.Value, tokenDef.Timestamp)
			tokens = append(tokens, token)
		}

		cpn.SetInitialMarking(placeName, tokens)
	}
	return nil
}

// CPNToJSON converts a CPN to JSON format
func (p *CPNParser) CPNToJSON(cpn *CPN) ([]byte, error) {
	cpnDef := &CPNDefinitionJSON{
		ID:             cpn.ID,
		Name:           cpn.Name,
		Description:    cpn.Description,
		Places:         make([]PlaceJSON, len(cpn.Places)),
		Transitions:    make([]TransitionJSON, len(cpn.Transitions)),
		Arcs:           make([]ArcJSON, len(cpn.Arcs)),
		InitialMarking: make(map[string][]TokenJSON),
		EndPlaces:      cpn.EndPlaces,
	}

	// Convert places
	for i, place := range cpn.Places {
		cpnDef.Places[i] = PlaceJSON{
			ID:       place.ID,
			Name:     place.Name,
			ColorSet: place.ColorSet.Name(),
			Position: place.Position,
		}
	}

	// Convert transitions
	for i, transition := range cpn.Transitions {
		cpnDef.Transitions[i] = TransitionJSON{
			ID:              transition.ID,
			Name:            transition.Name,
			GuardExpression: transition.GuardExpression,
			Variables:       transition.Variables,
			TransitionDelay: transition.TransitionDelay,
			Kind:            string(transition.Kind),
			Position:        transition.Position,
		}
	}

	// Convert arcs
	for i, arc := range cpn.Arcs {
		cpnDef.Arcs[i] = ArcJSON{
			ID:         arc.ID,
			SourceID:   arc.SourceID,
			TargetID:   arc.TargetID,
			Expression: arc.Expression,
			Direction:  string(arc.Direction),
		}
	}

	// Convert initial marking
	for placeName, tokens := range cpn.InitialMarking {
		tokenDefs := make([]TokenJSON, len(tokens))
		for i, token := range tokens {
			tokenDefs[i] = TokenJSON{
				Value:     token.Value,
				Timestamp: token.Timestamp,
			}
		}
		cpnDef.InitialMarking[placeName] = tokenDefs
	}

	return json.MarshalIndent(cpnDef, "", "  ")
}

// GetColorSetParser returns the color set parser
func (p *CPNParser) GetColorSetParser() *ColorSetParser {
	return p.colorSetParser
}
