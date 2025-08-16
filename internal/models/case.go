package models

import (
	"fmt"
	"time"
)

// CaseStatus represents the status of a case
type CaseStatus string

const (
	CaseStatusCreated   CaseStatus = "CREATED"
	CaseStatusRunning   CaseStatus = "RUNNING"
	CaseStatusCompleted CaseStatus = "COMPLETED"
	CaseStatusSuspended CaseStatus = "SUSPENDED"
	CaseStatusAborted   CaseStatus = "ABORTED"
)

// Case represents a case instance in the CPN
type Case struct {
	ID          string                 `json:"id"`
	CPNID       string                 `json:"cpnId"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      CaseStatus             `json:"status"`
	CreatedAt   time.Time              `json:"createdAt"`
	StartedAt   *time.Time             `json:"startedAt,omitempty"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	Marking     *Marking               `json:"marking"`
	Variables   map[string]interface{} `json:"variables"` // Case-level variables
	Metadata    map[string]interface{} `json:"metadata"`  // Additional metadata
}

// NewCase creates a new case instance
func NewCase(id, cpnID, name, description string) *Case {
	return &Case{
		ID:          id,
		CPNID:       cpnID,
		Name:        name,
		Description: description,
		Status:      CaseStatusCreated,
		CreatedAt:   time.Now(),
		Variables:   make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}
}

// Start starts the case execution
func (c *Case) Start(initialMarking *Marking) {
	c.Status = CaseStatusRunning
	now := time.Now()
	c.StartedAt = &now
	c.Marking = initialMarking
}

// Complete marks the case as completed
func (c *Case) Complete() {
	c.Status = CaseStatusCompleted
	now := time.Now()
	c.CompletedAt = &now
}

// Suspend suspends the case execution
func (c *Case) Suspend() {
	if c.Status == CaseStatusRunning {
		c.Status = CaseStatusSuspended
	}
}

// Resume resumes the case execution
func (c *Case) Resume() {
	if c.Status == CaseStatusSuspended {
		c.Status = CaseStatusRunning
	}
}

// Abort aborts the case execution
func (c *Case) Abort() {
	c.Status = CaseStatusAborted
	now := time.Now()
	c.CompletedAt = &now
}

// IsActive returns true if the case is in an active state
func (c *Case) IsActive() bool {
	return c.Status == CaseStatusRunning || c.Status == CaseStatusSuspended
}

// IsCompleted returns true if the case is completed
func (c *Case) IsCompleted() bool {
	return c.Status == CaseStatusCompleted
}

// IsTerminated returns true if the case is in a terminal state
func (c *Case) IsTerminated() bool {
	return c.Status == CaseStatusCompleted || c.Status == CaseStatusAborted
}

// SetVariable sets a case-level variable
func (c *Case) SetVariable(name string, value interface{}) {
	c.Variables[name] = value
}

// GetVariable gets a case-level variable
func (c *Case) GetVariable(name string) (interface{}, bool) {
	value, exists := c.Variables[name]
	return value, exists
}

// SetMetadata sets metadata for the case
func (c *Case) SetMetadata(key string, value interface{}) {
	c.Metadata[key] = value
}

// GetMetadata gets metadata for the case
func (c *Case) GetMetadata(key string) (interface{}, bool) {
	value, exists := c.Metadata[key]
	return value, exists
}

// GetDuration returns the duration of the case execution
func (c *Case) GetDuration() time.Duration {
	if c.StartedAt == nil {
		return 0
	}
	
	endTime := time.Now()
	if c.CompletedAt != nil {
		endTime = *c.CompletedAt
	}
	
	return endTime.Sub(*c.StartedAt)
}

// Clone creates a copy of the case
func (c *Case) Clone() *Case {
	clone := &Case{
		ID:          c.ID,
		CPNID:       c.CPNID,
		Name:        c.Name,
		Description: c.Description,
		Status:      c.Status,
		CreatedAt:   c.CreatedAt,
		Variables:   make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}
	
	if c.StartedAt != nil {
		startedAt := *c.StartedAt
		clone.StartedAt = &startedAt
	}
	
	if c.CompletedAt != nil {
		completedAt := *c.CompletedAt
		clone.CompletedAt = &completedAt
	}
	
	if c.Marking != nil {
		clone.Marking = c.Marking.Clone()
	}
	
	// Copy variables
	for k, v := range c.Variables {
		clone.Variables[k] = v
	}
	
	// Copy metadata
	for k, v := range c.Metadata {
		clone.Metadata[k] = v
	}
	
	return clone
}

// String returns a string representation of the case
func (c *Case) String() string {
	duration := c.GetDuration()
	return fmt.Sprintf("Case{ID: %s, CPNID: %s, Name: %s, Status: %s, Duration: %v}", 
		c.ID, c.CPNID, c.Name, c.Status, duration)
}

// CaseFilter represents filters for case queries
type CaseFilter struct {
	CPNID     string     `json:"cpnId,omitempty"`
	Status    CaseStatus `json:"status,omitempty"`
	CreatedAfter  *time.Time `json:"createdAfter,omitempty"`
	CreatedBefore *time.Time `json:"createdBefore,omitempty"`
	Name      string     `json:"name,omitempty"`
}

// Matches checks if a case matches the filter criteria
func (f *CaseFilter) Matches(c *Case) bool {
	if f.CPNID != "" && c.CPNID != f.CPNID {
		return false
	}
	
	if f.Status != "" && c.Status != f.Status {
		return false
	}
	
	if f.CreatedAfter != nil && c.CreatedAt.Before(*f.CreatedAfter) {
		return false
	}
	
	if f.CreatedBefore != nil && c.CreatedAt.After(*f.CreatedBefore) {
		return false
	}
	
	if f.Name != "" && c.Name != f.Name {
		return false
	}
	
	return true
}

// CaseSortBy represents sorting options for cases
type CaseSortBy string

const (
	CaseSortByCreatedAt   CaseSortBy = "createdAt"
	CaseSortByStartedAt   CaseSortBy = "startedAt"
	CaseSortByCompletedAt CaseSortBy = "completedAt"
	CaseSortByName        CaseSortBy = "name"
	CaseSortByStatus      CaseSortBy = "status"
)

// CaseSort represents sorting configuration
type CaseSort struct {
	By        CaseSortBy `json:"by"`
	Ascending bool       `json:"ascending"`
}

// CaseQuery represents a query for cases
type CaseQuery struct {
	Filter *CaseFilter `json:"filter,omitempty"`
	Sort   *CaseSort   `json:"sort,omitempty"`
	Limit  int         `json:"limit,omitempty"`
	Offset int         `json:"offset,omitempty"`
}

