package test

import (
	"fmt"
	"testing"

	case_manager "go-petri-flow/internal/case"
	"go-petri-flow/internal/engine"
	"go-petri-flow/internal/models"
)

func TestCaseCreation(t *testing.T) {
	// Create engine and case manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)

	// Create a simple CPN
	cpn := createSimpleCPN()
	caseManager.RegisterCPN(cpn)

	// Create a case
	case_, err := caseManager.CreateCase("test-case-1", "simple-cpn", "Test Case", "A test case", nil)
	if err != nil {
		t.Fatalf("Failed to create case: %v", err)
	}

	if case_.ID != "test-case-1" {
		t.Errorf("Expected case ID 'test-case-1', got %s", case_.ID)
	}

	if case_.CPNID != "simple-cpn" {
		t.Errorf("Expected CPN ID 'simple-cpn', got %s", case_.CPNID)
	}

	if case_.Status != models.CaseStatusCreated {
		t.Errorf("Expected case status CREATED, got %s", case_.Status)
	}
}

func TestCaseLifecycle(t *testing.T) {
	// Create engine and case manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)

	// Create a simple CPN
	cpn := createSimpleCPN()
	caseManager.RegisterCPN(cpn)

	// Create a case
	_, err := caseManager.CreateCase("test-case-2", "simple-cpn", "Test Case 2", "A test case", nil)
	if err != nil {
		t.Fatalf("Failed to create case: %v", err)
	}

	// Start the case
	err = caseManager.StartCase("test-case-2")
	if err != nil {
		t.Fatalf("Failed to start case: %v", err)
	}

	// Get the case and check status
	case_, err := caseManager.GetCase("test-case-2")
	if err != nil {
		t.Fatalf("Failed to get case: %v", err)
	}

	if case_.Status != models.CaseStatusRunning {
		t.Errorf("Expected case status RUNNING, got %s", case_.Status)
	}

	if case_.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}

	if case_.Marking == nil {
		t.Error("Expected marking to be set")
	}

	// Suspend the case
	err = caseManager.SuspendCase("test-case-2")
	if err != nil {
		t.Fatalf("Failed to suspend case: %v", err)
	}

	case_, _ = caseManager.GetCase("test-case-2")
	if case_.Status != models.CaseStatusSuspended {
		t.Errorf("Expected case status SUSPENDED, got %s", case_.Status)
	}

	// Resume the case
	err = caseManager.ResumeCase("test-case-2")
	if err != nil {
		t.Fatalf("Failed to resume case: %v", err)
	}

	case_, _ = caseManager.GetCase("test-case-2")
	if case_.Status != models.CaseStatusRunning {
		t.Errorf("Expected case status RUNNING, got %s", case_.Status)
	}

	// Abort the case
	err = caseManager.AbortCase("test-case-2")
	if err != nil {
		t.Fatalf("Failed to abort case: %v", err)
	}

	case_, _ = caseManager.GetCase("test-case-2")
	if case_.Status != models.CaseStatusAborted {
		t.Errorf("Expected case status ABORTED, got %s", case_.Status)
	}

	if case_.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestCaseExecution(t *testing.T) {
	// Create engine and case manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)

	// Create a CPN with manual transitions
	cpn := createManualCPN()
	caseManager.RegisterCPN(cpn)
	_, err := caseManager.CreateCase("test-case-3", "manual-cpn", "Test Case 3", "A test case", nil)
	if err != nil {
		t.Fatalf("Failed to create case: %v", err)
	}

	err = caseManager.StartCase("test-case-3")
	if err != nil {
		t.Fatalf("Failed to start case: %v", err)
	}

	// Execute initial step to enable manual transition
	_, err = caseManager.ExecuteStep("test-case-3")
	if err != nil {
		t.Fatalf("Failed to execute initial step: %v", err)
	}

	// Fire a specific transition
	err = caseManager.FireTransition("test-case-3", "ManualTransition", 0)
	if err != nil {
		t.Fatalf("Failed to fire transition: %v", err)
	}
}

func TestCaseVariablesAndMetadata(t *testing.T) {
	// Create engine and case manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)

	// Create a simple CPN
	cpn := createSimpleCPN()
	caseManager.RegisterCPN(cpn)

	// Create a case with variables
	variables := map[string]interface{}{
		"priority": "high",
		"customer": "test-customer",
	}

	case_, err := caseManager.CreateCase("test-case-4", "simple-cpn", "Test Case 4", "A test case", variables)
	if err != nil {
		t.Fatalf("Failed to create case: %v", err)
	}

	// Check variables
	if case_.Variables["priority"] != "high" {
		t.Errorf("Expected priority 'high', got %v", case_.Variables["priority"])
	}

	// Update variables and metadata
	newVariables := map[string]interface{}{
		"status": "processing",
	}

	metadata := map[string]interface{}{
		"source":  "api",
		"version": "1.0",
	}

	err = caseManager.UpdateCase("test-case-4", newVariables, metadata)
	if err != nil {
		t.Fatalf("Failed to update case: %v", err)
	}

	// Get updated case
	case_, err = caseManager.GetCase("test-case-4")
	if err != nil {
		t.Fatalf("Failed to get case: %v", err)
	}

	// Check updated variables and metadata
	if case_.Variables["status"] != "processing" {
		t.Errorf("Expected status 'processing', got %v", case_.Variables["status"])
	}

	if case_.Metadata["source"] != "api" {
		t.Errorf("Expected source 'api', got %v", case_.Metadata["source"])
	}
}

func TestCaseQuery(t *testing.T) {
	// Create engine and case manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)

	// Create a simple CPN
	cpn := createSimpleCPN()
	caseManager.RegisterCPN(cpn)

	// Create multiple cases
	for i := 0; i < 5; i++ {
		caseID := fmt.Sprintf("test-case-query-%d", i)
		_, err := caseManager.CreateCase(caseID, "simple-cpn", fmt.Sprintf("Test Case %d", i), "A test case", nil)
		if err != nil {
			t.Fatalf("Failed to create case %d: %v", i, err)
		}

		// Start some cases
		if i%2 == 0 {
			err = caseManager.StartCase(caseID)
			if err != nil {
				t.Fatalf("Failed to start case %d: %v", i, err)
			}
		}
	}

	// Query all cases
	query := &models.CaseQuery{}
	cases, err := caseManager.QueryCases(query)
	if err != nil {
		t.Fatalf("Failed to query cases: %v", err)
	}

	if len(cases) != 5 {
		t.Errorf("Expected 5 cases, got %d", len(cases))
	}

	// Query running cases only
	query = &models.CaseQuery{
		Filter: &models.CaseFilter{
			Status: models.CaseStatusRunning,
		},
	}

	runningCases, err := caseManager.QueryCases(query)
	if err != nil {
		t.Fatalf("Failed to query running cases: %v", err)
	}

	if len(runningCases) != 3 {
		t.Errorf("Expected 3 running cases, got %d", len(runningCases))
	}

	// Query with sorting
	query = &models.CaseQuery{
		Sort: &models.CaseSort{
			By:        models.CaseSortByCreatedAt,
			Ascending: false,
		},
	}

	sortedCases, err := caseManager.QueryCases(query)
	if err != nil {
		t.Fatalf("Failed to query sorted cases: %v", err)
	}

	// Check that cases are sorted by creation time (descending)
	for i := 1; i < len(sortedCases); i++ {
		if sortedCases[i-1].CreatedAt.Before(sortedCases[i].CreatedAt) {
			t.Error("Cases are not sorted correctly by creation time (descending)")
			break
		}
	}
}

func TestCaseStatistics(t *testing.T) {
	// Create engine and case manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)

	// Create a simple CPN
	cpn := createSimpleCPN()
	caseManager.RegisterCPN(cpn)

	// Create cases with different statuses
	caseManager.CreateCase("stats-case-1", "simple-cpn", "Stats Case 1", "A test case", nil)
	caseManager.CreateCase("stats-case-2", "simple-cpn", "Stats Case 2", "A test case", nil)
	caseManager.CreateCase("stats-case-3", "simple-cpn", "Stats Case 3", "A test case", nil)

	// Start some cases
	caseManager.StartCase("stats-case-1")
	caseManager.StartCase("stats-case-2")

	// Complete one case
	caseManager.StartCase("stats-case-3")
	// Simulate completion by aborting (for testing purposes)
	caseManager.AbortCase("stats-case-3")

	// Get statistics
	stats := caseManager.GetCaseStatistics()

	total, ok := stats["total"].(int)
	if !ok || total != 3 {
		t.Errorf("Expected total 3, got %v", stats["total"])
	}

	byStatus, ok := stats["byStatus"].(map[models.CaseStatus]int)
	if !ok {
		t.Error("Expected byStatus to be a map")
	} else {
		if byStatus[models.CaseStatusRunning] != 2 {
			t.Errorf("Expected 2 running cases, got %d", byStatus[models.CaseStatusRunning])
		}
		if byStatus[models.CaseStatusAborted] != 1 {
			t.Errorf("Expected 1 aborted case, got %d", byStatus[models.CaseStatusAborted])
		}
	}
}

func TestCaseDeletion(t *testing.T) {
	// Create engine and case manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)

	// Create a simple CPN
	cpn := createSimpleCPN()
	caseManager.RegisterCPN(cpn)

	// Create and complete a case
	_, err := caseManager.CreateCase("delete-case", "simple-cpn", "Delete Case", "A test case", nil)
	if err != nil {
		t.Fatalf("Failed to create case: %v", err)
	}

	// Try to delete active case (should fail)
	err = caseManager.DeleteCase("delete-case")
	if err == nil {
		t.Error("Expected error when deleting active case")
	}

	// Start and abort the case
	err = caseManager.StartCase("delete-case")
	if err != nil {
		t.Fatalf("Failed to start case: %v", err)
	}

	err = caseManager.AbortCase("delete-case")
	if err != nil {
		t.Fatalf("Failed to abort case: %v", err)
	}

	// Now deletion should succeed
	err = caseManager.DeleteCase("delete-case")
	if err != nil {
		t.Fatalf("Failed to delete terminated case: %v", err)
	}

	// Verify case is deleted
	_, err = caseManager.GetCase("delete-case")
	if err == nil {
		t.Error("Expected error when getting deleted case")
	}
}

// Helper function to create a simple CPN for testing
func createSimpleCPN() *models.CPN {
	cpn := models.NewCPN("simple-cpn", "Simple CPN", "A simple CPN for testing")

	// Add color set (using string for simplicity)
	stringColorSet := &models.StringColorSet{}

	// Add places
	startPlace := &models.Place{
		ID:       "Start",
		Name:     "Start",
		ColorSet: stringColorSet,
	}
	endPlace := &models.Place{
		ID:       "End",
		Name:     "End",
		ColorSet: stringColorSet,
	}

	cpn.AddPlace(startPlace)
	cpn.AddPlace(endPlace)

	// Add transition
	transition := &models.Transition{
		ID:   "Process",
		Name: "Process",
		Kind: models.TransitionKindAuto,
	}
	cpn.AddTransition(transition)

	// Add arcs
	inArc := &models.Arc{
		ID:         "in",
		SourceID:   "Start",
		TargetID:   "Process",
		Expression: "x",
		Direction:  models.ArcDirectionIn,
	}
	outArc := &models.Arc{
		ID:         "out",
		SourceID:   "Process",
		TargetID:   "End",
		Expression: "x",
		Direction:  models.ArcDirectionOut,
	}

	cpn.AddArc(inArc)
	cpn.AddArc(outArc)

	// Set initial marking
	initialTokens := []*models.Token{
		models.NewToken("start_token", 1),
	}
	cpn.SetInitialMarking("Start", initialTokens)

	// Set end places
	cpn.EndPlaces = []string{"End"}

	return cpn
}
