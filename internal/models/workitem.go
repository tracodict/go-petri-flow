package models

import (
	"fmt"
	"time"
)

// WorkItemStatus represents the status of a work item
type WorkItemStatus string

const (
	WorkItemStatusCreated   WorkItemStatus = "CREATED"
	WorkItemStatusOffered   WorkItemStatus = "OFFERED"
	WorkItemStatusAllocated WorkItemStatus = "ALLOCATED"
	WorkItemStatusStarted   WorkItemStatus = "STARTED"
	WorkItemStatusCompleted WorkItemStatus = "COMPLETED"
	WorkItemStatusFailed    WorkItemStatus = "FAILED"
	WorkItemStatusCancelled WorkItemStatus = "CANCELLED"
	WorkItemStatusOverdue   WorkItemStatus = "OVERDUE"
)

// WorkItemPriority represents the priority of a work item
type WorkItemPriority string

const (
	WorkItemPriorityLow    WorkItemPriority = "LOW"
	WorkItemPriorityNormal WorkItemPriority = "NORMAL"
	WorkItemPriorityHigh   WorkItemPriority = "HIGH"
	WorkItemPriorityUrgent WorkItemPriority = "URGENT"
)

// WorkItem represents a work item in the system
type WorkItem struct {
	ID           string                 `json:"id"`
	CaseID       string                 `json:"caseId"`
	TransitionID string                 `json:"transitionId"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Status       WorkItemStatus         `json:"status"`
	Priority     WorkItemPriority       `json:"priority"`
	CreatedAt    time.Time              `json:"createdAt"`
	OfferedAt    *time.Time             `json:"offeredAt,omitempty"`
	AllocatedAt  *time.Time             `json:"allocatedAt,omitempty"`
	StartedAt    *time.Time             `json:"startedAt,omitempty"`
	CompletedAt  *time.Time             `json:"completedAt,omitempty"`
	DueDate      *time.Time             `json:"dueDate,omitempty"`
	AllocatedTo  string                 `json:"allocatedTo,omitempty"`  // User/Resource ID
	OfferedTo    []string               `json:"offeredTo,omitempty"`    // List of User/Resource IDs
	Data         map[string]interface{} `json:"data"`                   // Work item data
	Metadata     map[string]interface{} `json:"metadata"`               // Additional metadata
	BindingIndex int                    `json:"bindingIndex"`           // Transition binding index
}

// NewWorkItem creates a new work item
func NewWorkItem(id, caseID, transitionID, name, description string) *WorkItem {
	return &WorkItem{
		ID:           id,
		CaseID:       caseID,
		TransitionID: transitionID,
		Name:         name,
		Description:  description,
		Status:       WorkItemStatusCreated,
		Priority:     WorkItemPriorityNormal,
		CreatedAt:    time.Now(),
		Data:         make(map[string]interface{}),
		Metadata:     make(map[string]interface{}),
		OfferedTo:    make([]string, 0),
	}
}

// Offer offers the work item to specific users/resources
func (w *WorkItem) Offer(userIDs []string) {
	w.Status = WorkItemStatusOffered
	now := time.Now()
	w.OfferedAt = &now
	w.OfferedTo = userIDs
}

// Allocate allocates the work item to a specific user/resource
func (w *WorkItem) Allocate(userID string) {
	w.Status = WorkItemStatusAllocated
	now := time.Now()
	w.AllocatedAt = &now
	w.AllocatedTo = userID
}

// Start starts the work item execution
func (w *WorkItem) Start() {
	if w.Status == WorkItemStatusAllocated {
		w.Status = WorkItemStatusStarted
		now := time.Now()
		w.StartedAt = &now
	}
}

// Complete completes the work item
func (w *WorkItem) Complete() {
	w.Status = WorkItemStatusCompleted
	now := time.Now()
	w.CompletedAt = &now
}

// Fail marks the work item as failed
func (w *WorkItem) Fail() {
	w.Status = WorkItemStatusFailed
	now := time.Now()
	w.CompletedAt = &now
}

// Cancel cancels the work item
func (w *WorkItem) Cancel() {
	w.Status = WorkItemStatusCancelled
	now := time.Now()
	w.CompletedAt = &now
}

// IsActive returns true if the work item is in an active state
func (w *WorkItem) IsActive() bool {
	return w.Status == WorkItemStatusOffered ||
		   w.Status == WorkItemStatusAllocated ||
		   w.Status == WorkItemStatusStarted
}

// IsCompleted returns true if the work item is completed
func (w *WorkItem) IsCompleted() bool {
	return w.Status == WorkItemStatusCompleted
}

// IsTerminated returns true if the work item is in a terminal state
func (w *WorkItem) IsTerminated() bool {
	return w.Status == WorkItemStatusCompleted ||
		   w.Status == WorkItemStatusFailed ||
		   w.Status == WorkItemStatusCancelled
}

// IsOverdue returns true if the work item is overdue
func (w *WorkItem) IsOverdue() bool {
	return w.DueDate != nil && time.Now().After(*w.DueDate) && !w.IsTerminated()
}

// SetData sets work item data
func (w *WorkItem) SetData(key string, value interface{}) {
	w.Data[key] = value
}

// GetData gets work item data
func (w *WorkItem) GetData(key string) (interface{}, bool) {
	value, exists := w.Data[key]
	return value, exists
}

// SetMetadata sets metadata for the work item
func (w *WorkItem) SetMetadata(key string, value interface{}) {
	w.Metadata[key] = value
}

// GetMetadata gets metadata for the work item
func (w *WorkItem) GetMetadata(key string) (interface{}, bool) {
	value, exists := w.Metadata[key]
	return value, exists
}

// GetDuration returns the duration of the work item execution
func (w *WorkItem) GetDuration() time.Duration {
	if w.StartedAt == nil {
		return 0
	}
	
	endTime := time.Now()
	if w.CompletedAt != nil {
		endTime = *w.CompletedAt
	}
	
	return endTime.Sub(*w.StartedAt)
}

// GetWaitTime returns the time the work item has been waiting
func (w *WorkItem) GetWaitTime() time.Duration {
	if w.Status == WorkItemStatusCreated {
		return time.Since(w.CreatedAt)
	}
	
	if w.OfferedAt != nil && w.Status == WorkItemStatusOffered {
		return time.Since(*w.OfferedAt)
	}
	
	if w.AllocatedAt != nil && w.Status == WorkItemStatusAllocated {
		return time.Since(*w.AllocatedAt)
	}
	
	return 0
}

// Clone creates a copy of the work item
func (w *WorkItem) Clone() *WorkItem {
	clone := &WorkItem{
		ID:           w.ID,
		CaseID:       w.CaseID,
		TransitionID: w.TransitionID,
		Name:         w.Name,
		Description:  w.Description,
		Status:       w.Status,
		Priority:     w.Priority,
		CreatedAt:    w.CreatedAt,
		AllocatedTo:  w.AllocatedTo,
		BindingIndex: w.BindingIndex,
		Data:         make(map[string]interface{}),
		Metadata:     make(map[string]interface{}),
		OfferedTo:    make([]string, len(w.OfferedTo)),
	}
	
	// Copy time pointers
	if w.OfferedAt != nil {
		offeredAt := *w.OfferedAt
		clone.OfferedAt = &offeredAt
	}
	if w.AllocatedAt != nil {
		allocatedAt := *w.AllocatedAt
		clone.AllocatedAt = &allocatedAt
	}
	if w.StartedAt != nil {
		startedAt := *w.StartedAt
		clone.StartedAt = &startedAt
	}
	if w.CompletedAt != nil {
		completedAt := *w.CompletedAt
		clone.CompletedAt = &completedAt
	}
	if w.DueDate != nil {
		dueDate := *w.DueDate
		clone.DueDate = &dueDate
	}
	
	// Copy slices and maps
	copy(clone.OfferedTo, w.OfferedTo)
	
	for k, v := range w.Data {
		clone.Data[k] = v
	}
	
	for k, v := range w.Metadata {
		clone.Metadata[k] = v
	}
	
	return clone
}

// String returns a string representation of the work item
func (w *WorkItem) String() string {
	return fmt.Sprintf("WorkItem{ID: %s, CaseID: %s, TransitionID: %s, Name: %s, Status: %s, Priority: %s}", 
		w.ID, w.CaseID, w.TransitionID, w.Name, w.Status, w.Priority)
}

// WorkItemFilter represents filters for work item queries
type WorkItemFilter struct {
	CaseID       string             `json:"caseId,omitempty"`
	TransitionID string             `json:"transitionId,omitempty"`
	Status       WorkItemStatus     `json:"status,omitempty"`
	Priority     WorkItemPriority   `json:"priority,omitempty"`
	AllocatedTo  string             `json:"allocatedTo,omitempty"`
	OfferedTo    string             `json:"offeredTo,omitempty"`
	CreatedAfter *time.Time         `json:"createdAfter,omitempty"`
	CreatedBefore *time.Time        `json:"createdBefore,omitempty"`
	DueBefore    *time.Time         `json:"dueBefore,omitempty"`
	Overdue      *bool              `json:"overdue,omitempty"`
}

// Matches checks if a work item matches the filter criteria
func (f *WorkItemFilter) Matches(w *WorkItem) bool {
	if f.CaseID != "" && w.CaseID != f.CaseID {
		return false
	}
	
	if f.TransitionID != "" && w.TransitionID != f.TransitionID {
		return false
	}
	
	if f.Status != "" && w.Status != f.Status {
		return false
	}
	
	if f.Priority != "" && w.Priority != f.Priority {
		return false
	}
	
	if f.AllocatedTo != "" && w.AllocatedTo != f.AllocatedTo {
		return false
	}
	
	if f.OfferedTo != "" {
		found := false
		for _, userID := range w.OfferedTo {
			if userID == f.OfferedTo {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	if f.CreatedAfter != nil && w.CreatedAt.Before(*f.CreatedAfter) {
		return false
	}
	
	if f.CreatedBefore != nil && w.CreatedAt.After(*f.CreatedBefore) {
		return false
	}
	
	if f.DueBefore != nil && (w.DueDate == nil || w.DueDate.After(*f.DueBefore)) {
		return false
	}
	
	if f.Overdue != nil && *f.Overdue != w.IsOverdue() {
		return false
	}
	
	return true
}

// WorkItemSortBy represents sorting options for work items
type WorkItemSortBy string

const (
	WorkItemSortByCreatedAt   WorkItemSortBy = "createdAt"
	WorkItemSortByAllocatedAt WorkItemSortBy = "allocatedAt"
	WorkItemSortByStartedAt   WorkItemSortBy = "startedAt"
	WorkItemSortByCompletedAt WorkItemSortBy = "completedAt"
	WorkItemSortByDueDate     WorkItemSortBy = "dueDate"
	WorkItemSortByPriority    WorkItemSortBy = "priority"
	WorkItemSortByStatus      WorkItemSortBy = "status"
	WorkItemSortByName        WorkItemSortBy = "name"
)

// WorkItemSort represents sorting configuration
type WorkItemSort struct {
	By        WorkItemSortBy `json:"by"`
	Ascending bool           `json:"ascending"`
}

// WorkItemQuery represents a query for work items
type WorkItemQuery struct {
	Filter *WorkItemFilter `json:"filter,omitempty"`
	Sort   *WorkItemSort   `json:"sort,omitempty"`
	Limit  int             `json:"limit,omitempty"`
	Offset int             `json:"offset,omitempty"`
}

