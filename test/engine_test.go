package test

import (
	"go-petri-flow/internal/engine"
	"go-petri-flow/internal/models"
	"testing"
)

func TestBasicTransitionFiring(t *testing.T) {
	// Create a simple CPN: Place1 -> Transition1 -> Place2
	cpn := models.NewCPN("test-cpn", "Test CPN", "A simple test CPN")

	// Create color set
	intCS := models.NewIntegerColorSet("INT", false)

	// Create places
	place1 := models.NewPlace("p1", "Place1", intCS)
	place2 := models.NewPlace("p2", "Place2", intCS)
	cpn.AddPlace(place1)
	cpn.AddPlace(place2)

	// Create transition
	transition := models.NewTransition("t1", "Transition1")
	cpn.AddTransition(transition)

	// Create arcs
	inputArc := models.NewInputArc("a1", "p1", "t1", "x")
	outputArc := models.NewOutputArc("a2", "t1", "p2", "x + 1")
	cpn.AddArc(inputArc)
	cpn.AddArc(outputArc)

	// Create initial marking
	marking := models.NewMarking()
	token := models.NewToken(5, 0)
	marking.AddToken("p1", token)

	// Create engine
	eng := engine.NewEngine()
	defer eng.Close()

	// Check if transition is enabled
	enabled, bindings, err := eng.IsEnabled(cpn, transition, marking)
	if err != nil {
		t.Fatalf("Failed to check if transition is enabled: %v", err)
	}
	if !enabled {
		t.Error("Transition should be enabled")
	}
	if len(bindings) == 0 {
		t.Error("Should have at least one binding")
	}

	// Fire the transition
	err = eng.FireTransition(cpn, transition, bindings[0], marking)
	if err != nil {
		t.Fatalf("Failed to fire transition: %v", err)
	}

	// Check the result
	if marking.HasTokens("p1") {
		t.Error("Place1 should be empty after firing")
	}
	if !marking.HasTokens("p2") {
		t.Error("Place2 should have tokens after firing")
	}
	tokens := marking.GetTokens("p2")
	if len(tokens) != 1 {
		t.Errorf("Expected 1 token in Place2, got %d", len(tokens))
	}
	if tokens[0].Value != 6 {
		t.Errorf("Expected token value 6, got %v", tokens[0].Value)
	}
}

func TestTransitionWithGuard(t *testing.T) {
	// Create CPN with guarded transition
	cpn := models.NewCPN("guarded-cpn", "Guarded CPN", "CPN with guard")

	intCS := models.NewIntegerColorSet("INT", false)

	place1 := models.NewPlace("p1", "Place1", intCS)
	place2 := models.NewPlace("p2", "Place2", intCS)
	cpn.AddPlace(place1)
	cpn.AddPlace(place2)

	// Transition with guard x > 10
	transition := models.NewTransition("t1", "Transition1")
	transition.SetGuard("x > 10", []string{"x"})
	cpn.AddTransition(transition)

	inputArc := models.NewInputArc("a1", "p1", "t1", "x")
	outputArc := models.NewOutputArc("a2", "t1", "p2", "x")
	cpn.AddArc(inputArc)
	cpn.AddArc(outputArc)

	eng := engine.NewEngine()
	defer eng.Close()

	// Test with token value 5 (should not be enabled)
	marking1 := models.NewMarking()
	token1 := models.NewToken(5, 0)
	marking1.AddToken("p1", token1)

	enabled, _, err := eng.IsEnabled(cpn, transition, marking1)
	if err != nil {
		t.Fatalf("Failed to check if transition is enabled: %v", err)
	}
	if enabled {
		t.Error("Transition should not be enabled with token value 5")
	}

	// Test with token value 15 (should be enabled)
	marking2 := models.NewMarking()
	token2 := models.NewToken(15, 0)
	marking2.AddToken("p1", token2)

	enabled, bindings, err := eng.IsEnabled(cpn, transition, marking2)
	if err != nil {
		t.Fatalf("Failed to check if transition is enabled: %v", err)
	}
	if !enabled {
		t.Error("Transition should be enabled with token value 15")
	}

	// Fire the transition
	err = eng.FireTransition(cpn, transition, bindings[0], marking2)
	if err != nil {
		t.Fatalf("Failed to fire transition: %v", err)
	}

	// Check result
	tokens := marking2.GetTokens("p2")
	if len(tokens) != 1 || tokens[0].Value != 15 {
		t.Errorf("Expected token value 15 in Place2, got %v", tokens)
	}
}

func TestTransitionDelay(t *testing.T) {
	// Create CPN with delayed transition
	cpn := models.NewCPN("delayed-cpn", "Delayed CPN", "CPN with delay")

	intCS := models.NewIntegerColorSet("INT", false)

	place1 := models.NewPlace("p1", "Place1", intCS)
	place2 := models.NewPlace("p2", "Place2", intCS)
	cpn.AddPlace(place1)
	cpn.AddPlace(place2)

	// Transition with delay 5
	transition := models.NewTransition("t1", "Transition1")
	transition.SetDelay(5)
	cpn.AddTransition(transition)

	inputArc := models.NewInputArc("a1", "p1", "t1", "x")
	outputArc := models.NewOutputArc("a2", "t1", "p2", "x")
	cpn.AddArc(inputArc)
	cpn.AddArc(outputArc)

	// Create marking with global clock 10
	marking := models.NewMarkingWithClock(10)
	token := models.NewToken(42, 0)
	marking.AddToken("p1", token)

	eng := engine.NewEngine()
	defer eng.Close()

	// Fire the transition
	enabled, bindings, err := eng.IsEnabled(cpn, transition, marking)
	if err != nil {
		t.Fatalf("Failed to check if transition is enabled: %v", err)
	}
	if !enabled {
		t.Error("Transition should be enabled")
	}

	err = eng.FireTransition(cpn, transition, bindings[0], marking)
	if err != nil {
		t.Fatalf("Failed to fire transition: %v", err)
	}

	// Check that global clock advanced
	if marking.GlobalClock != 15 {
		t.Errorf("Expected global clock 15, got %d", marking.GlobalClock)
	}

	// Check that token was produced
	tokens := marking.GetTokens("p2")
	if len(tokens) != 1 || tokens[0].Value != 42 {
		t.Errorf("Expected token value 42 in Place2, got %v", tokens)
	}
}

func TestMultipleEnabledTransitions(t *testing.T) {
	// Create CPN with multiple transitions
	cpn := models.NewCPN("multi-cpn", "Multi CPN", "CPN with multiple transitions")

	intCS := models.NewIntegerColorSet("INT", false)

	place1 := models.NewPlace("p1", "Place1", intCS)
	place2 := models.NewPlace("p2", "Place2", intCS)
	place3 := models.NewPlace("p3", "Place3", intCS)
	cpn.AddPlace(place1)
	cpn.AddPlace(place2)
	cpn.AddPlace(place3)

	// Two transitions from Place1
	transition1 := models.NewTransition("t1", "Transition1")
	transition2 := models.NewTransition("t2", "Transition2")
	cpn.AddTransition(transition1)
	cpn.AddTransition(transition2)

	// Arcs for transition1: Place1 -> Place2
	inputArc1 := models.NewInputArc("a1", "p1", "t1", "x")
	outputArc1 := models.NewOutputArc("a2", "t1", "p2", "x * 2")
	cpn.AddArc(inputArc1)
	cpn.AddArc(outputArc1)

	// Arcs for transition2: Place1 -> Place3
	inputArc2 := models.NewInputArc("a3", "p1", "t2", "x")
	outputArc2 := models.NewOutputArc("a4", "t2", "p3", "x + 10")
	cpn.AddArc(inputArc2)
	cpn.AddArc(outputArc2)

	// Create marking
	marking := models.NewMarking()
	token := models.NewToken(5, 0)
	marking.AddToken("p1", token)

	eng := engine.NewEngine()
	defer eng.Close()

	// Get enabled transitions
	enabledTransitions, bindingsMap, err := eng.GetEnabledTransitions(cpn, marking)
	if err != nil {
		t.Fatalf("Failed to get enabled transitions: %v", err)
	}

	if len(enabledTransitions) != 2 {
		t.Errorf("Expected 2 enabled transitions, got %d", len(enabledTransitions))
	}

	// Fire one transition
	transition := enabledTransitions[0]
	bindings := bindingsMap[transition.ID]

	err = eng.FireTransition(cpn, transition, bindings[0], marking)
	if err != nil {
		t.Fatalf("Failed to fire transition: %v", err)
	}

	// Check that only one transition fired and Place1 is empty
	if marking.HasTokens("p1") {
		t.Error("Place1 should be empty after firing")
	}

	// Check that exactly one of Place2 or Place3 has tokens
	hasPlace2 := marking.HasTokens("p2")
	hasPlace3 := marking.HasTokens("p3")

	if hasPlace2 && hasPlace3 {
		t.Error("Only one of Place2 or Place3 should have tokens")
	}
	if !hasPlace2 && !hasPlace3 {
		t.Error("One of Place2 or Place3 should have tokens")
	}
}

func TestManualTransition(t *testing.T) {
	// Create CPN with manual transition
	cpn := models.NewCPN("manual-cpn", "Manual CPN", "CPN with manual transition")

	intCS := models.NewIntegerColorSet("INT", false)

	place1 := models.NewPlace("p1", "Place1", intCS)
	place2 := models.NewPlace("p2", "Place2", intCS)
	cpn.AddPlace(place1)
	cpn.AddPlace(place2)

	// Manual transition
	transition := models.NewTransition("t1", "ManualTransition")
	transition.SetKind(models.TransitionKindManual)
	cpn.AddTransition(transition)

	inputArc := models.NewInputArc("a1", "p1", "t1", "x")
	outputArc := models.NewOutputArc("a2", "t1", "p2", "x")
	cpn.AddArc(inputArc)
	cpn.AddArc(outputArc)

	// Create marking
	marking := models.NewMarking()
	token := models.NewToken(42, 0)
	marking.AddToken("p1", token)

	eng := engine.NewEngine()
	defer eng.Close()

	// Check that automatic firing doesn't fire manual transitions
	firedCount, err := eng.FireEnabledTransitions(cpn, marking)
	if err != nil {
		t.Fatalf("Failed to fire enabled transitions: %v", err)
	}
	if firedCount != 0 {
		t.Errorf("Expected 0 automatic transitions fired, got %d", firedCount)
	}

	// Check that manual transition is still enabled
	manualTransitions, bindingsMap, err := eng.GetManualTransitions(cpn, marking)
	if err != nil {
		t.Fatalf("Failed to get manual transitions: %v", err)
	}
	if len(manualTransitions) != 1 {
		t.Errorf("Expected 1 manual transition, got %d", len(manualTransitions))
	}

	// Manually fire the transition
	bindings := bindingsMap[transition.ID]
	err = eng.FireTransition(cpn, transition, bindings[0], marking)
	if err != nil {
		t.Fatalf("Failed to manually fire transition: %v", err)
	}

	// Check result
	if marking.HasTokens("p1") {
		t.Error("Place1 should be empty after manual firing")
	}
	if !marking.HasTokens("p2") {
		t.Error("Place2 should have tokens after manual firing")
	}
}

func TestSimulationStep(t *testing.T) {
	// Create a chain of automatic transitions
	cpn := models.NewCPN("chain-cpn", "Chain CPN", "CPN with chained transitions")

	intCS := models.NewIntegerColorSet("INT", false)

	// Places: P1 -> T1 -> P2 -> T2 -> P3
	place1 := models.NewPlace("p1", "Place1", intCS)
	place2 := models.NewPlace("p2", "Place2", intCS)
	place3 := models.NewPlace("p3", "Place3", intCS)
	cpn.AddPlace(place1)
	cpn.AddPlace(place2)
	cpn.AddPlace(place3)

	transition1 := models.NewTransition("t1", "Transition1")
	transition2 := models.NewTransition("t2", "Transition2")
	cpn.AddTransition(transition1)
	cpn.AddTransition(transition2)

	// T1: P1 -> P2
	cpn.AddArc(models.NewInputArc("a1", "p1", "t1", "x"))
	cpn.AddArc(models.NewOutputArc("a2", "t1", "p2", "x + 1"))

	// T2: P2 -> P3
	cpn.AddArc(models.NewInputArc("a3", "p2", "t2", "x"))
	cpn.AddArc(models.NewOutputArc("a4", "t2", "p3", "x + 1"))

	// Initial marking
	marking := models.NewMarking()
	token := models.NewToken(1, 0)
	marking.AddToken("p1", token)

	eng := engine.NewEngine()
	defer eng.Close()

	// Perform first simulation step (layer 1) - only first transition fires
	firedCount, err := eng.SimulateStep(cpn, marking)
	if err != nil {
		t.Fatalf("Failed to simulate first step: %v", err)
	}
	if firedCount != 1 {
		t.Errorf("Expected 1 transition fired in first step, got %d", firedCount)
	}
	// After first step: Place1 empty, Place2 has token (value 2), Place3 empty
	if marking.HasTokens("p1") == true {
		t.Error("Place1 should be empty after first step")
	}
	if !marking.HasTokens("p2") {
		t.Error("Place2 should have token after first step")
	}
	if marking.HasTokens("p3") {
		t.Error("Place3 should be empty after first step")
	}

	// Perform second simulation step (layer 2) - second transition fires
	secondFired, err := eng.SimulateStep(cpn, marking)
	if err != nil {
		t.Fatalf("Failed to simulate second step: %v", err)
	}
	if secondFired != 1 {
		t.Errorf("Expected 1 transition fired in second step, got %d", secondFired)
	}

	// Final state after two steps
	if marking.HasTokens("p1") || marking.HasTokens("p2") {
		t.Error("Place1 and Place2 should be empty after second step")
	}
	if !marking.HasTokens("p3") {
		t.Error("Place3 should have tokens after second step")
	}
	tokens := marking.GetTokens("p3")
	if len(tokens) != 1 || tokens[0].Value != 3 {
		t.Errorf("Expected token value 3 in Place3 after two steps, got %v", tokens)
	}
}

func TestCPNCompletion(t *testing.T) {
	// Create CPN with end places
	cpn := models.NewCPN("completion-cpn", "Completion CPN", "CPN with end places")

	intCS := models.NewIntegerColorSet("INT", false)

	place1 := models.NewPlace("p1", "Place1", intCS)
	place2 := models.NewPlace("p2", "EndPlace", intCS)
	cpn.AddPlace(place1)
	cpn.AddPlace(place2)

	transition := models.NewTransition("t1", "Transition1")
	cpn.AddTransition(transition)

	cpn.AddArc(models.NewInputArc("a1", "p1", "t1", "x"))
	cpn.AddArc(models.NewOutputArc("a2", "t1", "p2", "x"))

	// Set end places
	cpn.SetEndPlaces([]string{"EndPlace"})

	// Initial marking
	marking := models.NewMarking()
	token := models.NewToken(42, 0)
	marking.AddToken("p1", token)

	eng := engine.NewEngine()
	defer eng.Close()

	// Check not completed initially
	if eng.IsCompleted(cpn, marking) {
		t.Error("CPN should not be completed initially")
	}

	// Fire transition
	enabled, bindings, err := eng.IsEnabled(cpn, transition, marking)
	if err != nil {
		t.Fatalf("Failed to check if transition is enabled: %v", err)
	}
	if !enabled {
		t.Error("Transition should be enabled")
	}

	err = eng.FireTransition(cpn, transition, bindings[0], marking)
	if err != nil {
		t.Fatalf("Failed to fire transition: %v", err)
	}

	// Check completed after firing
	if !eng.IsCompleted(cpn, marking) {
		t.Error("CPN should be completed after firing")
	}
}
