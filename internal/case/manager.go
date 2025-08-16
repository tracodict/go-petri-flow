package case_manager

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"go-petri-flow/internal/engine"
	"go-petri-flow/internal/models"
)

// Manager handles case lifecycle management
type Manager struct {
	cases  map[string]*models.Case // Case ID -> Case
	cpns   map[string]*models.CPN  // CPN ID -> CPN
	engine *engine.Engine
	mutex  sync.RWMutex
}

// NewManager creates a new case manager
func NewManager(engine *engine.Engine) *Manager {
	return &Manager{
		cases:  make(map[string]*models.Case),
		cpns:   make(map[string]*models.CPN),
		engine: engine,
	}
}

// RegisterCPN registers a CPN for case management
func (m *Manager) RegisterCPN(cpn *models.CPN) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.cpns[cpn.ID] = cpn
}

// UnregisterCPN unregisters a CPN
func (m *Manager) UnregisterCPN(cpnID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.cpns, cpnID)
	
	// Remove all cases for this CPN
	for caseID, case_ := range m.cases {
		if case_.CPNID == cpnID {
			delete(m.cases, caseID)
		}
	}
}

// CreateCase creates a new case instance
func (m *Manager) CreateCase(caseID, cpnID, name, description string, variables map[string]interface{}) (*models.Case, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check if CPN exists
	_, exists := m.cpns[cpnID]
	if !exists {
		return nil, fmt.Errorf("CPN with ID %s not found", cpnID)
	}
	
	// Check if case ID already exists
	if _, exists := m.cases[caseID]; exists {
		return nil, fmt.Errorf("case with ID %s already exists", caseID)
	}
	
	// Create new case
	case_ := models.NewCase(caseID, cpnID, name, description)
	
	// Set variables if provided
	if variables != nil {
		for k, v := range variables {
			case_.SetVariable(k, v)
		}
	}
	
	// Store the case
	m.cases[caseID] = case_
	
	return case_, nil
}

// StartCase starts a case execution
func (m *Manager) StartCase(caseID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	case_, exists := m.cases[caseID]
	if !exists {
		return fmt.Errorf("case with ID %s not found", caseID)
	}
	
	if case_.Status != models.CaseStatusCreated {
		return fmt.Errorf("case %s is not in CREATED status, current status: %s", caseID, case_.Status)
	}
	
	cpn, exists := m.cpns[case_.CPNID]
	if !exists {
		return fmt.Errorf("CPN with ID %s not found", case_.CPNID)
	}
	
	// Create initial marking
	initialMarking := cpn.CreateInitialMarking()
	
	// Start the case
	case_.Start(initialMarking)
	
	return nil
}

// GetCase retrieves a case by ID
func (m *Manager) GetCase(caseID string) (*models.Case, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	case_, exists := m.cases[caseID]
	if !exists {
		return nil, fmt.Errorf("case with ID %s not found", caseID)
	}
	
	return case_.Clone(), nil
}

// UpdateCase updates case metadata and variables
func (m *Manager) UpdateCase(caseID string, variables map[string]interface{}, metadata map[string]interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	case_, exists := m.cases[caseID]
	if !exists {
		return fmt.Errorf("case with ID %s not found", caseID)
	}
	
	// Update variables
	if variables != nil {
		for k, v := range variables {
			case_.SetVariable(k, v)
		}
	}
	
	// Update metadata
	if metadata != nil {
		for k, v := range metadata {
			case_.SetMetadata(k, v)
		}
	}
	
	return nil
}

// SuspendCase suspends a case execution
func (m *Manager) SuspendCase(caseID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	case_, exists := m.cases[caseID]
	if !exists {
		return fmt.Errorf("case with ID %s not found", caseID)
	}
	
	case_.Suspend()
	return nil
}

// ResumeCase resumes a case execution
func (m *Manager) ResumeCase(caseID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	case_, exists := m.cases[caseID]
	if !exists {
		return fmt.Errorf("case with ID %s not found", caseID)
	}
	
	case_.Resume()
	return nil
}

// AbortCase aborts a case execution
func (m *Manager) AbortCase(caseID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	case_, exists := m.cases[caseID]
	if !exists {
		return fmt.Errorf("case with ID %s not found", caseID)
	}
	
	case_.Abort()
	return nil
}

// DeleteCase deletes a case
func (m *Manager) DeleteCase(caseID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	case_, exists := m.cases[caseID]
	if !exists {
		return fmt.Errorf("case with ID %s not found", caseID)
	}
	
	// Only allow deletion of terminated cases
	if !case_.IsTerminated() {
		return fmt.Errorf("cannot delete active case %s, current status: %s", caseID, case_.Status)
	}
	
	delete(m.cases, caseID)
	return nil
}

// ExecuteStep executes one simulation step for a case
func (m *Manager) ExecuteStep(caseID string) (int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	case_, exists := m.cases[caseID]
	if !exists {
		return 0, fmt.Errorf("case with ID %s not found", caseID)
	}
	
	if case_.Status != models.CaseStatusRunning {
		return 0, fmt.Errorf("case %s is not running, current status: %s", caseID, case_.Status)
	}
	
	cpn, exists := m.cpns[case_.CPNID]
	if !exists {
		return 0, fmt.Errorf("CPN with ID %s not found", case_.CPNID)
	}
	
	// Execute simulation step
	firedCount, err := m.engine.SimulateStep(cpn, case_.Marking)
	if err != nil {
		return 0, fmt.Errorf("failed to execute simulation step: %v", err)
	}
	
	// Check if case is completed
	if m.engine.IsCompleted(cpn, case_.Marking) {
		case_.Complete()
	}
	
	return firedCount, nil
}

// FireTransition fires a specific transition for a case
func (m *Manager) FireTransition(caseID, transitionID string, bindingIndex int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	case_, exists := m.cases[caseID]
	if !exists {
		return fmt.Errorf("case with ID %s not found", caseID)
	}
	
	if case_.Status != models.CaseStatusRunning {
		return fmt.Errorf("case %s is not running, current status: %s", caseID, case_.Status)
	}
	
	cpn, exists := m.cpns[case_.CPNID]
	if !exists {
		return fmt.Errorf("CPN with ID %s not found", case_.CPNID)
	}
	
	transition := cpn.GetTransition(transitionID)
	if transition == nil {
		return fmt.Errorf("transition with ID %s not found", transitionID)
	}
	
	// Check if transition is enabled
	enabled, bindings, err := m.engine.IsEnabled(cpn, transition, case_.Marking)
	if err != nil {
		return fmt.Errorf("failed to check if transition is enabled: %v", err)
	}
	
	if !enabled {
		return fmt.Errorf("transition %s is not enabled", transitionID)
	}
	
	if bindingIndex >= len(bindings) {
		return fmt.Errorf("binding index %d out of range", bindingIndex)
	}
	
	// Fire the transition
	binding := bindings[bindingIndex]
	if err := m.engine.FireTransition(cpn, transition, binding, case_.Marking); err != nil {
		return fmt.Errorf("failed to fire transition: %v", err)
	}
	
	// Check if case is completed
	if m.engine.IsCompleted(cpn, case_.Marking) {
		case_.Complete()
	}
	
	return nil
}

// GetEnabledTransitions returns enabled transitions for a case
func (m *Manager) GetEnabledTransitions(caseID string) ([]*models.Transition, map[string][]engine.TokenBinding, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	case_, exists := m.cases[caseID]
	if !exists {
		return nil, nil, fmt.Errorf("case with ID %s not found", caseID)
	}
	
	if case_.Status != models.CaseStatusRunning {
		return nil, nil, fmt.Errorf("case %s is not running, current status: %s", caseID, case_.Status)
	}
	
	cpn, exists := m.cpns[case_.CPNID]
	if !exists {
		return nil, nil, fmt.Errorf("CPN with ID %s not found", case_.CPNID)
	}
	
	return m.engine.GetEnabledTransitions(cpn, case_.Marking)
}

// QueryCases queries cases based on filter criteria
func (m *Manager) QueryCases(query *models.CaseQuery) ([]*models.Case, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	var result []*models.Case
	
	// Apply filter
	for _, case_ := range m.cases {
		if query.Filter == nil || query.Filter.Matches(case_) {
			result = append(result, case_.Clone())
		}
	}
	
	// Apply sorting
	if query.Sort != nil {
		m.sortCases(result, query.Sort)
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

// sortCases sorts cases based on the sort configuration
func (m *Manager) sortCases(cases []*models.Case, sortConfig *models.CaseSort) {
	sort.Slice(cases, func(i, j int) bool {
		var less bool
		
		switch sortConfig.By {
		case models.CaseSortByCreatedAt:
			less = cases[i].CreatedAt.Before(cases[j].CreatedAt)
		case models.CaseSortByStartedAt:
			if cases[i].StartedAt == nil && cases[j].StartedAt == nil {
				less = false
			} else if cases[i].StartedAt == nil {
				less = true
			} else if cases[j].StartedAt == nil {
				less = false
			} else {
				less = cases[i].StartedAt.Before(*cases[j].StartedAt)
			}
		case models.CaseSortByCompletedAt:
			if cases[i].CompletedAt == nil && cases[j].CompletedAt == nil {
				less = false
			} else if cases[i].CompletedAt == nil {
				less = true
			} else if cases[j].CompletedAt == nil {
				less = false
			} else {
				less = cases[i].CompletedAt.Before(*cases[j].CompletedAt)
			}
		case models.CaseSortByName:
			less = cases[i].Name < cases[j].Name
		case models.CaseSortByStatus:
			less = string(cases[i].Status) < string(cases[j].Status)
		default:
			less = cases[i].CreatedAt.Before(cases[j].CreatedAt)
		}
		
		if !sortConfig.Ascending {
			less = !less
		}
		
		return less
	})
}

// GetCaseStatistics returns statistics about cases
func (m *Manager) GetCaseStatistics() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	stats := map[string]interface{}{
		"total":     len(m.cases),
		"byStatus":  make(map[models.CaseStatus]int),
		"byCPN":     make(map[string]int),
		"avgDuration": 0.0,
	}
	
	var totalDuration time.Duration
	completedCount := 0
	
	for _, case_ := range m.cases {
		// Count by status
		statusCounts := stats["byStatus"].(map[models.CaseStatus]int)
		statusCounts[case_.Status]++
		
		// Count by CPN
		cpnCounts := stats["byCPN"].(map[string]int)
		cpnCounts[case_.CPNID]++
		
		// Calculate average duration for completed cases
		if case_.IsCompleted() {
			totalDuration += case_.GetDuration()
			completedCount++
		}
	}
	
	if completedCount > 0 {
		avgDuration := totalDuration / time.Duration(completedCount)
		stats["avgDuration"] = avgDuration.Seconds()
	}
	
	return stats
}

// GetActiveCases returns all active cases
func (m *Manager) GetActiveCases() []*models.Case {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	var activeCases []*models.Case
	for _, case_ := range m.cases {
		if case_.IsActive() {
			activeCases = append(activeCases, case_.Clone())
		}
	}
	
	return activeCases
}

// GetCaseCount returns the total number of cases
func (m *Manager) GetCaseCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.cases)
}

