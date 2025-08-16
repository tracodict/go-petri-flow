package test

import (
	"go-petri-flow/internal/models"
	"testing"
)

func TestColorSetParser(t *testing.T) {
	parser := models.NewColorSetParser()

	// Test basic integer color set
	cs, err := parser.ParseColorSetDefinition("colset MyInt = int;")
	if err != nil {
		t.Fatalf("Failed to parse integer color set: %v", err)
	}
	if cs.Name() != "MyInt" {
		t.Errorf("Expected name 'MyInt', got %s", cs.Name())
	}
	if cs.IsTimed() {
		t.Error("Color set should not be timed")
	}

	// Test timed string color set
	cs, err = parser.ParseColorSetDefinition("colset MyString = string timed;")
	if err != nil {
		t.Fatalf("Failed to parse timed string color set: %v", err)
	}
	if !cs.IsTimed() {
		t.Error("Color set should be timed")
	}

	// Test integer range
	cs, err = parser.ParseColorSetDefinition("colset Range = int[1..10];")
	if err != nil {
		t.Fatalf("Failed to parse integer range: %v", err)
	}
	if !cs.IsMember(5) {
		t.Error("5 should be a member of Range")
	}
	if cs.IsMember(15) {
		t.Error("15 should not be a member of Range")
	}

	// Test enumerated color set
	cs, err = parser.ParseColorSetDefinition("colset Color = with red | green | blue;")
	if err != nil {
		t.Fatalf("Failed to parse enumerated color set: %v", err)
	}
	if !cs.IsMember("red") {
		t.Error("'red' should be a member of Color")
	}
	if cs.IsMember("yellow") {
		t.Error("'yellow' should not be a member of Color")
	}

	// Test product color set
	cs, err = parser.ParseColorSetDefinition("colset Pair = product INT * STRING;")
	if err != nil {
		t.Fatalf("Failed to parse product color set: %v", err)
	}
	if !cs.IsMember([]interface{}{42, "hello"}) {
		t.Error("[42, 'hello'] should be a member of Pair")
	}
	if cs.IsMember([]interface{}{42}) {
		t.Error("[42] should not be a member of Pair (wrong length)")
	}
}

func TestCPNParser(t *testing.T) {
	parser := models.NewCPNParser()

	// Test simple CPN JSON
	jsonData := `{
		"id": "simple-cpn",
		"name": "Simple CPN",
		"description": "A simple test CPN",
		"colorSets": [
			"colset MyInt = int;"
		],
		"places": [
			{
				"id": "p1",
				"name": "Place1",
				"colorSet": "MyInt"
			},
			{
				"id": "p2",
				"name": "Place2",
				"colorSet": "MyInt"
			}
		],
		"transitions": [
			{
				"id": "t1",
				"name": "Transition1",
				"guardExpression": "x > 0",
				"variables": ["x"],
				"kind": "Auto"
			}
		],
		"arcs": [
			{
				"id": "a1",
				"sourceId": "p1",
				"targetId": "t1",
				"expression": "x",
				"direction": "IN"
			},
			{
				"id": "a2",
				"sourceId": "t1",
				"targetId": "p2",
				"expression": "x + 1",
				"direction": "OUT"
			}
		],
		"initialMarking": {
			"Place1": [
				{
					"value": 1,
					"timestamp": 0
				}
			]
		},
		"endPlaces": ["Place2"]
	}`

	cpn, err := parser.ParseCPNFromJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to parse CPN from JSON: %v", err)
	}

	// Verify CPN structure
	if cpn.ID != "simple-cpn" {
		t.Errorf("Expected ID 'simple-cpn', got %s", cpn.ID)
	}
	if cpn.Name != "Simple CPN" {
		t.Errorf("Expected name 'Simple CPN', got %s", cpn.Name)
	}

	// Check places
	if len(cpn.Places) != 2 {
		t.Errorf("Expected 2 places, got %d", len(cpn.Places))
	}
	place1 := cpn.GetPlace("p1")
	if place1 == nil {
		t.Error("Place p1 should exist")
	} else {
		if place1.Name != "Place1" {
			t.Errorf("Expected place name 'Place1', got %s", place1.Name)
		}
		if place1.ColorSet.Name() != "MyInt" {
			t.Errorf("Expected color set 'MyInt', got %s", place1.ColorSet.Name())
		}
	}

	// Check transitions
	if len(cpn.Transitions) != 1 {
		t.Errorf("Expected 1 transition, got %d", len(cpn.Transitions))
	}
	transition1 := cpn.GetTransition("t1")
	if transition1 == nil {
		t.Error("Transition t1 should exist")
	} else {
		if transition1.GuardExpression != "x > 0" {
			t.Errorf("Expected guard 'x > 0', got %s", transition1.GuardExpression)
		}
		if len(transition1.Variables) != 1 || transition1.Variables[0] != "x" {
			t.Errorf("Expected variables ['x'], got %v", transition1.Variables)
		}
		if !transition1.IsAuto() {
			t.Error("Transition should be automatic")
		}
	}

	// Check arcs
	if len(cpn.Arcs) != 2 {
		t.Errorf("Expected 2 arcs, got %d", len(cpn.Arcs))
	}
	inputArcs := cpn.GetInputArcs("t1")
	if len(inputArcs) != 1 {
		t.Errorf("Expected 1 input arc, got %d", len(inputArcs))
	}
	outputArcs := cpn.GetOutputArcs("t1")
	if len(outputArcs) != 1 {
		t.Errorf("Expected 1 output arc, got %d", len(outputArcs))
	}

	// Check initial marking
	if len(cpn.InitialMarking) != 1 {
		t.Errorf("Expected 1 place in initial marking, got %d", len(cpn.InitialMarking))
	}
	tokens, exists := cpn.InitialMarking["Place1"]
	if !exists {
		t.Error("Place1 should have initial tokens")
	} else {
		if len(tokens) != 1 {
			t.Errorf("Expected 1 initial token, got %d", len(tokens))
		}
		if tokens[0].Value != 1.0 { // JSON unmarshaling converts numbers to float64
			t.Errorf("Expected token value 1, got %v", tokens[0].Value)
		}
	}

	// Check end places
	if len(cpn.EndPlaces) != 1 || cpn.EndPlaces[0] != "Place2" {
		t.Errorf("Expected end places ['Place2'], got %v", cpn.EndPlaces)
	}

	// Test creating initial marking
	marking := cpn.CreateInitialMarking()
	if !marking.HasTokens("Place1") {
		t.Error("Initial marking should have tokens in Place1")
	}
	if marking.CountTokens("Place1") != 1 {
		t.Errorf("Expected 1 token in Place1, got %d", marking.CountTokens("Place1"))
	}
}

func TestCPNToJSON(t *testing.T) {
	parser := models.NewCPNParser()

	// Create a simple CPN
	cpn := models.NewCPN("test-cpn", "Test CPN", "A test CPN")

	// Add color set to parser
	intCS := models.NewIntegerColorSet("TestInt", false)
	parser.GetColorSetParser().RegisterColorSet(intCS)

	// Add places
	place1 := models.NewPlace("p1", "Place1", intCS)
	place2 := models.NewPlace("p2", "Place2", intCS)
	cpn.AddPlace(place1)
	cpn.AddPlace(place2)

	// Add transition
	transition := models.NewTransition("t1", "Transition1")
	transition.SetGuard("x > 0", []string{"x"})
	cpn.AddTransition(transition)

	// Add arcs
	inputArc := models.NewInputArc("a1", "p1", "t1", "x")
	outputArc := models.NewOutputArc("a2", "t1", "p2", "x")
	cpn.AddArc(inputArc)
	cpn.AddArc(outputArc)

	// Add initial marking
	token := models.NewToken(5, 0)
	cpn.AddInitialToken("Place1", token)

	// Set end places
	cpn.SetEndPlaces([]string{"Place2"})

	// Convert to JSON
	jsonData, err := parser.CPNToJSON(cpn)
	if err != nil {
		t.Fatalf("Failed to convert CPN to JSON: %v", err)
	}

	// Parse it back
	cpn2, err := parser.ParseCPNFromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to parse CPN back from JSON: %v", err)
	}

	// Verify it's the same
	if cpn2.ID != cpn.ID {
		t.Errorf("ID mismatch: expected %s, got %s", cpn.ID, cpn2.ID)
	}
	if cpn2.Name != cpn.Name {
		t.Errorf("Name mismatch: expected %s, got %s", cpn.Name, cpn2.Name)
	}
	if len(cpn2.Places) != len(cpn.Places) {
		t.Errorf("Places count mismatch: expected %d, got %d", len(cpn.Places), len(cpn2.Places))
	}
	if len(cpn2.Transitions) != len(cpn.Transitions) {
		t.Errorf("Transitions count mismatch: expected %d, got %d", len(cpn.Transitions), len(cpn2.Transitions))
	}
	if len(cpn2.Arcs) != len(cpn.Arcs) {
		t.Errorf("Arcs count mismatch: expected %d, got %d", len(cpn.Arcs), len(cpn2.Arcs))
	}
}

func TestMultipleColorSetDefinitions(t *testing.T) {
	parser := models.NewColorSetParser()

	definitions := `
		colset MyInt = int;
		colset MyString = string timed;
		colset Color = with red | green | blue;
		colset Pair = product MyInt * MyString;
	`

	colorSets, err := parser.ParseMultipleDefinitions(definitions)
	if err != nil {
		t.Fatalf("Failed to parse multiple definitions: %v", err)
	}

	if len(colorSets) != 4 {
		t.Errorf("Expected 4 color sets, got %d", len(colorSets))
	}

	// Check that the product color set can reference previously defined color sets
	pairCS := colorSets[3]
	if pairCS.Name() != "Pair" {
		t.Errorf("Expected name 'Pair', got %s", pairCS.Name())
	}

	// Test that the product color set works correctly
	if !pairCS.IsMember([]interface{}{42, "hello"}) {
		t.Error("Pair should accept [42, 'hello']")
	}
}
