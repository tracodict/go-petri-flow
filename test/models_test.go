package test

import (
	"go-petri-flow/internal/models"
	"testing"
)

func TestToken(t *testing.T) {
	// Test token creation
	token := models.NewToken("test", 10)
	if token.Value != "test" {
		t.Errorf("Expected value 'test', got %v", token.Value)
	}
	if token.Timestamp != 10 {
		t.Errorf("Expected timestamp 10, got %d", token.Timestamp)
	}

	// Test token equality
	token2 := models.NewToken("test", 10)
	if !token.Equals(token2) {
		t.Error("Tokens should be equal")
	}

	token3 := models.NewToken("test", 20)
	if token.Equals(token3) {
		t.Error("Tokens should not be equal (different timestamp)")
	}

	// Test value string
	intToken := models.NewToken(42, 0)
	if intToken.ValueString() != "42" {
		t.Errorf("Expected '42', got %s", intToken.ValueString())
	}
}

func TestMultiset(t *testing.T) {
	ms := models.NewMultiset()

	// Test empty multiset
	if !ms.IsEmpty() {
		t.Error("New multiset should be empty")
	}
	if ms.Size() != 0 {
		t.Errorf("Expected size 0, got %d", ms.Size())
	}

	// Test adding tokens
	token1 := models.NewToken("a", 0)
	token2 := models.NewToken("a", 5)
	token3 := models.NewToken("b", 0)

	ms.Add(token1)
	ms.Add(token2)
	ms.Add(token3)

	if ms.Size() != 3 {
		t.Errorf("Expected size 3, got %d", ms.Size())
	}

	// Test contains
	if !ms.Contains("a") {
		t.Error("Multiset should contain 'a'")
	}
	if !ms.Contains("b") {
		t.Error("Multiset should contain 'b'")
	}
	if ms.Contains("c") {
		t.Error("Multiset should not contain 'c'")
	}

	// Test count
	if ms.Count("a") != 2 {
		t.Errorf("Expected count 2 for 'a', got %d", ms.Count("a"))
	}
	if ms.Count("b") != 1 {
		t.Errorf("Expected count 1 for 'b', got %d", ms.Count("b"))
	}

	// Test remove
	if !ms.Remove(token1) {
		t.Error("Should be able to remove token1")
	}
	if ms.Count("a") != 1 {
		t.Errorf("Expected count 1 for 'a' after removal, got %d", ms.Count("a"))
	}

	// Test remove by value
	removedToken := ms.RemoveByValue("a")
	if removedToken == nil {
		t.Error("Should be able to remove token by value")
	}
	if ms.Count("a") != 0 {
		t.Errorf("Expected count 0 for 'a' after removal, got %d", ms.Count("a"))
	}
}

func TestMarking(t *testing.T) {
	marking := models.NewMarking()

	// Test empty marking
	if !marking.IsEmpty() {
		t.Error("New marking should be empty")
	}

	// Test adding tokens
	token1 := models.NewToken("x", 0)
	token2 := models.NewToken("y", 5)

	marking.AddToken("place1", token1)
	marking.AddToken("place2", token2)

	if marking.IsEmpty() {
		t.Error("Marking should not be empty after adding tokens")
	}

	// Test has tokens
	if !marking.HasTokens("place1") {
		t.Error("place1 should have tokens")
	}
	if marking.HasTokens("place3") {
		t.Error("place3 should not have tokens")
	}

	// Test count tokens
	if marking.CountTokens("place1") != 1 {
		t.Errorf("Expected 1 token in place1, got %d", marking.CountTokens("place1"))
	}

	// Test global clock
	if marking.GlobalClock != 0 {
		t.Errorf("Expected global clock 0, got %d", marking.GlobalClock)
	}

	marking.AdvanceGlobalClock(10)
	if marking.GlobalClock != 10 {
		t.Errorf("Expected global clock 10, got %d", marking.GlobalClock)
	}

	// Test earliest timestamp
	earliest := marking.GetEarliestTimestamp()
	if earliest != 0 {
		t.Errorf("Expected earliest timestamp 0, got %d", earliest)
	}
}

func TestColorSets(t *testing.T) {
	// Test integer color set
	intCS := models.NewIntegerColorSet("INT", false)
	if !intCS.IsMember(42) {
		t.Error("42 should be a member of INT")
	}
	if intCS.IsMember("hello") {
		t.Error("'hello' should not be a member of INT")
	}
	if intCS.IsTimed() {
		t.Error("INT should not be timed")
	}

	// Test string color set
	stringCS := models.NewStringColorSet("STRING", true)
	if !stringCS.IsMember("hello") {
		t.Error("'hello' should be a member of STRING")
	}
	if stringCS.IsMember(42) {
		t.Error("42 should not be a member of STRING")
	}
	if !stringCS.IsTimed() {
		t.Error("STRING should be timed")
	}

	// Test enumerated color set
	enumCS := models.NewEnumeratedColorSet("COLOR", false, []string{"red", "green", "blue"})
	if !enumCS.IsMember("red") {
		t.Error("'red' should be a member of COLOR")
	}
	if enumCS.IsMember("yellow") {
		t.Error("'yellow' should not be a member of COLOR")
	}
}

func TestPlace(t *testing.T) {
	intCS := models.NewIntegerColorSet("INT", false)
	place := models.NewPlace("p1", "Place1", intCS)

	if place.ID != "p1" {
		t.Errorf("Expected ID 'p1', got %s", place.ID)
	}
	if place.Name != "Place1" {
		t.Errorf("Expected name 'Place1', got %s", place.Name)
	}

	// Test token validation
	validToken := models.NewToken(42, 0)
	if err := place.ValidateToken(validToken); err != nil {
		t.Errorf("Valid token should pass validation: %v", err)
	}

	invalidToken := models.NewToken("hello", 0)
	if err := place.ValidateToken(invalidToken); err == nil {
		t.Error("Invalid token should fail validation")
	}
}

func TestTransition(t *testing.T) {
	transition := models.NewTransition("t1", "Transition1")

	if transition.ID != "t1" {
		t.Errorf("Expected ID 't1', got %s", transition.ID)
	}
	if transition.Name != "Transition1" {
		t.Errorf("Expected name 'Transition1', got %s", transition.Name)
	}
	if !transition.IsAuto() {
		t.Error("Transition should be automatic by default")
	}
	if transition.HasGuard() {
		t.Error("Transition should not have guard by default")
	}

	// Test setting guard
	transition.SetGuard("x > 0", []string{"x"})
	if !transition.HasGuard() {
		t.Error("Transition should have guard after setting")
	}

	// Test setting kind
	transition.SetKind(models.TransitionKindManual)
	if !transition.IsManual() {
		t.Error("Transition should be manual after setting")
	}
}

func TestArc(t *testing.T) {
	inputArc := models.NewInputArc("a1", "p1", "t1", "x")
	if !inputArc.IsInputArc() {
		t.Error("Should be input arc")
	}
	if inputArc.GetPlaceID() != "p1" {
		t.Errorf("Expected place ID 'p1', got %s", inputArc.GetPlaceID())
	}
	if inputArc.GetTransitionID() != "t1" {
		t.Errorf("Expected transition ID 't1', got %s", inputArc.GetTransitionID())
	}

	outputArc := models.NewOutputArc("a2", "t1", "p2", "y")
	if !outputArc.IsOutputArc() {
		t.Error("Should be output arc")
	}
	if outputArc.GetPlaceID() != "p2" {
		t.Errorf("Expected place ID 'p2', got %s", outputArc.GetPlaceID())
	}
}

func TestCPN(t *testing.T) {
	cpn := models.NewCPN("cpn1", "Test CPN", "A test CPN")

	// Add places
	intCS := models.NewIntegerColorSet("INT", false)
	place1 := models.NewPlace("p1", "Place1", intCS)
	place2 := models.NewPlace("p2", "Place2", intCS)
	cpn.AddPlace(place1)
	cpn.AddPlace(place2)

	// Add transition
	transition := models.NewTransition("t1", "Transition1")
	cpn.AddTransition(transition)

	// Add arcs
	inputArc := models.NewInputArc("a1", "p1", "t1", "x")
	outputArc := models.NewOutputArc("a2", "t1", "p2", "x")
	cpn.AddArc(inputArc)
	cpn.AddArc(outputArc)

	// Test retrieval
	if cpn.GetPlace("p1") != place1 {
		t.Error("Should retrieve correct place")
	}
	if cpn.GetTransition("t1") != transition {
		t.Error("Should retrieve correct transition")
	}

	// Test input/output arcs
	inputArcs := cpn.GetInputArcs("t1")
	if len(inputArcs) != 1 || inputArcs[0] != inputArc {
		t.Error("Should retrieve correct input arcs")
	}

	outputArcs := cpn.GetOutputArcs("t1")
	if len(outputArcs) != 1 || outputArcs[0] != outputArc {
		t.Error("Should retrieve correct output arcs")
	}

	// Test initial marking
	token := models.NewToken(1, 0)
	cpn.AddInitialToken("Place1", token)
	marking := cpn.CreateInitialMarking()
	if !marking.HasTokens("Place1") {
		t.Error("Initial marking should have tokens in Place1")
	}

	// Test validation
	errors := cpn.ValidateStructure()
	if len(errors) != 0 {
		t.Errorf("CPN should be valid, got errors: %v", errors)
	}
}
