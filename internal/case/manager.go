package case_manager

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"go-petri-flow/internal/engine"
	"go-petri-flow/internal/expression"
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
	for k, v := range variables {
		case_.SetVariable(k, v)
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
	for k, v := range variables {
		case_.SetVariable(k, v)
	}

	// Update metadata
	for k, v := range metadata {
		case_.SetMetadata(k, v)
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
		// If this is a child case, propagate to parent
		if case_.ParentCaseID != "" {
			m.propagateChildCompletion(case_)
		}
	}

	return firedCount, nil
}

// ExecuteAll executes automatic transitions repeatedly until quiescent for a case
func (m *Manager) ExecuteAll(caseID string) (int, error) {
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

	firedCount, err := m.engine.FireEnabledTransitions(cpn, case_.Marking)
	if err != nil {
		return 0, fmt.Errorf("failed to execute all automatic transitions: %v", err)
	}
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

	// Determine if this is a hierarchical call transition
	sw := cpn.GetSubWorkflowByTransition(transitionID)
	binding := bindings[bindingIndex]
	if sw != nil {
		if err := m.fireSubWorkflowTransition(case_, cpn, transition, sw, binding); err != nil {
			return err
		}
	} else {
		// Fire normally
		if err := m.engine.FireTransition(cpn, transition, binding, case_.Marking); err != nil {
			return fmt.Errorf("failed to fire transition: %v", err)
		}
	}

	// Check if case is completed
	if m.engine.IsCompleted(cpn, case_.Marking) {
		case_.Complete()
		if case_.ParentCaseID != "" {
			m.propagateChildCompletion(case_)
		}
	}

	return nil
}

// fireSubWorkflowTransition handles hierarchical call semantics
func (m *Manager) fireSubWorkflowTransition(parentCase *models.Case, parentCPN *models.CPN, transition *models.Transition, sw *models.SubWorkflowLink, binding engine.TokenBinding) error {
	// Consume parent inputs & execute action.
	// Only suppress (defer) output arcs if PropagateOnComplete is true; otherwise allow normal firing.

	originalOutputs := parentCPN.GetOutputArcs(transition.ID)
	suppressed := false
	if sw.PropagateOnComplete {
		suppressed = true
		// Temporarily remove output arcs so engine skips producing them now.
		var remaining []*models.Arc
		for _, arc := range parentCPN.Arcs {
			skip := false
			if arc.IsOutputArc() && arc.GetTransitionID() == transition.ID {
				skip = true
			}
			if !skip {
				remaining = append(remaining, arc)
			}
		}
		parentCPN.Arcs = remaining
	}

	// Fire transition (inputs + action (+ outputs if not suppressed))
	if err := m.engine.FireTransition(parentCPN, transition, binding, parentCase.Marking); err != nil {
		if suppressed { // restore arcs on error
			parentCPN.Arcs = append(parentCPN.Arcs, originalOutputs...)
		}
		return fmt.Errorf("failed to fire hierarchical transition (inputs/action): %v", err)
	}
	if suppressed {
		// Restore arcs immediately so future enablement checks see them.
		parentCPN.Arcs = append(parentCPN.Arcs, originalOutputs...)
	}

	// Step 2: Spawn child case
	childSeq := len(parentCase.Children) + 1
	childCaseID := fmt.Sprintf("%s:%s:%d", parentCase.ID, sw.ID, childSeq)
	// Create case
	childCase := models.NewCase(childCaseID, sw.CPNID, fmt.Sprintf("Child %s of %s", sw.CPNID, parentCase.ID), "")
	childCase.ParentCaseID = parentCase.ID
	// Register in manager
	m.cases[childCaseID] = childCase
	parentCase.Children = append(parentCase.Children, childCaseID)

	childCPN, exists := m.cpns[sw.CPNID]
	if !exists {
		return fmt.Errorf("child CPN %s not loaded", sw.CPNID)
	}
	// Create child initial marking (clone of defined initial marking)
	childMarking := childCPN.CreateInitialMarking()
	// Apply input mapping: parent variable -> child variable by creating bound variable tokens
	// For now we interpret variables as binding variable names; we need the token values from binding
	for parentVar, childVar := range sw.InputMapping {
		if tk, ok := binding[parentVar]; ok && tk != nil {
			// Store as child case variable for later arc expressions; also create a pseudo token variable? We'll set case variable.
			childCase.SetVariable(childVar, tk.Value)
		}
	}
	childCase.Start(childMarking)

	// Auto-start execution if configured
	if sw.AutoStart {
		// Fire automatic transitions to quiescence
		if _, err := m.engine.FireEnabledTransitions(childCPN, childCase.Marking); err != nil {
			return fmt.Errorf("failed autoStart child case %s: %v", childCaseID, err)
		}
		if m.engine.IsCompleted(childCPN, childCase.Marking) {
			childCase.Complete()
		}
	}

	// If outputs were suppressed we record them for deferred emission
	if suppressed {
		defListKey := "_deferredOutputs"
		var list []map[string]string
		if v, ok := parentCase.Metadata[defListKey]; ok {
			if existing, ok2 := v.([]map[string]string); ok2 {
				list = existing
			}
		}
		for _, arc := range originalOutputs {
			if !arc.IsOutputArc() {
				continue
			}
			list = append(list, map[string]string{"transitionId": transition.ID, "arcId": arc.ID, "childCaseId": childCaseID})
		}
		parentCase.Metadata[defListKey] = list
	}

	// If child finished during autoStart and we suppressed outputs, propagate immediately
	if suppressed && childCase.IsCompleted() {
		m.propagateChildCompletion(childCase)
	}

	return nil
}

// propagateChildCompletion emits deferred outputs for a completed child case
func (m *Manager) propagateChildCompletion(child *models.Case) {
	parentCase, ok := m.cases[child.ParentCaseID]
	if !ok {
		return
	}
	parentCPN := m.cpns[parentCase.CPNID]
	defListKey := "_deferredOutputs"
	raw, ok := parentCase.Metadata[defListKey]
	if !ok {
		return
	}
	list, ok := raw.([]map[string]string)
	if !ok {
		return
	}
	// Rebuild new list excluding processed entries
	var remaining []map[string]string
	for _, entry := range list {
		if entry["childCaseId"] != child.ID { // keep entries for other children
			remaining = append(remaining, entry)
			continue
		}
		tID := entry["transitionId"]
		arcID := entry["arcId"]
		transition := parentCPN.GetTransition(tID)
		if transition == nil {
			continue
		}
		arc := parentCPN.GetArc(arcID)
		if arc == nil {
			continue
		}
		// Build binding from output mapping
		sw := parentCPN.GetSubWorkflowByTransition(tID)
		if sw == nil {
			continue
		}
		// Only propagate if this link requested deferral / propagation on completion
		if !sw.PropagateOnComplete {
			continue
		}
		b := engine.TokenBinding{}
		// Extract child values
		for childVar, parentVar := range sw.OutputMapping {
			var val interface{}
			if v, ok := child.Variables[childVar]; ok {
				val = v
			} else if child.Marking != nil {
				// Scan all places for first token value
				for placeID := range child.Marking.Places {
					toks := child.Marking.GetTokens(placeID)
					if len(toks) > 0 {
						val = toks[0].Value
						break
					}
				}
			}
			if val == nil {
				continue
			}
			b[parentVar] = models.NewToken(val, parentCase.Marking.GlobalClock)
		}
		// Produce all output arcs of the transition (not just one arc) to honor original semantics
		outArcs := parentCPN.GetOutputArcs(tID)
		_ = outArcs
		// For simplicity, only produce for arcs matching arcID (deferred entry) now
		if bindingTokenCount := len(b); bindingTokenCount > 0 {
			// Use engine to produce tokens on this arc expression for each variable binding
			// Reuse a mini context by firing artificial output production
			// Implement quick re-eval similar to engine.processOutputArc
			// We'll call a helper function produceSingleArc
			m.produceSingleArc(parentCPN, arc, b, parentCase.Marking)
		}
	}
	if len(remaining) == 0 {
		delete(parentCase.Metadata, defListKey)
	} else {
		parentCase.Metadata[defListKey] = remaining
	}
}

// produceSingleArc replicates engine output arc logic (simplified) for deferred emission
func (m *Manager) produceSingleArc(cpn *models.CPN, arc *models.Arc, binding engine.TokenBinding, marking *models.Marking) {
	// Reuse engine's evaluator indirectly: create a fresh evaluation context
	ctx := expression.NewEvaluationContext()
	ctx.SetGlobalClock(marking.GlobalClock)
	for varName, token := range binding {
		ctx.BindVariable(varName, token)
	}
	// Evaluate expression
	result, err := m.engineEvaluator().EvaluateArcExpression(arc.Expression, ctx)
	if err != nil {
		return
	}
	place := cpn.GetPlace(arc.GetPlaceID())
	if place == nil {
		return
	}
	newToken := models.NewToken(result, marking.GlobalClock)
	if err := place.ValidateToken(newToken); err != nil {
		return
	}
	marking.AddToken(place.ID, newToken)
}

// engineEvaluator exposes underlying evaluator (package-private compromise)
func (m *Manager) engineEvaluator() *expression.Evaluator { return m.engine.EvaluatorAccessor() }

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
		"total":       len(m.cases),
		"byStatus":    make(map[models.CaseStatus]int),
		"byCPN":       make(map[string]int),
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
