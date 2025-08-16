package workitem

import (
	"fmt"
	"sort"
	"sync"
	"time"

	case_manager "go-petri-flow/internal/case"
	"go-petri-flow/internal/models"
)

// Manager handles work item lifecycle management
type Manager struct {
	workItems   map[string]*models.WorkItem // Work Item ID -> Work Item
	caseManager *case_manager.Manager       // Reference to case manager
	mutex       sync.RWMutex
}

// NewManager creates a new work item manager
func NewManager(caseManager *case_manager.Manager) *Manager {
	return &Manager{
		workItems:   make(map[string]*models.WorkItem),
		caseManager: caseManager,
	}
}

// CreateWorkItem creates a new work item for a manual transition
func (m *Manager) CreateWorkItem(workItemID, caseID, transitionID, name, description string, bindingIndex int) (*models.WorkItem, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check if work item ID already exists
	if _, exists := m.workItems[workItemID]; exists {
		return nil, fmt.Errorf("work item with ID %s already exists", workItemID)
	}
	
	// Verify that the case exists and the transition is enabled
	enabledTransitions, bindingsMap, err := m.caseManager.GetEnabledTransitions(caseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled transitions for case %s: %v", caseID, err)
	}
	
	// Check if the transition is enabled
	var transitionFound bool
	for _, transition := range enabledTransitions {
		if transition.ID == transitionID {
			transitionFound = true
			// Check if the binding index is valid
			bindings := bindingsMap[transitionID]
			if bindingIndex >= len(bindings) {
				return nil, fmt.Errorf("binding index %d out of range for transition %s", bindingIndex, transitionID)
			}
			break
		}
	}
	
	if !transitionFound {
		return nil, fmt.Errorf("transition %s is not enabled for case %s", transitionID, caseID)
	}
	
	// Create the work item
	workItem := models.NewWorkItem(workItemID, caseID, transitionID, name, description)
	workItem.BindingIndex = bindingIndex
	
	// Store the work item
	m.workItems[workItemID] = workItem
	
	return workItem, nil
}

// GetWorkItem retrieves a work item by ID
func (m *Manager) GetWorkItem(workItemID string) (*models.WorkItem, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	workItem, exists := m.workItems[workItemID]
	if !exists {
		return nil, fmt.Errorf("work item with ID %s not found", workItemID)
	}
	
	return workItem.Clone(), nil
}

// UpdateWorkItem updates work item data and metadata
func (m *Manager) UpdateWorkItem(workItemID string, data map[string]interface{}, metadata map[string]interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	workItem, exists := m.workItems[workItemID]
	if !exists {
		return fmt.Errorf("work item with ID %s not found", workItemID)
	}
	
	// Update data
	if data != nil {
		for k, v := range data {
			workItem.SetData(k, v)
		}
	}
	
	// Update metadata
	if metadata != nil {
		for k, v := range metadata {
			workItem.SetMetadata(k, v)
		}
	}
	
	return nil
}

// SetPriority sets the priority of a work item
func (m *Manager) SetPriority(workItemID string, priority models.WorkItemPriority) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	workItem, exists := m.workItems[workItemID]
	if !exists {
		return fmt.Errorf("work item with ID %s not found", workItemID)
	}
	
	workItem.Priority = priority
	return nil
}

// SetDueDate sets the due date of a work item
func (m *Manager) SetDueDate(workItemID string, dueDate *time.Time) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	workItem, exists := m.workItems[workItemID]
	if !exists {
		return fmt.Errorf("work item with ID %s not found", workItemID)
	}
	
	workItem.DueDate = dueDate
	return nil
}

// OfferWorkItem offers a work item to specific users/resources
func (m *Manager) OfferWorkItem(workItemID string, userIDs []string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	workItem, exists := m.workItems[workItemID]
	if !exists {
		return fmt.Errorf("work item with ID %s not found", workItemID)
	}
	
	if workItem.Status != models.WorkItemStatusCreated {
		return fmt.Errorf("work item %s is not in CREATED status, current status: %s", workItemID, workItem.Status)
	}
	
	workItem.Offer(userIDs)
	return nil
}

// AllocateWorkItem allocates a work item to a specific user/resource
func (m *Manager) AllocateWorkItem(workItemID, userID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	workItem, exists := m.workItems[workItemID]
	if !exists {
		return fmt.Errorf("work item with ID %s not found", workItemID)
	}
	
	if workItem.Status != models.WorkItemStatusOffered && workItem.Status != models.WorkItemStatusCreated {
		return fmt.Errorf("work item %s cannot be allocated, current status: %s", workItemID, workItem.Status)
	}
	
	// Check if user is in the offered list (if work item was offered)
	if workItem.Status == models.WorkItemStatusOffered {
		userFound := false
		for _, offeredUserID := range workItem.OfferedTo {
			if offeredUserID == userID {
				userFound = true
				break
			}
		}
		if !userFound {
			return fmt.Errorf("user %s is not in the offered list for work item %s", userID, workItemID)
		}
	}
	
	workItem.Allocate(userID)
	return nil
}

// StartWorkItem starts a work item execution
func (m *Manager) StartWorkItem(workItemID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	workItem, exists := m.workItems[workItemID]
	if !exists {
		return fmt.Errorf("work item with ID %s not found", workItemID)
	}
	
	if workItem.Status != models.WorkItemStatusAllocated {
		return fmt.Errorf("work item %s is not allocated, current status: %s", workItemID, workItem.Status)
	}
	
	workItem.Start()
	return nil
}

// CompleteWorkItem completes a work item and fires the associated transition
func (m *Manager) CompleteWorkItem(workItemID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	workItem, exists := m.workItems[workItemID]
	if !exists {
		return fmt.Errorf("work item with ID %s not found", workItemID)
	}
	
	if workItem.Status != models.WorkItemStatusStarted {
		return fmt.Errorf("work item %s is not started, current status: %s", workItemID, workItem.Status)
	}
	
	// Fire the associated transition
	err := m.caseManager.FireTransition(workItem.CaseID, workItem.TransitionID, workItem.BindingIndex)
	if err != nil {
		return fmt.Errorf("failed to fire transition %s for case %s: %v", workItem.TransitionID, workItem.CaseID, err)
	}
	
	// Mark work item as completed
	workItem.Complete()
	
	return nil
}

// FailWorkItem marks a work item as failed
func (m *Manager) FailWorkItem(workItemID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	workItem, exists := m.workItems[workItemID]
	if !exists {
		return fmt.Errorf("work item with ID %s not found", workItemID)
	}
	
	if !workItem.IsActive() {
		return fmt.Errorf("work item %s is not active, current status: %s", workItemID, workItem.Status)
	}
	
	workItem.Fail()
	return nil
}

// CancelWorkItem cancels a work item
func (m *Manager) CancelWorkItem(workItemID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	workItem, exists := m.workItems[workItemID]
	if !exists {
		return fmt.Errorf("work item with ID %s not found", workItemID)
	}
	
	if !workItem.IsActive() {
		return fmt.Errorf("work item %s is not active, current status: %s", workItemID, workItem.Status)
	}
	
	workItem.Cancel()
	return nil
}

// DeleteWorkItem deletes a work item
func (m *Manager) DeleteWorkItem(workItemID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	workItem, exists := m.workItems[workItemID]
	if !exists {
		return fmt.Errorf("work item with ID %s not found", workItemID)
	}
	
	// Only allow deletion of terminated work items
	if !workItem.IsTerminated() {
		return fmt.Errorf("cannot delete active work item %s, current status: %s", workItemID, workItem.Status)
	}
	
	delete(m.workItems, workItemID)
	return nil
}

// QueryWorkItems queries work items based on filter criteria
func (m *Manager) QueryWorkItems(query *models.WorkItemQuery) ([]*models.WorkItem, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	var result []*models.WorkItem
	
	// Apply filter
	for _, workItem := range m.workItems {
		if query.Filter == nil || query.Filter.Matches(workItem) {
			result = append(result, workItem.Clone())
		}
	}
	
	// Apply sorting
	if query.Sort != nil {
		m.sortWorkItems(result, query.Sort)
	}
	
	// Apply pagination
	if query.Offset > 0 || query.Limit > 0 {
		start := query.Offset
		if start > len(result) {
			start = len(result)
		}
		
		end := len(result)
		if query.Limit > 0 && start+query.Limit < end {
			end = start + query.Limit
		}
		
		result = result[start:end]
	}
	
	return result, nil
}

// sortWorkItems sorts work items based on the sort configuration
func (m *Manager) sortWorkItems(workItems []*models.WorkItem, sortConfig *models.WorkItemSort) {
	sort.Slice(workItems, func(i, j int) bool {
		var less bool
		
		switch sortConfig.By {
		case models.WorkItemSortByCreatedAt:
			less = workItems[i].CreatedAt.Before(workItems[j].CreatedAt)
		case models.WorkItemSortByAllocatedAt:
			if workItems[i].AllocatedAt == nil && workItems[j].AllocatedAt == nil {
				less = false
			} else if workItems[i].AllocatedAt == nil {
				less = true
			} else if workItems[j].AllocatedAt == nil {
				less = false
			} else {
				less = workItems[i].AllocatedAt.Before(*workItems[j].AllocatedAt)
			}
		case models.WorkItemSortByStartedAt:
			if workItems[i].StartedAt == nil && workItems[j].StartedAt == nil {
				less = false
			} else if workItems[i].StartedAt == nil {
				less = true
			} else if workItems[j].StartedAt == nil {
				less = false
			} else {
				less = workItems[i].StartedAt.Before(*workItems[j].StartedAt)
			}
		case models.WorkItemSortByCompletedAt:
			if workItems[i].CompletedAt == nil && workItems[j].CompletedAt == nil {
				less = false
			} else if workItems[i].CompletedAt == nil {
				less = true
			} else if workItems[j].CompletedAt == nil {
				less = false
			} else {
				less = workItems[i].CompletedAt.Before(*workItems[j].CompletedAt)
			}
		case models.WorkItemSortByDueDate:
			if workItems[i].DueDate == nil && workItems[j].DueDate == nil {
				less = false
			} else if workItems[i].DueDate == nil {
				less = true
			} else if workItems[j].DueDate == nil {
				less = false
			} else {
				less = workItems[i].DueDate.Before(*workItems[j].DueDate)
			}
		case models.WorkItemSortByPriority:
			priorityOrder := map[models.WorkItemPriority]int{
				models.WorkItemPriorityLow:    1,
				models.WorkItemPriorityNormal: 2,
				models.WorkItemPriorityHigh:   3,
				models.WorkItemPriorityUrgent: 4,
			}
			less = priorityOrder[workItems[i].Priority] < priorityOrder[workItems[j].Priority]
		case models.WorkItemSortByStatus:
			less = string(workItems[i].Status) < string(workItems[j].Status)
		case models.WorkItemSortByName:
			less = workItems[i].Name < workItems[j].Name
		default:
			less = workItems[i].CreatedAt.Before(workItems[j].CreatedAt)
		}
		
		if !sortConfig.Ascending {
			less = !less
		}
		
		return less
	})
}

// GetWorkItemsByCase returns all work items for a specific case
func (m *Manager) GetWorkItemsByCase(caseID string) ([]*models.WorkItem, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	var workItems []*models.WorkItem
	for _, workItem := range m.workItems {
		if workItem.CaseID == caseID {
			workItems = append(workItems, workItem.Clone())
		}
	}
	
	return workItems, nil
}

// GetWorkItemsByUser returns all work items allocated to or offered to a specific user
func (m *Manager) GetWorkItemsByUser(userID string) ([]*models.WorkItem, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	var workItems []*models.WorkItem
	for _, workItem := range m.workItems {
		// Check if allocated to user
		if workItem.AllocatedTo == userID {
			workItems = append(workItems, workItem.Clone())
			continue
		}
		
		// Check if offered to user
		for _, offeredUserID := range workItem.OfferedTo {
			if offeredUserID == userID {
				workItems = append(workItems, workItem.Clone())
				break
			}
		}
	}
	
	return workItems, nil
}

// GetOverdueWorkItems returns all overdue work items
func (m *Manager) GetOverdueWorkItems() ([]*models.WorkItem, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	var overdueWorkItems []*models.WorkItem
	for _, workItem := range m.workItems {
		if workItem.IsOverdue() {
			overdueWorkItems = append(overdueWorkItems, workItem.Clone())
		}
	}
	
	return overdueWorkItems, nil
}

// GetWorkItemStatistics returns statistics about work items
func (m *Manager) GetWorkItemStatistics() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	stats := map[string]interface{}{
		"total":        len(m.workItems),
		"byStatus":     make(map[models.WorkItemStatus]int),
		"byPriority":   make(map[models.WorkItemPriority]int),
		"overdue":      0,
		"avgDuration":  0.0,
		"avgWaitTime":  0.0,
	}
	
	var totalDuration time.Duration
	var totalWaitTime time.Duration
	completedCount := 0
	overdueCount := 0
	
	for _, workItem := range m.workItems {
		// Count by status
		statusCounts := stats["byStatus"].(map[models.WorkItemStatus]int)
		statusCounts[workItem.Status]++
		
		// Count by priority
		priorityCounts := stats["byPriority"].(map[models.WorkItemPriority]int)
		priorityCounts[workItem.Priority]++
		
		// Count overdue
		if workItem.IsOverdue() {
			overdueCount++
		}
		
		// Calculate average duration for completed work items
		if workItem.IsCompleted() {
			totalDuration += workItem.GetDuration()
			completedCount++
		}
		
		// Calculate average wait time
		totalWaitTime += workItem.GetWaitTime()
	}
	
	stats["overdue"] = overdueCount
	
	if completedCount > 0 {
		avgDuration := totalDuration / time.Duration(completedCount)
		stats["avgDuration"] = avgDuration.Seconds()
	}
	
	if len(m.workItems) > 0 {
		avgWaitTime := totalWaitTime / time.Duration(len(m.workItems))
		stats["avgWaitTime"] = avgWaitTime.Seconds()
	}
	
	return stats
}

// GetActiveWorkItems returns all active work items
func (m *Manager) GetActiveWorkItems() []*models.WorkItem {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	var activeWorkItems []*models.WorkItem
	for _, workItem := range m.workItems {
		if workItem.IsActive() {
			activeWorkItems = append(activeWorkItems, workItem.Clone())
		}
	}
	
	return activeWorkItems
}

// GetWorkItemCount returns the total number of work items
func (m *Manager) GetWorkItemCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.workItems)
}

// CreateWorkItemsForCase creates work items for all manual transitions in a case
func (m *Manager) CreateWorkItemsForCase(caseID string) ([]*models.WorkItem, error) {
	// Get enabled transitions for the case
	enabledTransitions, bindingsMap, err := m.caseManager.GetEnabledTransitions(caseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled transitions for case %s: %v", caseID, err)
	}
	
	var createdWorkItems []*models.WorkItem
	
	for _, transition := range enabledTransitions {
		// Only create work items for manual transitions
		if transition.Kind == models.TransitionKindManual {
			bindings := bindingsMap[transition.ID]
			
			// Create a work item for each binding
			for i := range bindings {
				workItemID := fmt.Sprintf("%s-%s-%d", caseID, transition.ID, i)
				
				// Check if work item already exists
				if _, exists := m.workItems[workItemID]; exists {
					continue
				}
				
				workItem, err := m.CreateWorkItem(workItemID, caseID, transition.ID, transition.Name, transition.Name, i)
				if err != nil {
					// Log error but continue with other work items
					continue
				}
				
				createdWorkItems = append(createdWorkItems, workItem)
			}
		}
	}
	
	return createdWorkItems, nil
}



// GetAllWorkItems returns all work items
func (m *Manager) GetAllWorkItems() map[string]*models.WorkItem {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Return a copy to prevent external modification
	clonedWorkItems := make(map[string]*models.WorkItem)
	for id, wi := range m.workItems {
		clonedWorkItems[id] = wi.Clone()
	}
	return clonedWorkItems
}


