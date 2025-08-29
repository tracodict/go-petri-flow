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
	InitialMarking map[string][]TokenJSON `json:"initialMarking,omitempty"` // Keys: place IDs (preferred) or legacy place names
	EndPlaces      []string               `json:"endPlaces,omitempty"`
	SubWorkflows   []SubWorkflowJSON      `json:"subWorkflows,omitempty"`
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
		// Preserve original for round-trip
		p.colorSetParser.StoreOriginalJsonSchema(d.Name, d.Schema)
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
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	GuardExpression  string    `json:"guardExpression,omitempty"`
	Variables        []string  `json:"variables,omitempty"`
	TransitionDelay  int       `json:"transitionDelay,omitempty"`
	Kind             string    `json:"kind,omitempty"` // "Auto" or "Manual"
	Position         *Position `json:"position,omitempty"`
	ActionExpression string    `json:"actionExpression,omitempty"`
	FormSchema       string    `json:"formSchema,omitempty"`
	LayoutSchema     string    `json:"layoutSchema,omitempty"`
}

// ArcJSON represents the JSON structure for arcs
type ArcJSON struct {
	ID           string `json:"id"`
	SourceID     string `json:"sourceId"`
	TargetID     string `json:"targetId"`
	Expression   string `json:"expression"`
	Direction    string `json:"direction"` // "IN" or "OUT"
	Multiplicity int    `json:"multiplicity,omitempty"`
}

// SubWorkflowJSON represents JSON structure for a hierarchical link
type SubWorkflowJSON struct {
	ID                  string            `json:"id"`
	CPNID               string            `json:"cpnId"`
	CallTransitionID    string            `json:"callTransitionId"`
	AutoStart           bool              `json:"autoStart"`
	PropagateOnComplete bool              `json:"propagateOnComplete"`
	InputMapping        map[string]string `json:"inputMapping,omitempty"`
	OutputMapping       map[string]string `json:"outputMapping,omitempty"`
}

// TokenJSON represents the JSON structure for tokens
type TokenJSON struct {
	Value     interface{} `json:"value"`
	Timestamp int         `json:"timestamp"`
	Count     int         `json:"count,omitempty"` // Optional multiplicity shorthand (>=1)
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

	// Set end places (preserve as provided for backward compatibility; IsCompleted handles name or ID)
	if len(cpnDef.EndPlaces) > 0 {
		cpn.SetEndPlaces(cpnDef.EndPlaces)
	}

	// Parse sub workflows
	if err := p.parseSubWorkflows(cpn, cpnDef.SubWorkflows); err != nil {
		return nil, fmt.Errorf("failed to parse subWorkflows: %v", err)
	}

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
			case "Message":
				transition.SetKind(TransitionKindMessage)
			case "LLM":
				transition.SetKind(TransitionKindLLM)
			default:
				return fmt.Errorf("unknown transition kind '%s' for transition '%s'", transitionDef.Kind, transitionDef.Name)
			}
		}

		// Set action if provided
		if transitionDef.ActionExpression != "" {
			transition.SetAction(transitionDef.ActionExpression)
		}

		// Set form/layout schemas if provided (Manual transitions only, but we can store regardless)
		if transitionDef.FormSchema != "" {
			transition.FormSchema = transitionDef.FormSchema
		}
		if transitionDef.LayoutSchema != "" {
			transition.LayoutSchema = transitionDef.LayoutSchema
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
		if arcDef.Multiplicity > 0 {
			arc.Multiplicity = arcDef.Multiplicity
		}
		cpn.AddArc(arc)
	}
	return nil
}

// parseInitialMarking parses initial marking definitions
func (p *CPNParser) parseInitialMarking(cpn *CPN, initialMarkingDef map[string][]TokenJSON) error {
	for key, tokenDefs := range initialMarkingDef {
		// First try key as place ID
		place := cpn.GetPlace(key)
		legacyName := false
		if place == nil { // fallback to legacy name
			place = cpn.GetPlaceByName(key)
			legacyName = place != nil
		}
		if place == nil {
			return fmt.Errorf("unknown place identifier '%s' in initial marking", key)
		}

		// Parse tokens (expanding Count shorthand)
		var tokens []*Token
		for _, tokenDef := range tokenDefs {
			count := tokenDef.Count
			if count <= 0 { // default to 1 when omitted
				count = 1
			}
			// Validate once
			if !place.ColorSet.IsMember(tokenDef.Value) {
				return fmt.Errorf("token value %v is not valid for color set %s in place %s",
					tokenDef.Value, place.ColorSet.Name(), place.Name)
			}
			for i := 0; i < count; i++ {
				token := NewToken(tokenDef.Value, tokenDef.Timestamp)
				tokens = append(tokens, token)
			}
		}

		// Always store by place ID internally
		cpn.SetInitialMarking(place.ID, tokens)
		_ = legacyName // placeholder; potential warning hook in future
	}
	return nil
}

// CPNToJSON converts a CPN to JSON format
func (p *CPNParser) CPNToJSON(cpn *CPN) ([]byte, error) {
	cpnDef := &CPNDefinitionJSON{
		ID:             cpn.ID,
		Name:           cpn.Name,
		Description:    cpn.Description,
		ColorSets:      []string{}, // reconstruct color set declarations (best-effort)
		JsonSchemas:    []JsonSchemaDef{},
		Places:         make([]PlaceJSON, len(cpn.Places)),
		Transitions:    make([]TransitionJSON, len(cpn.Transitions)),
		Arcs:           make([]ArcJSON, len(cpn.Arcs)),
		InitialMarking: make(map[string][]TokenJSON),
		EndPlaces:      cpn.EndPlaces,
		SubWorkflows:   make([]SubWorkflowJSON, len(cpn.SubWorkflows)),
	}

	// Use preserved original definitions if available
	if p.colorSetParser != nil {
		cpnDef.ColorSets = p.colorSetParser.GetOriginalColorSetDefinitions()
		cpnDef.JsonSchemas = p.colorSetParser.GetOriginalJsonSchemas()
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
			ID:               transition.ID,
			Name:             transition.Name,
			GuardExpression:  transition.GuardExpression,
			Variables:        transition.Variables,
			TransitionDelay:  transition.TransitionDelay,
			Kind:             string(transition.Kind),
			Position:         transition.Position,
			ActionExpression: transition.ActionExpression,
			FormSchema:       transition.FormSchema,
			LayoutSchema:     transition.LayoutSchema,
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

	// Convert initial marking (ids as keys)
	for placeID, tokens := range cpn.InitialMarking {
		tokenDefs := make([]TokenJSON, len(tokens))
		for i, token := range tokens {
			tokenDefs[i] = TokenJSON{Value: token.Value, Timestamp: token.Timestamp}
		}
		cpnDef.InitialMarking[placeID] = tokenDefs
	}

	// Convert sub workflows
	for i, sw := range cpn.SubWorkflows {
		if sw == nil {
			continue
		}
		cpnDef.SubWorkflows[i] = SubWorkflowJSON{
			ID:                  sw.ID,
			CPNID:               sw.CPNID,
			CallTransitionID:    sw.CallTransitionID,
			AutoStart:           sw.AutoStart,
			PropagateOnComplete: sw.PropagateOnComplete,
			InputMapping:        sw.InputMapping,
			OutputMapping:       sw.OutputMapping,
		}
	}

	return json.MarshalIndent(cpnDef, "", "  ")
}

// GetColorSetParser returns the color set parser
func (p *CPNParser) GetColorSetParser() *ColorSetParser {
	return p.colorSetParser
}

// parseSubWorkflows loads sub workflow links
func (p *CPNParser) parseSubWorkflows(cpn *CPN, defs []SubWorkflowJSON) error {
	for _, d := range defs {
		// Validate transition exists
		if cpn.GetTransition(d.CallTransitionID) == nil {
			return fmt.Errorf("subWorkflow %s references unknown transition %s", d.ID, d.CallTransitionID)
		}
		sw := &SubWorkflowLink{
			ID:                  d.ID,
			CPNID:               d.CPNID,
			CallTransitionID:    d.CallTransitionID,
			AutoStart:           d.AutoStart,
			PropagateOnComplete: d.PropagateOnComplete,
			InputMapping:        d.InputMapping,
			OutputMapping:       d.OutputMapping,
		}
		cpn.SubWorkflows = append(cpn.SubWorkflows, sw)
	}
	return nil
}
