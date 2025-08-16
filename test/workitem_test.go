package test

import (
	"fmt"
	"testing"
	"time"

	case_manager "go-petri-flow/internal/case"
	"go-petri-flow/internal/engine"
	"go-petri-flow/internal/models"
	"go-petri-flow/internal/workitem"
)

func TestWorkItemCreation(t *testing.T) {
	var err error
	// Create engine, case manager, and work item manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)
	workItemManager := workitem.NewManager(caseManager)

	// Create a CPN with manual transitions
	cpn := createManualCPN()
	caseManager.RegisterCPN(cpn)

	// Create and start a case
	caseID := "test-case-wi-1"
	_, err = caseManager.CreateCase(caseID, "manual-cpn", "Test Case WI 1", "A test case", nil)
	if err != nil {
		t.Fatalf("Failed to create case: %v", err)
	}

	err = caseManager.StartCase(caseID)
	if err != nil {
		t.Fatalf("Failed to start case: %v", err)
	}

	// Fire the initial transition to enable the manual transition
	// Assuming the initial transition is automatic and enables ManualTransition
	_, err = caseManager.ExecuteStep(caseID)
	if err != nil {
		t.Fatalf("Failed to execute initial step: %v", err)
	}

	// Create a work item for a manual transition
	workItem, err := workItemManager.CreateWorkItem("test-wi-1", caseID, "ManualTransition", "Test Work Item", "A test work item", 0)
	if err != nil {
		t.Fatalf("Failed to create work item: %v", err)
	}

	if workItem.ID != "test-wi-1" {
		t.Errorf("Expected work item ID 'test-wi-1', got %s", workItem.ID)
	}

	if workItem.CaseID != "test-case-wi-1" {
		t.Errorf("Expected case ID 'test-case-wi-1', got %s", workItem.CaseID)
	}

	if workItem.Status != models.WorkItemStatusCreated {
		t.Errorf("Expected work item status CREATED, got %s", workItem.Status)
	}
}

func TestWorkItemLifecycle(t *testing.T) {
	var err error
	// Create engine, case manager, and work item manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)
	workItemManager := workitem.NewManager(caseManager)

	// Create a CPN with manual transitions
	cpn := createManualCPN()
	caseManager.RegisterCPN(cpn)

	// Create and start a case
	caseID := "test-case-wi-2"
	_, err = caseManager.CreateCase(caseID, "manual-cpn", "Test Case WI 2", "A test case", nil)
	if err != nil {
		t.Fatalf("Failed to create case: %v", err)
	}

	err = caseManager.StartCase(caseID)
	if err != nil {
		t.Fatalf("Failed to start case: %v", err)
	}

	// Fire the initial transition to enable the manual transition
	_, err = caseManager.ExecuteStep(caseID)
	if err != nil {
		t.Fatalf("Failed to execute initial step: %v", err)
	}

	// Create a work item
	workItem, err := workItemManager.CreateWorkItem("test-wi-2", caseID, "ManualTransition", "Test Work Item 2", "A test work item", 0)
	if err != nil {
		t.Fatalf("Failed to create work item: %v", err)
	}

	// Offer the work item to users
	userIDs := []string{"user1", "user2"}
	err = workItemManager.OfferWorkItem("test-wi-2", userIDs)
	if err != nil {
		t.Fatalf("Failed to offer work item: %v", err)
	}

	// Get the work item and check status
	workItem, err = workItemManager.GetWorkItem("test-wi-2")
	if err != nil {
		t.Fatalf("Failed to get work item: %v", err)
	}

	if workItem.Status != models.WorkItemStatusOffered {
		t.Errorf("Expected work item status OFFERED, got %s", workItem.Status)
	}

	if workItem.OfferedAt == nil {
		t.Error("Expected OfferedAt to be set")
	}

	if len(workItem.OfferedTo) != 2 {
		t.Errorf("Expected 2 offered users, got %d", len(workItem.OfferedTo))
	}

	// Allocate the work item to a user
	err = workItemManager.AllocateWorkItem("test-wi-2", "user1")
	if err != nil {
		t.Fatalf("Failed to allocate work item: %v", err)
	}

	workItem, _ = workItemManager.GetWorkItem("test-wi-2")
	if workItem.Status != models.WorkItemStatusAllocated {
		t.Errorf("Expected work item status ALLOCATED, got %s", workItem.Status)
	}

	if workItem.AllocatedTo != "user1" {
		t.Errorf("Expected allocated to 'user1', got %s", workItem.AllocatedTo)
	}

	// Start the work item
	err = workItemManager.StartWorkItem("test-wi-2")
	if err != nil {
		t.Fatalf("Failed to start work item: %v", err)
	}

	workItem, _ = workItemManager.GetWorkItem("test-wi-2")
	if workItem.Status != models.WorkItemStatusStarted {
		t.Errorf("Expected work item status STARTED, got %s", workItem.Status)
	}

	if workItem.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}

	// Complete the work item
	err = workItemManager.CompleteWorkItem("test-wi-2")
	if err != nil {
		t.Fatalf("Failed to complete work item: %v", err)
	}

	workItem, _ = workItemManager.GetWorkItem("test-wi-2")
	if workItem.Status != models.WorkItemStatusCompleted {
		t.Errorf("Expected work item status COMPLETED, got %s", workItem.Status)
	}

	if workItem.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestWorkItemPriorityAndDueDate(t *testing.T) {
	var err error
	// Create engine, case manager, and work item manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)
	workItemManager := workitem.NewManager(caseManager)

	// Create a CPN with manual transitions
	cpn := createManualCPN()
	caseManager.RegisterCPN(cpn)

	// Create and start a case
	caseID := "test-case-wi-3"
	_, err = caseManager.CreateCase(caseID, "manual-cpn", "Test Case WI 3", "A test case", nil)
	if err != nil {
		t.Fatalf("Failed to create case: %v", err)
	}

	err = caseManager.StartCase(caseID)
	if err != nil {
		t.Fatalf("Failed to start case: %v", err)
	}

	// Fire the initial transition to enable the manual transition
	_, err = caseManager.ExecuteStep(caseID)
	if err != nil {
		t.Fatalf("Failed to execute initial step: %v", err)
	}

	// Create a work item
	_, err = workItemManager.CreateWorkItem("test-wi-3", caseID, "ManualTransition", "Test Work Item 3", "A test work item", 0)
	if err != nil {
		t.Fatalf("Failed to create work item: %v", err)
	}

	// Set priority
	err = workItemManager.SetPriority("test-wi-3", models.WorkItemPriorityHigh)
	if err != nil {
		t.Fatalf("Failed to set priority: %v", err)
	}

	workItem, _ := workItemManager.GetWorkItem("test-wi-3")
	if workItem.Priority != models.WorkItemPriorityHigh {
		t.Errorf("Expected priority HIGH, got %s", workItem.Priority)
	}

	// Set due date
	dueDate := time.Now().Add(24 * time.Hour)
	err = workItemManager.SetDueDate("test-wi-3", &dueDate)
	if err != nil {
		t.Fatalf("Failed to set due date: %v", err)
	}

	workItem, _ = workItemManager.GetWorkItem("test-wi-3")
	if workItem.DueDate == nil {
		t.Error("Expected due date to be set")
	} else if !workItem.DueDate.Equal(dueDate) {
		t.Errorf("Expected due date %v, got %v", dueDate, *workItem.DueDate)
	}

	// Check overdue status (should not be overdue)
	if workItem.IsOverdue() {
		t.Error("Work item should not be overdue")
	}

	// Set past due date
	pastDueDate := time.Now().Add(-1 * time.Hour)
	err = workItemManager.SetDueDate("test-wi-3", &pastDueDate)
	if err != nil {
		t.Fatalf("Failed to set past due date: %v", err)
	}

	workItem, _ = workItemManager.GetWorkItem("test-wi-3")
	if !workItem.IsOverdue() {
		t.Error("Work item should be overdue")
	}
}

func TestWorkItemDataAndMetadata(t *testing.T) {
	var err error
	// Create engine, case manager, and work item manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)
	workItemManager := workitem.NewManager(caseManager)

	// Create a CPN with manual transitions
	cpn := createManualCPN()
	caseManager.RegisterCPN(cpn)

	// Create and start a case
	caseID := "test-case-wi-4"
	_, err = caseManager.CreateCase(caseID, "manual-cpn", "Test Case WI 4", "A test case", nil)
	if err != nil {
		t.Fatalf("Failed to create case: %v", err)
	}

	err = caseManager.StartCase(caseID)
	if err != nil {
		t.Fatalf("Failed to start case: %v", err)
	}

	// Fire the initial transition to enable the manual transition
	_, err = caseManager.ExecuteStep(caseID)
	if err != nil {
		t.Fatalf("Failed to execute initial step: %v", err)
	}

	// Create a work item
	_, err = workItemManager.CreateWorkItem("test-wi-4", caseID, "ManualTransition", "Test Work Item 4", "A test work item", 0)
	if err != nil {
		t.Fatalf("Failed to create work item: %v", err)
	}

	// Update data and metadata
	data := map[string]interface{}{
		"formData": map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
		},
		"amount": 1000,
	}

	metadata := map[string]interface{}{
		"source":    "web",
		"version":   "1.0",
		"timestamp": time.Now().Unix(),
	}

	err = workItemManager.UpdateWorkItem("test-wi-4", data, metadata)
	if err != nil {
		t.Fatalf("Failed to update work item: %v", err)
	}

	// Get updated work item
	workItem, err := workItemManager.GetWorkItem("test-wi-4")
	if err != nil {
		t.Fatalf("Failed to get work item: %v", err)
	}

	// Check data
	if workItem.Data["amount"] != 1000 {
		t.Errorf("Expected amount 1000, got %v", workItem.Data["amount"])
	}

	// Check metadata
	if workItem.Metadata["source"] != "web" {
		t.Errorf("Expected source 'web', got %v", workItem.Metadata["source"])
	}
}

func TestWorkItemQuery(t *testing.T) {
	var err error
	// Create engine, case manager, and work item manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)
	workItemManager := workitem.NewManager(caseManager)

	// Create a CPN with manual transitions
	cpn := createManualCPN()
	caseManager.RegisterCPN(cpn)

	// Create and start a case
	caseID := "test-case-wi-query"
	_, err = caseManager.CreateCase(caseID, "manual-cpn", "Test Case WI Query", "A test case", nil)
	if err != nil {
		t.Fatalf("Failed to create case: %v", err)
	}

	err = caseManager.StartCase(caseID)
	if err != nil {
		t.Fatalf("Failed to start case: %v", err)
	}

	// Fire the initial transition to enable the manual transition
	_, err = caseManager.ExecuteStep(caseID)
	if err != nil {
		t.Fatalf("Failed to execute initial step: %v", err)
	}

	// Create multiple work items
	for i := 0; i < 5; i++ {
		workItemID := fmt.Sprintf("test-wi-query-%d", i)
		_, err := workItemManager.CreateWorkItem(workItemID, caseID, "ManualTransition", fmt.Sprintf("Test Work Item %d", i), "A test work item", 0)
		if err != nil {
			t.Fatalf("Failed to create work item %d: %v", i, err)
		}

		// Set different priorities
		if i%2 == 0 {
			workItemManager.SetPriority(workItemID, models.WorkItemPriorityHigh)
		}

		// Offer some work items
		if i < 3 {
			workItemManager.OfferWorkItem(workItemID, []string{"user1"})
		}
	}

	// Query all work items
	query := &models.WorkItemQuery{}
	workItems, err := workItemManager.QueryWorkItems(query)
	if err != nil {
		t.Fatalf("Failed to query work items: %v", err)
	}

	if len(workItems) != 5 {
		t.Errorf("Expected 5 work items, got %d", len(workItems))
	}

	// Query offered work items only
	query = &models.WorkItemQuery{
		Filter: &models.WorkItemFilter{
			Status: models.WorkItemStatusOffered,
		},
	}

	offeredWorkItems, err := workItemManager.QueryWorkItems(query)
	if err != nil {
		t.Fatalf("Failed to query offered work items: %v", err)
	}

	if len(offeredWorkItems) != 3 {
		t.Errorf("Expected 3 offered work items, got %d", len(offeredWorkItems))
	}

	// Query high priority work items
	query = &models.WorkItemQuery{
		Filter: &models.WorkItemFilter{
			Priority: models.WorkItemPriorityHigh,
		},
	}

	highPriorityWorkItems, err := workItemManager.QueryWorkItems(query)
	if err != nil {
		t.Fatalf("Failed to query high priority work items: %v", err)
	}

	if len(highPriorityWorkItems) != 3 {
		t.Errorf("Expected 3 high priority work items, got %d", len(highPriorityWorkItems))
	}

	// Query with sorting by priority
	query = &models.WorkItemQuery{
		Sort: &models.WorkItemSort{
			By:        models.WorkItemSortByPriority,
			Ascending: false,
		},
	}

	sortedWorkItems, err := workItemManager.QueryWorkItems(query)
	if err != nil {
		t.Fatalf("Failed to query sorted work items: %v", err)
	}

	// Check that high priority items come first
	for i := 0; i < 3; i++ {
		if sortedWorkItems[i].Priority != models.WorkItemPriorityHigh {
			t.Errorf("Expected high priority work item at index %d, got %s", i, sortedWorkItems[i].Priority)
		}
	}
}

func TestWorkItemsByUser(t *testing.T) {
	var err error
	// Create engine, case manager, and work item manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)
	workItemManager := workitem.NewManager(caseManager)

	// Create a CPN with manual transitions
	cpn := createManualCPN()
	caseManager.RegisterCPN(cpn)

	// Create and start a case
	caseID := "test-case-wi-user"
	_, err = caseManager.CreateCase(caseID, "manual-cpn", "Test Case WI User", "A test case", nil)
	if err != nil {
		t.Fatalf("Failed to create case: %v", err)
	}

	err = caseManager.StartCase(caseID)
	if err != nil {
		t.Fatalf("Failed to start case: %v", err)
	}

	// Fire the initial transition to enable the manual transition
	_, err = caseManager.ExecuteStep(caseID)
	if err != nil {
		t.Fatalf("Failed to execute initial step: %v", err)
	}

	// Create work items and assign to different users
	_, _ = workItemManager.CreateWorkItem("test-wi-user-1", caseID, "ManualTransition", "Test Work Item User 1", "A test work item", 0)
	_, _ = workItemManager.CreateWorkItem("test-wi-user-2", caseID, "ManualTransition", "Test Work Item User 2", "A test work item", 0)
	_, _ = workItemManager.CreateWorkItem("test-wi-user-3", caseID, "ManualTransition", "Test Work Item User 3", "A test work item", 0)

	// Offer work items to users
	workItemManager.OfferWorkItem("test-wi-user-1", []string{"user1", "user2"})
	workItemManager.OfferWorkItem("test-wi-user-2", []string{"user2"})

	// Allocate one work item
	workItemManager.AllocateWorkItem("test-wi-user-3", "user1")

	// Get work items by user1
	user1WorkItems, err := workItemManager.GetWorkItemsByUser("user1")
	if err != nil {
		t.Fatalf("Failed to get work items by user1: %v", err)
	}

	if len(user1WorkItems) != 2 {
		t.Errorf("Expected 2 work items for user1, got %d", len(user1WorkItems))
	}

	// Get work items by user2
	user2WorkItems, err := workItemManager.GetWorkItemsByUser("user2")
	if err != nil {
		t.Fatalf("Failed to get work items by user2: %v", err)
	}

	if len(user2WorkItems) != 2 {
		t.Errorf("Expected 2 work items for user2, got %d", len(user2WorkItems))
	}
}

func TestWorkItemStatistics(t *testing.T) {
	var err error
	// Create engine, case manager, and work item manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)
	workItemManager := workitem.NewManager(caseManager)

	// Create a CPN with manual transitions
	cpn := createManualCPN()
	caseManager.RegisterCPN(cpn)

	// Create and start a case
	caseID := "test-case-wi-stats"
	_, err = caseManager.CreateCase(caseID, "manual-cpn", "Test Case WI Stats", "A test case", nil)
	if err != nil {
		t.Fatalf("Failed to create case: %v", err)
	}

	err = caseManager.StartCase(caseID)
	if err != nil {
		t.Fatalf("Failed to start case: %v", err)
	}

	// Fire the initial transition to enable the manual transition
	_, err = caseManager.ExecuteStep(caseID)
	if err != nil {
		t.Fatalf("Failed to execute initial step: %v", err)
	}

	// Create work items with different statuses and priorities
	workItemManager.CreateWorkItem("stats-wi-1", caseID, "ManualTransition", "Stats WI 1", "A test work item", 0)
	workItemManager.CreateWorkItem("stats-wi-2", caseID, "ManualTransition", "Stats WI 2", "A test work item", 0)
	workItemManager.CreateWorkItem("stats-wi-3", caseID, "ManualTransition", "Stats WI 3", "A test work item", 0)
	// Set different priorities
	workItemManager.SetPriority("stats-wi-1", models.WorkItemPriorityHigh)
	workItemManager.SetPriority("stats-wi-2", models.WorkItemPriorityUrgent)

	// Offer and allocate some work items
	workItemManager.OfferWorkItem("stats-wi-1", []string{"user1"})
	workItemManager.AllocateWorkItem("stats-wi-1", "user1")
	workItemManager.StartWorkItem("stats-wi-1")
	workItemManager.CompleteWorkItem("stats-wi-1")

	workItemManager.OfferWorkItem("stats-wi-2", []string{"user2"})

	// Set overdue work item
	pastDueDate := time.Now().Add(-1 * time.Hour)
	workItemManager.SetDueDate("stats-wi-3", &pastDueDate)

	// Get statistics
	stats := workItemManager.GetWorkItemStatistics()

	total, ok := stats["total"].(int)
	if !ok || total != 3 {
		t.Errorf("Expected total 3, got %v", stats["total"])
	}

	byStatus, ok := stats["byStatus"].(map[models.WorkItemStatus]int)
	if !ok {
		t.Error("Expected byStatus to be a map")
	} else {
		if byStatus[models.WorkItemStatusCompleted] != 1 {
			t.Errorf("Expected 1 completed work item, got %v", byStatus[models.WorkItemStatusCompleted])
		}
		if byStatus[models.WorkItemStatusOffered] != 1 {
			t.Errorf("Expected 1 offered work item, got %v", byStatus[models.WorkItemStatusOffered])
		}

	}
	
	// Check overdue count separately
	overdueCount := 0
	for _, wi := range workItemManager.GetAllWorkItems() {
		if wi.IsOverdue() {
			overdueCount++
		}
	}
	if overdueCount != 1 {
		t.Errorf("Expected 1 overdue work item, got %d", overdueCount)
	}

	byPriority, ok := stats["byPriority"].(map[models.WorkItemPriority]int)
	if !ok {
		t.Error("Expected byPriority to be a map")
	} else {
		if byPriority[models.WorkItemPriorityHigh] != 1 {
			t.Errorf("Expected 1 high priority work item, got %v", byPriority[models.WorkItemPriorityHigh])
		}
		if byPriority[models.WorkItemPriorityUrgent] != 1 {
			t.Errorf("Expected 1 urgent priority work item, got %v", byPriority[models.WorkItemPriorityUrgent])
		}
		if byPriority[models.WorkItemPriorityNormal] != 1 {
			t.Errorf("Expected 1 normal priority work item, got %v", byPriority[models.WorkItemPriorityNormal])
		}
	}
}

func TestCreateWorkItemsForCase(t *testing.T) {
	// Create engine, case manager, and work item manager
	eng := engine.NewEngine()
	defer eng.Close()

	caseManager := case_manager.NewManager(eng)
	workItemManager := workitem.NewManager(caseManager)

	// Create a CPN with manual transitions
	cpn := createManualCPN()
	caseManager.RegisterCPN(cpn)

	// Create and start a case
	caseID := "test-case-wi-auto"
	_, err := caseManager.CreateCase(caseID, "manual-cpn", "Test Case WI Auto", "A test case", nil)
	if err != nil {
		t.Fatalf("Failed to create case: %v", err)
	}

	err = caseManager.StartCase(caseID)
	if err != nil {
		t.Fatalf("Failed to start case: %v", err)
	}

	// Fire the initial transition to enable the manual transition
	_, err = caseManager.ExecuteStep(caseID)
	if err != nil {
		t.Fatalf("Failed to execute initial step: %v", err)
	}

	// Create work items for the case
	workItems, err := workItemManager.CreateWorkItemsForCase(caseID)
	if err != nil {
		t.Fatalf("Failed to create work items for case: %v", err)
	}

	if len(workItems) != 1 {
		t.Errorf("Expected 1 work item, got %d", len(workItems))
	}

	if workItems[0].TransitionID != "ManualTransition" {
		t.Errorf("Expected work item for ManualTransition, got %s", workItems[0].TransitionID)
	}
}

func createManualCPN() *models.CPN {
	cpn := models.NewCPN("manual-cpn", "Manual CPN", "A CPN with manual transitions for testing")

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

	// Add manual transition
	transition := &models.Transition{
		ID:   "ManualTransition",
		Name: "Manual Transition",
		Kind: models.TransitionKindManual,
	}
	cpn.AddTransition(transition)

	// Add arcs
	inArc := &models.Arc{
		ID:         "in",
		SourceID:   "Start",
		TargetID:   "ManualTransition",
		Expression: "x",
		Direction:  models.ArcDirectionIn,
	}
	outArc := &models.Arc{
		ID:         "out",
		SourceID:   "ManualTransition",
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

