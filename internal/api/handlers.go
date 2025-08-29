package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	case_manager "go-petri-flow/internal/case"
	"go-petri-flow/internal/engine"
	"go-petri-flow/internal/expression"
	"go-petri-flow/internal/models"
	"go-petri-flow/internal/workitem"
)

// Server represents the API server
type Server struct {
	engine           *engine.Engine
	parser           *models.CPNParser
	cpns             map[string]*models.CPN     // CPN registry by ID
	states           map[string]*models.Marking // Current markings by CPN ID
	caseManager      *case_manager.Manager      // Case manager
	caseHandlers     *CaseHandlers              // Case API handlers
	workItemManager  *workitem.Manager          // Work item manager
	workItemHandlers *WorkItemHandlers          // Work item API handlers
}

// NewServer creates a new API server
func NewServer() *Server {
	engine := engine.NewEngine()
	caseManager := case_manager.NewManager(engine)
	workItemManager := workitem.NewManager(caseManager)

	server := &Server{
		engine:           engine,
		parser:           models.NewCPNParser(),
		cpns:             make(map[string]*models.CPN),
		states:           make(map[string]*models.Marking),
		caseManager:      caseManager,
		caseHandlers:     NewCaseHandlers(caseManager),
		workItemManager:  workItemManager,
		workItemHandlers: NewWorkItemHandlers(workItemManager),
	}

	return server
}

// Close closes the server and releases resources
func (s *Server) Close() {
	if s.engine != nil {
		s.engine.Close()
	}
}

// Response structures

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

type CPNListResponse struct {
	CPNs []CPNInfo `json:"cpns"`
}

type CPNInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"` // "loaded", "running", "completed"
}

type MarkingResponse struct {
	GlobalClock int                    `json:"globalClock"`
	Places      map[string][]TokenInfo `json:"places"`
}

type TokenInfo struct {
	Value     interface{} `json:"value"`
	Timestamp int         `json:"timestamp"`
}

type TransitionInfo struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Enabled          bool     `json:"enabled"`
	Kind             string   `json:"kind"`
	GuardExpression  string   `json:"guardExpression,omitempty"`
	Variables        []string `json:"variables,omitempty"`
	BindingCount     int      `json:"bindingCount"`
	ActionExpression string   `json:"actionExpression,omitempty"`
	FormSchema       string   `json:"formSchema,omitempty"`
	LayoutSchema     string   `json:"layoutSchema,omitempty"`
}

// EnabledTransitionDetail extends TransitionInfo with concrete bindings
type EnabledTransitionDetail struct {
	TransitionInfo
	Bindings []map[string]interface{} `json:"bindings"` // variable -> token value
}

type SimulationStepResponse struct {
	TransitionsFired int             `json:"transitionsFired"`
	Completed        bool            `json:"completed"`
	NewMarking       MarkingResponse `json:"newMarking"`
	CurrentStep      int             `json:"currentStep"`
}

// Helper functions

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, err string, message string) {
	s.writeJSON(w, status, ErrorResponse{
		Error:   err,
		Message: message,
	})
}

func (s *Server) writeSuccess(w http.ResponseWriter, data interface{}, message string) {
	s.writeJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

func (s *Server) getCPN(cpnID string) (*models.CPN, *models.Marking, error) {
	cpn, exists := s.cpns[cpnID]
	if !exists {
		return nil, nil, fmt.Errorf("CPN with ID %s not found", cpnID)
	}

	marking, exists := s.states[cpnID]
	if !exists {
		return nil, nil, fmt.Errorf("marking for CPN %s not found", cpnID)
	}

	return cpn, marking, nil
}

func (s *Server) markingToResponse(marking *models.Marking) MarkingResponse {
	places := make(map[string][]TokenInfo)
	for placeID, multiset := range marking.Places {
		all := multiset.GetAllTokens()
		infos := make([]TokenInfo, len(all))
		for i, tk := range all {
			infos[i] = TokenInfo{Value: tk.Value, Timestamp: tk.Timestamp}
		}
		places[placeID] = infos
	}
	return MarkingResponse{GlobalClock: marking.GlobalClock, Places: places}
}

// API Handlers

// LoadCPN loads a CPN from JSON definition
func (s *Server) LoadCPN(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	var cpnDef models.CPNDefinitionJSON
	if err := json.NewDecoder(r.Body).Decode(&cpnDef); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_json", "Failed to parse JSON: "+err.Error())
		return
	}

	cpn, err := s.parser.ParseCPNFromDefinition(&cpnDef) // Ensure marking exists when CPN loaded
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_cpn", "Failed to parse CPN: "+err.Error())
		return
	}

	// Store the CPN and its initial marking (reset step counter)
	s.cpns[cpn.ID] = cpn
	s.states[cpn.ID] = cpn.CreateInitialMarking() // Fix GetCPN to use stored marking directly

	// Register CPN with case manager
	s.caseManager.RegisterCPN(cpn)

	s.writeSuccess(w, CPNInfo{
		ID:          cpn.ID,
		Name:        cpn.Name,
		Description: cpn.Description,
		Status:      "loaded",
	}, "CPN loaded successfully")
}

// ListCPNs returns a list of all loaded CPNs
func (s *Server) ListCPNs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	var cpns []CPNInfo
	for _, cpn := range s.cpns {
		marking := s.states[cpn.ID]
		status := "loaded"
		if s.engine.IsCompleted(cpn, marking) {
			status = "completed"
		}

		cpns = append(cpns, CPNInfo{
			ID:          cpn.ID,
			Name:        cpn.Name,
			Description: cpn.Description,
			Status:      status,
		})
	}

	s.writeSuccess(w, CPNListResponse{CPNs: cpns}, "")
}

// GetCPN returns details of a specific CPN
func (s *Server) GetCPN(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	cpnID := r.URL.Query().Get("id")
	if cpnID == "" {
		s.writeError(w, http.StatusBadRequest, "missing_parameter", "CPN ID is required")
		return
	}

	cpn, exists := s.cpns[cpnID]
	if !exists {
		s.writeError(w, http.StatusNotFound, "cpn_not_found", "CPN with ID "+cpnID+" not found")
		return
	}

	// Convert CPN to JSON
	jsonData, err := s.parser.CPNToJSON(cpn)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "serialization_error", "Failed to serialize CPN: "+err.Error())
		return
	}

	var cpnData map[string]interface{}
	if err := json.Unmarshal(jsonData, &cpnData); err != nil {
		s.writeError(w, http.StatusInternalServerError, "serialization_error", "Failed to parse CPN JSON: "+err.Error())
		return
	}

	// Attach runtime status (global clock & current step) from stored state
	if marking, ok := s.states[cpnID]; ok {
		cpnData["globalClock"] = marking.GlobalClock
		cpnData["currentStep"] = marking.StepCounter
	}

	s.writeSuccess(w, cpnData, "")
}

// GetMarking returns the current marking of a CPN
func (s *Server) GetMarking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	cpnID := r.URL.Query().Get("id")
	if cpnID == "" {
		s.writeError(w, http.StatusBadRequest, "missing_parameter", "CPN ID is required")
		return
	}

	_, marking, err := s.getCPN(cpnID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "cpn_not_found", err.Error())
		return
	}

	s.writeSuccess(w, s.markingToResponse(marking), "")
}

// GetTransitions returns information about transitions in a CPN
func (s *Server) GetTransitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	cpnID := r.URL.Query().Get("id")
	if cpnID == "" {
		s.writeError(w, http.StatusBadRequest, "missing_parameter", "CPN ID is required")
		return
	}

	cpn, marking, err := s.getCPN(cpnID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "cpn_not_found", err.Error())
		return
	}

	// Get enabled transitions
	enabledTransitions, bindingsMap, err := s.engine.GetEnabledTransitions(cpn, marking)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "engine_error", "Failed to get enabled transitions: "+err.Error())
		return
	}

	enabledIDs := make(map[string]bool)
	for _, t := range enabledTransitions {
		enabledIDs[t.ID] = true
	}

	var transitions []TransitionInfo
	for _, transition := range cpn.Transitions {
		enabled := enabledIDs[transition.ID]
		bindingCount := 0
		if enabled {
			bindingCount = len(bindingsMap[transition.ID])
		}

		transitions = append(transitions, TransitionInfo{
			ID:               transition.ID,
			Name:             transition.Name,
			Enabled:          enabled,
			Kind:             string(transition.Kind),
			GuardExpression:  transition.GuardExpression,
			Variables:        transition.Variables,
			BindingCount:     bindingCount,
			ActionExpression: transition.ActionExpression,
			FormSchema:       transition.FormSchema,
			LayoutSchema:     transition.LayoutSchema,
		})
	}

	s.writeSuccess(w, transitions, "")
}

// GetEnabledTransitions returns only enabled transitions with full binding candidates
func (s *Server) GetEnabledTransitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	cpnID := r.URL.Query().Get("id")
	if cpnID == "" {
		s.writeError(w, http.StatusBadRequest, "missing_parameter", "CPN ID is required")
		return
	}

	cpn, marking, err := s.getCPN(cpnID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "cpn_not_found", err.Error())
		return
	}

	enabledTransitions, bindingsMap, err := s.engine.GetEnabledTransitions(cpn, marking)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "engine_error", "Failed to get enabled transitions: "+err.Error())
		return
	}

	var details []EnabledTransitionDetail
	for _, t := range enabledTransitions {
		bindings := bindingsMap[t.ID]
		bindingObjs := make([]map[string]interface{}, 0, len(bindings))
		for _, b := range bindings {
			obj := make(map[string]interface{})
			for varName, token := range b {
				obj[varName] = token.Value
			}
			bindingObjs = append(bindingObjs, obj)
		}
		details = append(details, EnabledTransitionDetail{
			TransitionInfo: TransitionInfo{
				ID:               t.ID,
				Name:             t.Name,
				Enabled:          true,
				Kind:             string(t.Kind),
				GuardExpression:  t.GuardExpression,
				Variables:        t.Variables,
				BindingCount:     len(bindings),
				ActionExpression: t.ActionExpression,
				FormSchema:       t.FormSchema,
				LayoutSchema:     t.LayoutSchema,
			},
			Bindings: bindingObjs,
		})
	}

	s.writeSuccess(w, details, "")
}

// FireTransition manually fires a specific transition
func (s *Server) FireTransition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	var request struct {
		CPNID        string                 `json:"cpnId"`
		TransitionID string                 `json:"transitionId"`
		BindingIndex int                    `json:"bindingIndex,omitempty"`
		FormData     map[string]interface{} `json:"formData,omitempty"` // Optional user-provided data for manual transitions
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_json", "Failed to parse JSON: "+err.Error())
		return
	}

	cpn, marking, err := s.getCPN(request.CPNID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "cpn_not_found", err.Error())
		return
	}

	transition := cpn.GetTransition(request.TransitionID)
	if transition == nil {
		s.writeError(w, http.StatusNotFound, "transition_not_found", "Transition with ID "+request.TransitionID+" not found")
		return
	}

	// Check if transition is enabled
	enabled, bindings, err := s.engine.IsEnabled(cpn, transition, marking)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "engine_error", "Failed to check if transition is enabled: "+err.Error())
		return
	}

	if !enabled {
		s.writeError(w, http.StatusBadRequest, "transition_not_enabled", "Transition "+transition.Name+" is not enabled")
		return
	}

	if request.BindingIndex >= len(bindings) {
		s.writeError(w, http.StatusBadRequest, "invalid_binding", "Binding index out of range")
		return
	}

	// Fire the transition; handle hierarchical call if subWorkflow link present.
	binding := bindings[request.BindingIndex]
	if sw := cpn.GetSubWorkflowByTransition(transition.ID); sw != nil {
		// Step 1: Fire inputs + action only (engine suppresses outputs automatically for hierarchical transitions)
		if err := s.engine.FireTransitionWithData(cpn, transition, binding, marking, request.FormData); err != nil {
			s.writeError(w, http.StatusInternalServerError, "engine_error", "Failed to fire hierarchical transition: "+err.Error())
			return
		}
		// Step 2: Execute child net (fresh instance) with autoStart semantics
		childCPN, ok := s.cpns[sw.CPNID]
		if !ok {
			s.writeError(w, http.StatusBadRequest, "child_cpn_missing", fmt.Sprintf("Child CPN %s not loaded", sw.CPNID))
			return
		}
		childMarking := childCPN.CreateInitialMarking()
		// Apply simple inputMapping: parent binding var -> child variable as temporary variable tokens (not yet token injection into places)
		// (Improvement pending: inject tokens directly in mapped child input places.)
		if sw.AutoStart {
			if _, err := s.engine.FireEnabledTransitions(childCPN, childMarking); err != nil {
				s.writeError(w, http.StatusInternalServerError, "child_autostart_failed", err.Error())
				return
			}
		}
		// Step 3: If propagateOnComplete and child completed, map outputs and produce deferred parent outputs now.
		if sw.PropagateOnComplete && s.engine.IsCompleted(childCPN, childMarking) {
			parentBinding := engine.TokenBinding{}
			// Build parentBinding from outputMapping; naive: pick first token from child output places for each mapping
			for childVar, parentVar := range sw.OutputMapping {
				var val interface{}
				// Search for a token value corresponding to childVar heuristic: first token anywhere (since variable scoping not tracked in CPN-level path)
				for placeID := range childMarking.Places {
					toks := childMarking.GetTokens(placeID)
					if len(toks) > 0 {
						val = toks[0].Value
						break
					}
				}
				if val != nil {
					parentBinding[parentVar] = models.NewToken(val, marking.GlobalClock)
				}
				_ = childVar
			}
			// Produce each output arc expression with built binding
			for _, arc := range cpn.GetOutputArcs(transition.ID) {
				if len(parentBinding) == 0 {
					break
				}
				ctx := expression.NewEvaluationContext()
				ctx.SetGlobalClock(marking.GlobalClock)
				for varName, tk := range parentBinding {
					ctx.BindVariable(varName, tk)
				}
				result, err := s.engine.EvaluatorAccessor().EvaluateArcExpression(arc.Expression, ctx)
				if err != nil {
					continue
				}
				place := cpn.GetPlace(arc.GetPlaceID())
				if place == nil {
					continue
				}
				newToken := models.NewToken(result, marking.GlobalClock)
				if err := place.ValidateToken(newToken); err != nil {
					continue
				}
				marking.AddToken(place.ID, newToken)
			}
		}
	} else {
		if err := s.engine.FireTransitionWithData(cpn, transition, binding, marking, request.FormData); err != nil {
			s.writeError(w, http.StatusInternalServerError, "engine_error", "Failed to fire transition: "+err.Error())
			return
		}
	}

	s.writeSuccess(w, s.markingToResponse(marking), "Transition "+transition.Name+" fired successfully")
}

// SimulateStep performs one simulation step
func (s *Server) SimulateStep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	cpnID := r.URL.Query().Get("id")
	if cpnID == "" {
		s.writeError(w, http.StatusBadRequest, "missing_parameter", "CPN ID is required")
		return
	}

	cpn, marking, err := s.getCPN(cpnID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "cpn_not_found", err.Error())
		return
	}

	// Perform simulation step
	firedCount, err := s.engine.SimulateStep(cpn, marking)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "engine_error", "Failed to simulate step: "+err.Error())
		return
	}

	completed := s.engine.IsCompleted(cpn, marking)

	response := SimulationStepResponse{
		TransitionsFired: firedCount,
		Completed:        completed,
		NewMarking:       s.markingToResponse(marking),
		CurrentStep:      marking.StepCounter,
	}

	s.writeSuccess(w, response, "")
}

// SimulateSteps performs multiple simulation steps
func (s *Server) SimulateSteps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	cpnID := r.URL.Query().Get("id")
	if cpnID == "" {
		s.writeError(w, http.StatusBadRequest, "missing_parameter", "CPN ID is required")
		return
	}

	stepsStr := r.URL.Query().Get("steps")
	steps := 1
	if stepsStr != "" {
		if parsed, err := strconv.Atoi(stepsStr); err == nil && parsed > 0 {
			steps = parsed
		}
	}

	cpn, marking, err := s.getCPN(cpnID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "cpn_not_found", err.Error())
		return
	}

	totalFired := 0
	for i := 0; i < steps; i++ {
		if s.engine.IsCompleted(cpn, marking) {
			break
		}

		firedCount, err := s.engine.SimulateStep(cpn, marking)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "engine_error", "Failed to simulate step: "+err.Error())
			return
		}

		totalFired += firedCount

		// If no transitions fired, stop simulation
		if firedCount == 0 {
			break
		}
	}

	completed := s.engine.IsCompleted(cpn, marking)

	response := SimulationStepResponse{
		TransitionsFired: totalFired,
		Completed:        completed,
		NewMarking:       s.markingToResponse(marking),
		CurrentStep:      marking.StepCounter,
	}

	s.writeSuccess(w, response, "")
}

// ResetCPN resets a CPN to its initial marking
func (s *Server) ResetCPN(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	cpnID := r.URL.Query().Get("id")
	if cpnID == "" {
		s.writeError(w, http.StatusBadRequest, "missing_parameter", "CPN ID is required")
		return
	}

	cpn, exists := s.cpns[cpnID]
	if !exists {
		s.writeError(w, http.StatusNotFound, "cpn_not_found", "CPN with ID "+cpnID+" not found")
		return
	}

	// Reset to initial marking
	s.states[cpnID] = cpn.CreateInitialMarking()

	s.writeSuccess(w, s.markingToResponse(s.states[cpnID]), "CPN reset to initial marking")
}

// DeleteCPN removes a CPN from the server
func (s *Server) DeleteCPN(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only DELETE method is allowed")
		return
	}

	cpnID := r.URL.Query().Get("id")
	if cpnID == "" {
		s.writeError(w, http.StatusBadRequest, "missing_parameter", "CPN ID is required")
		return
	}

	if _, exists := s.cpns[cpnID]; !exists {
		s.writeError(w, http.StatusNotFound, "cpn_not_found", "CPN with ID "+cpnID+" not found")
		return
	}

	delete(s.cpns, cpnID)
	delete(s.states, cpnID)

	// Unregister from case manager
	s.caseManager.UnregisterCPN(cpnID)

	s.writeSuccess(w, nil, "CPN deleted successfully")
}
