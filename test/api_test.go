package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-petri-flow/internal/api"
	"go-petri-flow/internal/models"
)

func TestAPILoadCPN(t *testing.T) {
	server := api.NewServer()
	defer server.Close()

	// Create test CPN definition
	cpnDef := models.CPNDefinitionJSON{
		ID:          "test-cpn",
		Name:        "Test CPN",
		Description: "A test CPN for API testing",
		ColorSets:   []string{"colset INT = int;"},
		Places: []models.PlaceJSON{
			{ID: "p1", Name: "Place1", ColorSet: "INT"},
			{ID: "p2", Name: "Place2", ColorSet: "INT"},
		},
		Transitions: []models.TransitionJSON{
			{ID: "t1", Name: "Transition1", Kind: "Auto"},
		},
		Arcs: []models.ArcJSON{
			{ID: "a1", SourceID: "p1", TargetID: "t1", Expression: "x", Direction: "IN"},
			{ID: "a2", SourceID: "t1", TargetID: "p2", Expression: "x + 1", Direction: "OUT"},
		},
		InitialMarking: map[string][]models.TokenJSON{"p1": {{Value: 5, Timestamp: 0}}},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(cpnDef)
	if err != nil {
		t.Fatalf("Failed to marshal CPN definition: %v", err)
	}

	// Create request
	req, err := http.NewRequest("POST", "/api/cpn/load", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler := server.SetupRoutes()
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Parse response
	var response api.SuccessResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}
}

func TestAPIListCPNs(t *testing.T) {
	server := api.NewServer()
	defer server.Close()

	// First load a CPN
	cpnDef := models.CPNDefinitionJSON{
		ID:          "list-test-cpn",
		Name:        "List Test CPN",
		Description: "A test CPN for list testing",
		ColorSets:   []string{"colset INT = int;"},
		Places: []models.PlaceJSON{
			{ID: "p1", Name: "Place1", ColorSet: "INT"},
		},
		Transitions: []models.TransitionJSON{
			{ID: "t1", Name: "Transition1", Kind: "Auto"},
		},
		Arcs: []models.ArcJSON{},
	}

	jsonData, _ := json.Marshal(cpnDef)
	loadReq, _ := http.NewRequest("POST", "/api/cpn/load", bytes.NewBuffer(jsonData))
	loadReq.Header.Set("Content-Type", "application/json")
	loadRR := httptest.NewRecorder()

	handler := server.SetupRoutes()
	handler.ServeHTTP(loadRR, loadReq)

	// Now test list endpoint
	req, err := http.NewRequest("GET", "/api/cpn/list", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	var response api.SuccessResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}

	// Check that the response contains our CPN
	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	cpns, ok := data["cpns"].([]interface{})
	if !ok {
		t.Fatal("Expected cpns to be an array")
	}

	if len(cpns) != 1 {
		t.Errorf("Expected 1 CPN, got %d", len(cpns))
	}
}

func TestAPIGetMarking(t *testing.T) {
	server := api.NewServer()
	defer server.Close()

	// Load a CPN first
	cpnDef := models.CPNDefinitionJSON{
		ID:          "marking-test-cpn",
		Name:        "Marking Test CPN",
		Description: "A test CPN for marking testing",
		ColorSets:   []string{"colset INT = int;"},
		Places: []models.PlaceJSON{
			{ID: "p1", Name: "Place1", ColorSet: "INT"},
			{ID: "p2", Name: "Place2", ColorSet: "INT"},
		},
		Transitions: []models.TransitionJSON{
			{ID: "t1", Name: "Transition1", Kind: "Auto"},
		},
		Arcs: []models.ArcJSON{
			{ID: "a1", SourceID: "p1", TargetID: "t1", Expression: "x", Direction: "IN"},
			{ID: "a2", SourceID: "t1", TargetID: "p2", Expression: "x", Direction: "OUT"},
		},
		InitialMarking: map[string][]models.TokenJSON{"p1": {{Value: 42, Timestamp: 0}}},
	}

	jsonData, _ := json.Marshal(cpnDef)
	loadReq, _ := http.NewRequest("POST", "/api/cpn/load", bytes.NewBuffer(jsonData))
	loadReq.Header.Set("Content-Type", "application/json")
	loadRR := httptest.NewRecorder()

	handler := server.SetupRoutes()
	handler.ServeHTTP(loadRR, loadReq)

	// Test get marking
	req, err := http.NewRequest("GET", "/api/marking/get?id=marking-test-cpn", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	var response api.SuccessResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}

	// Check marking data
	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	places, ok := data["places"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected places to be a map")
	}

	// Updated key format: Name(ID)
	place1Key := "Place1(p1)"
	place1TokensRaw, exists := places[place1Key]
	if !exists {
		place1TokensRaw = places["p1"]
		place1Key = "p1"
	}
	place1Tokens, ok := place1TokensRaw.([]interface{})
	if !ok {
		t.Fatalf("Expected %s tokens to be an array", place1Key)
	}

	if len(place1Tokens) != 1 {
		t.Errorf("Expected 1 token in Place1, got %d", len(place1Tokens))
	}
}

func TestAPIGetTransitions(t *testing.T) {
	server := api.NewServer()
	defer server.Close()

	// Load a CPN first
	cpnDef := models.CPNDefinitionJSON{
		ID:          "transitions-test-cpn",
		Name:        "Transitions Test CPN",
		Description: "A test CPN for transitions testing",
		ColorSets:   []string{"colset INT = int;"},
		Places: []models.PlaceJSON{
			{ID: "p1", Name: "Place1", ColorSet: "INT"},
			{ID: "p2", Name: "Place2", ColorSet: "INT"},
		},
		Transitions: []models.TransitionJSON{
			{ID: "t1", Name: "Transition1", Kind: "Auto"},
			{ID: "t2", Name: "Transition2", Kind: "Manual", GuardExpression: "x > 10", Variables: []string{"x"}},
		},
		Arcs: []models.ArcJSON{
			{ID: "a1", SourceID: "p1", TargetID: "t1", Expression: "x", Direction: "IN"},
			{ID: "a2", SourceID: "t1", TargetID: "p2", Expression: "x", Direction: "OUT"},
			{ID: "a3", SourceID: "p1", TargetID: "t2", Expression: "x", Direction: "IN"},
			{ID: "a4", SourceID: "t2", TargetID: "p2", Expression: "x", Direction: "OUT"},
		},
		InitialMarking: map[string][]models.TokenJSON{"p1": {{Value: 5, Timestamp: 0}}},
	}

	jsonData, _ := json.Marshal(cpnDef)
	loadReq, _ := http.NewRequest("POST", "/api/cpn/load", bytes.NewBuffer(jsonData))
	loadReq.Header.Set("Content-Type", "application/json")
	loadRR := httptest.NewRecorder()

	handler := server.SetupRoutes()
	handler.ServeHTTP(loadRR, loadReq)

	// Test get transitions
	req, err := http.NewRequest("GET", "/api/transitions/list?id=transitions-test-cpn", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	var response api.SuccessResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}

	// Check transitions data
	transitions, ok := response.Data.([]interface{})
	if !ok {
		t.Fatal("Expected data to be an array")
	}

	if len(transitions) != 2 {
		t.Errorf("Expected 2 transitions, got %d", len(transitions))
	}

	// Check that t1 is enabled and t2 is not (due to guard x > 10 with token value 5)
	for _, transitionData := range transitions {
		transition, ok := transitionData.(map[string]interface{})
		if !ok {
			t.Fatal("Expected transition to be a map")
		}

		id, ok := transition["id"].(string)
		if !ok {
			t.Fatal("Expected transition id to be a string")
		}

		enabled, ok := transition["enabled"].(bool)
		if !ok {
			t.Fatal("Expected transition enabled to be a boolean")
		}

		if id == "t1" && !enabled {
			t.Error("Expected t1 to be enabled")
		}
		if id == "t2" && enabled {
			t.Error("Expected t2 to be disabled (guard x > 10 with token value 5)")
		}
	}
}

func TestAPIFireTransition(t *testing.T) {
	server := api.NewServer()
	defer server.Close()

	// Load a CPN first
	cpnDef := models.CPNDefinitionJSON{
		ID:          "fire-test-cpn",
		Name:        "Fire Test CPN",
		Description: "A test CPN for firing testing",
		ColorSets:   []string{"colset INT = int;"},
		Places: []models.PlaceJSON{
			{ID: "p1", Name: "Place1", ColorSet: "INT"},
			{ID: "p2", Name: "Place2", ColorSet: "INT"},
		},
		Transitions: []models.TransitionJSON{
			{ID: "t1", Name: "Transition1", Kind: "Manual"},
		},
		Arcs: []models.ArcJSON{
			{ID: "a1", SourceID: "p1", TargetID: "t1", Expression: "x", Direction: "IN"},
			{ID: "a2", SourceID: "t1", TargetID: "p2", Expression: "x + 10", Direction: "OUT"},
		},
		InitialMarking: map[string][]models.TokenJSON{"p1": {{Value: 5, Timestamp: 0}}},
	}

	jsonData, _ := json.Marshal(cpnDef)
	loadReq, _ := http.NewRequest("POST", "/api/cpn/load", bytes.NewBuffer(jsonData))
	loadReq.Header.Set("Content-Type", "application/json")
	loadRR := httptest.NewRecorder()

	handler := server.SetupRoutes()
	handler.ServeHTTP(loadRR, loadReq)

	// Test fire transition
	fireRequest := map[string]interface{}{
		"cpnId":        "fire-test-cpn",
		"transitionId": "t1",
		"bindingIndex": 0,
	}

	fireJSON, _ := json.Marshal(fireRequest)
	req, err := http.NewRequest("POST", "/api/transitions/fire", bytes.NewBuffer(fireJSON))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	var response api.SuccessResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}

	// Check that the marking changed
	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	places, ok := data["places"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected places to be a map")
	}

	// Place1 should be empty
	if place1Tokens, exists := places["Place1(p1)"]; exists {
		if tokens, ok := place1Tokens.([]interface{}); ok && len(tokens) > 0 {
			t.Error("Expected Place1 to be empty after firing")
		}
	}

	// Place2 should have a token with value 15 (5 + 10)
	place2Key := "Place2(p2)"
	place2TokensRaw, exist2 := places[place2Key]
	if !exist2 {
		place2TokensRaw = places["p2"]
		place2Key = "p2"
	}
	place2Tokens, ok := place2TokensRaw.([]interface{})
	if !ok {
		t.Fatalf("Expected %s tokens to be an array", place2Key)
	}

	if len(place2Tokens) != 1 {
		t.Errorf("Expected 1 token in Place2, got %d", len(place2Tokens))
	}

	token, ok := place2Tokens[0].(map[string]interface{})
	if !ok {
		t.Fatal("Expected token to be a map")
	}

	value, ok := token["value"].(float64) // JSON numbers are float64
	if !ok {
		t.Fatal("Expected token value to be a number")
	}

	if int(value) != 15 {
		t.Errorf("Expected token value 15, got %d", int(value))
	}
}

func TestAPISimulateStep(t *testing.T) {
	server := api.NewServer()
	defer server.Close()

	// Load a CPN with automatic transitions
	cpnDef := models.CPNDefinitionJSON{
		ID:          "simulate-test-cpn",
		Name:        "Simulate Test CPN",
		Description: "A test CPN for simulation testing",
		ColorSets:   []string{"colset INT = int;"},
		Places: []models.PlaceJSON{
			{ID: "p1", Name: "Place1", ColorSet: "INT"},
			{ID: "p2", Name: "Place2", ColorSet: "INT"},
			{ID: "p3", Name: "Place3", ColorSet: "INT"},
		},
		Transitions: []models.TransitionJSON{
			{ID: "t1", Name: "Transition1", Kind: "Auto"},
			{ID: "t2", Name: "Transition2", Kind: "Auto"},
		},
		Arcs: []models.ArcJSON{
			{ID: "a1", SourceID: "p1", TargetID: "t1", Expression: "x", Direction: "IN"},
			{ID: "a2", SourceID: "t1", TargetID: "p2", Expression: "x", Direction: "OUT"},
			{ID: "a3", SourceID: "p2", TargetID: "t2", Expression: "x", Direction: "IN"},
			{ID: "a4", SourceID: "t2", TargetID: "p3", Expression: "x", Direction: "OUT"},
		},
		InitialMarking: map[string][]models.TokenJSON{"p1": {{Value: 1, Timestamp: 0}}},
		EndPlaces:      []string{"Place3"},
	}

	jsonData, _ := json.Marshal(cpnDef)
	loadReq, _ := http.NewRequest("POST", "/api/cpn/load", bytes.NewBuffer(jsonData))
	loadReq.Header.Set("Content-Type", "application/json")
	loadRR := httptest.NewRecorder()

	handler := server.SetupRoutes()
	handler.ServeHTTP(loadRR, loadReq)

	// First simulate step (layer 1)
	req1, err := http.NewRequest("POST", "/api/simulation/step?id=simulate-test-cpn", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	if status := rr1.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
	var response1 api.SuccessResponse
	if err := json.Unmarshal(rr1.Body.Bytes(), &response1); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if !response1.Success {
		t.Error("Expected success to be true (step 1)")
	}
	data1, ok := response1.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map (step 1)")
	}
	if int(data1["transitionsFired"].(float64)) != 1 {
		t.Errorf("Expected 1 transition fired in step 1, got %v", data1["transitionsFired"])
	}
	if data1["completed"].(bool) {
		t.Error("Did not expect CPN to be completed after first step")
	}

	// Second simulate step (layer 2)
	req2, err := http.NewRequest("POST", "/api/simulation/step?id=simulate-test-cpn", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if status := rr2.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
	var response2 api.SuccessResponse
	if err := json.Unmarshal(rr2.Body.Bytes(), &response2); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if !response2.Success {
		t.Error("Expected success to be true (step 2)")
	}
	data2, ok := response2.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map (step 2)")
	}
	if int(data2["transitionsFired"].(float64)) != 1 {
		t.Errorf("Expected 1 transition fired in step 2, got %v", data2["transitionsFired"])
	}
	if !data2["completed"].(bool) {
		t.Error("Expected CPN to be completed after second step")
	}
}

func TestAPIHealthCheck(t *testing.T) {
	server := api.NewServer()
	defer server.Close()

	req, err := http.NewRequest("GET", "/api/health", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := server.SetupRoutes()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	var response api.SuccessResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}

	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	status, ok := data["status"].(string)
	if !ok {
		t.Fatal("Expected status to be a string")
	}

	if status != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", status)
	}
}
