package api

import (
	"encoding/json"
	"net/http"
	"time"

	case_manager "go-petri-flow/internal/case"
	"go-petri-flow/internal/models"
)

// CaseHandlers contains handlers for case management endpoints
type CaseHandlers struct {
	caseManager *case_manager.Manager
}

// NewCaseHandlers creates new case handlers
func NewCaseHandlers(caseManager *case_manager.Manager) *CaseHandlers {
	return &CaseHandlers{
		caseManager: caseManager,
	}
}

// Request/Response structures for case management

type CreateCaseRequest struct {
	ID          string                 `json:"id"`
	CPNID       string                 `json:"cpnId"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
}

type UpdateCaseRequest struct {
	Variables map[string]interface{} `json:"variables,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type FireTransitionRequest struct {
	TransitionID string `json:"transitionId"`
	BindingIndex int    `json:"bindingIndex,omitempty"`
}

type CaseResponse struct {
	ID          string                 `json:"id"`
	CPNID       string                 `json:"cpnId"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"createdAt"`
	StartedAt   *time.Time             `json:"startedAt,omitempty"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	Duration    float64                `json:"duration"` // Duration in seconds
	Variables   map[string]interface{} `json:"variables"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type CaseListResponse struct {
	Cases  []CaseResponse `json:"cases"`
	Total  int            `json:"total"`
	Offset int            `json:"offset"`
	Limit  int            `json:"limit"`
}

type CaseExecutionResponse struct {
	TransitionsFired int             `json:"transitionsFired"`
	Completed        bool            `json:"completed"`
	NewMarking       MarkingResponse `json:"newMarking"`
}

// ExecuteAll executes automatic transitions until quiescent for a case
func (h *CaseHandlers) ExecuteAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}
	caseID := r.URL.Query().Get("id")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}
	firedCount, err := h.caseManager.ExecuteAll(caseID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "execution_failed", err.Error())
		return
	}
	case_, err := h.caseManager.GetCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}
	var markingResponse MarkingResponse
	if case_.Marking != nil {
		places := make(map[string][]TokenInfo)
		for placeName, multiset := range case_.Marking.Places {
			tokens := multiset.GetAllTokens()
			tokenInfos := make([]TokenInfo, len(tokens))
			for i, token := range tokens {
				tokenInfos[i] = TokenInfo{Value: token.Value, Timestamp: token.Timestamp}
			}
			places[placeName] = tokenInfos
		}
		markingResponse = MarkingResponse{GlobalClock: case_.Marking.GlobalClock, Places: places}
	}
	response := CaseExecutionResponse{TransitionsFired: firedCount, Completed: case_.IsCompleted(), NewMarking: markingResponse}
	h.writeSuccess(w, response, "")
}

// Helper functions

func (h *CaseHandlers) caseToResponse(case_ *models.Case) CaseResponse {
	response := CaseResponse{
		ID:          case_.ID,
		CPNID:       case_.CPNID,
		Name:        case_.Name,
		Description: case_.Description,
		Status:      string(case_.Status),
		CreatedAt:   case_.CreatedAt,
		Duration:    case_.GetDuration().Seconds(),
		Variables:   case_.Variables,
		Metadata:    case_.Metadata,
	}

	if case_.StartedAt != nil {
		response.StartedAt = case_.StartedAt
	}

	if case_.CompletedAt != nil {
		response.CompletedAt = case_.CompletedAt
	}

	return response
}

func (h *CaseHandlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *CaseHandlers) writeError(w http.ResponseWriter, status int, err string, message string) {
	h.writeJSON(w, status, ErrorResponse{
		Error:   err,
		Message: message,
	})
}

func (h *CaseHandlers) writeSuccess(w http.ResponseWriter, data interface{}, message string) {
	h.writeJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// API Handlers

// CreateCase creates a new case
func (h *CaseHandlers) CreateCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	var request CreateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_json", "Failed to parse JSON: "+err.Error())
		return
	}

	// Validate required fields
	if request.ID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_field", "Case ID is required")
		return
	}
	if request.CPNID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_field", "CPN ID is required")
		return
	}
	if request.Name == "" {
		h.writeError(w, http.StatusBadRequest, "missing_field", "Case name is required")
		return
	}

	// Create the case
	case_, err := h.caseManager.CreateCase(request.ID, request.CPNID, request.Name, request.Description, request.Variables)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "creation_failed", "Failed to create case: "+err.Error())
		return
	}

	h.writeSuccess(w, h.caseToResponse(case_), "Case created successfully")
}

// GetCase retrieves a case by ID
func (h *CaseHandlers) GetCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	caseID := r.URL.Query().Get("id")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}

	case_, err := h.caseManager.GetCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "case_not_found", err.Error())
		return
	}

	h.writeSuccess(w, h.caseToResponse(case_), "")
}

// UpdateCase updates case variables and metadata
func (h *CaseHandlers) UpdateCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only PUT method is allowed")
		return
	}

	caseID := r.URL.Query().Get("id")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}

	var request UpdateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_json", "Failed to parse JSON: "+err.Error())
		return
	}

	err := h.caseManager.UpdateCase(caseID, request.Variables, request.Metadata)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "update_failed", err.Error())
		return
	}

	// Get updated case
	case_, err := h.caseManager.GetCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.caseToResponse(case_), "Case updated successfully")
}

// StartCase starts a case execution
func (h *CaseHandlers) StartCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	caseID := r.URL.Query().Get("id")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}

	err := h.caseManager.StartCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "start_failed", err.Error())
		return
	}

	// Get updated case
	case_, err := h.caseManager.GetCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.caseToResponse(case_), "Case started successfully")
}

// SuspendCase suspends a case execution
func (h *CaseHandlers) SuspendCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	caseID := r.URL.Query().Get("id")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}

	err := h.caseManager.SuspendCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "suspend_failed", err.Error())
		return
	}

	// Get updated case
	case_, err := h.caseManager.GetCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.caseToResponse(case_), "Case suspended successfully")
}

// ResumeCase resumes a case execution
func (h *CaseHandlers) ResumeCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	caseID := r.URL.Query().Get("id")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}

	err := h.caseManager.ResumeCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "resume_failed", err.Error())
		return
	}

	// Get updated case
	case_, err := h.caseManager.GetCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.caseToResponse(case_), "Case resumed successfully")
}

// AbortCase aborts a case execution
func (h *CaseHandlers) AbortCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	caseID := r.URL.Query().Get("id")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}

	err := h.caseManager.AbortCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "abort_failed", err.Error())
		return
	}

	// Get updated case
	case_, err := h.caseManager.GetCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.caseToResponse(case_), "Case aborted successfully")
}

// DeleteCase deletes a case
func (h *CaseHandlers) DeleteCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only DELETE method is allowed")
		return
	}

	caseID := r.URL.Query().Get("id")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}

	err := h.caseManager.DeleteCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "delete_failed", err.Error())
		return
	}

	h.writeSuccess(w, nil, "Case deleted successfully")
}

// ExecuteStep executes one simulation step for a case
func (h *CaseHandlers) ExecuteStep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	caseID := r.URL.Query().Get("id")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}

	firedCount, err := h.caseManager.ExecuteStep(caseID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "execution_failed", err.Error())
		return
	}

	// Get updated case to check completion status and get marking
	case_, err := h.caseManager.GetCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	// Convert marking to response format
	var markingResponse MarkingResponse
	if case_.Marking != nil {
		places := make(map[string][]TokenInfo)
		for placeName, multiset := range case_.Marking.Places {
			tokens := multiset.GetAllTokens()
			tokenInfos := make([]TokenInfo, len(tokens))
			for i, token := range tokens {
				tokenInfos[i] = TokenInfo{
					Value:     token.Value,
					Timestamp: token.Timestamp,
				}
			}
			places[placeName] = tokenInfos
		}
		markingResponse = MarkingResponse{
			GlobalClock: case_.Marking.GlobalClock,
			Places:      places,
		}
	}

	response := CaseExecutionResponse{
		TransitionsFired: firedCount,
		Completed:        case_.IsCompleted(),
		NewMarking:       markingResponse,
	}

	h.writeSuccess(w, response, "")
}

// FireTransition fires a specific transition for a case
func (h *CaseHandlers) FireTransition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	caseID := r.URL.Query().Get("id")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}

	var request FireTransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_json", "Failed to parse JSON: "+err.Error())
		return
	}

	if request.TransitionID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_field", "Transition ID is required")
		return
	}

	err := h.caseManager.FireTransition(caseID, request.TransitionID, request.BindingIndex)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "fire_failed", err.Error())
		return
	}

	// Get updated case
	case_, err := h.caseManager.GetCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.caseToResponse(case_), "Transition fired successfully")
}

// GetCaseMarking returns the current marking of a case
func (h *CaseHandlers) GetCaseMarking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	caseID := r.URL.Query().Get("id")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}

	case_, err := h.caseManager.GetCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "case_not_found", err.Error())
		return
	}

	if case_.Marking == nil {
		h.writeError(w, http.StatusBadRequest, "no_marking", "Case has no marking (not started)")
		return
	}

	// Convert marking to response format
	places := make(map[string][]TokenInfo)
	for placeName, multiset := range case_.Marking.Places {
		tokens := multiset.GetAllTokens()
		tokenInfos := make([]TokenInfo, len(tokens))
		for i, token := range tokens {
			tokenInfos[i] = TokenInfo{
				Value:     token.Value,
				Timestamp: token.Timestamp,
			}
		}
		places[placeName] = tokenInfos
	}

	markingResponse := MarkingResponse{
		GlobalClock: case_.Marking.GlobalClock,
		Places:      places,
	}

	h.writeSuccess(w, markingResponse, "")
}

// GetCaseTransitions returns enabled transitions for a case
func (h *CaseHandlers) GetCaseTransitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	caseID := r.URL.Query().Get("id")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}

	enabledTransitions, bindingsMap, err := h.caseManager.GetEnabledTransitions(caseID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "get_transitions_failed", err.Error())
		return
	}

	var transitions []TransitionInfo
	for _, transition := range enabledTransitions {
		bindingCount := len(bindingsMap[transition.ID])
		transitions = append(transitions, TransitionInfo{
			ID:               transition.ID,
			Name:             transition.Name,
			Enabled:          true,
			Kind:             string(transition.Kind),
			GuardExpression:  transition.GuardExpression,
			Variables:        transition.Variables,
			BindingCount:     bindingCount,
			ActionExpression: transition.ActionExpression,
			FormSchema:       transition.FormSchema,
			LayoutSchema:     transition.LayoutSchema,
		})
	}

	h.writeSuccess(w, transitions, "")
}

// GetCaseEnabledTransitions returns only enabled transitions with explicit binding candidates for a case.
func (h *CaseHandlers) GetCaseEnabledTransitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}
	caseID := r.URL.Query().Get("id")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}
	case_, err := h.caseManager.GetCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "case_not_found", err.Error())
		return
	}
	if case_.Marking == nil {
		h.writeError(w, http.StatusBadRequest, "no_marking", "Case has no marking (not started)")
		return
	}
	// Use engine through case manager
	transitions, bindingsMap, err := h.caseManager.GetEnabledTransitions(caseID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "get_enabled_failed", err.Error())
		return
	}
	type EnabledCaseTransition struct {
		ID               string                   `json:"id"`
		Name             string                   `json:"name"`
		Kind             string                   `json:"kind"`
		GuardExpression  string                   `json:"guardExpression,omitempty"`
		ActionExpression string                   `json:"actionExpression,omitempty"`
		FormSchema       string                   `json:"formSchema,omitempty"`
		LayoutSchema     string                   `json:"layoutSchema,omitempty"`
		Bindings         []map[string]interface{} `json:"bindings"`
	}
	var result []EnabledCaseTransition
	for _, t := range transitions {
		bList := bindingsMap[t.ID]
		ect := EnabledCaseTransition{
			ID:               t.ID,
			Name:             t.Name,
			Kind:             string(t.Kind),
			GuardExpression:  t.GuardExpression,
			ActionExpression: t.ActionExpression,
			FormSchema:       t.FormSchema,
			LayoutSchema:     t.LayoutSchema,
		}
		for _, bind := range bList {
			row := make(map[string]interface{})
			for varName, token := range bind {
				if token != nil {
					row[varName] = token.Value
				}
			}
			ect.Bindings = append(ect.Bindings, row)
		}
		result = append(result, ect)
	}
	h.writeSuccess(w, result, "")
}

// QueryCases queries cases based on filter criteria
func (h *CaseHandlers) QueryCases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	var query models.CaseQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_json", "Failed to parse JSON: "+err.Error())
		return
	}

	cases, err := h.caseManager.QueryCases(&query)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	var caseResponses []CaseResponse
	for _, case_ := range cases {
		caseResponses = append(caseResponses, h.caseToResponse(case_))
	}

	response := CaseListResponse{
		Cases:  caseResponses,
		Total:  len(caseResponses),
		Offset: query.Offset,
		Limit:  query.Limit,
	}

	h.writeSuccess(w, response, "")
}

// GetCaseStatistics returns statistics about cases
func (h *CaseHandlers) GetCaseStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	stats := h.caseManager.GetCaseStatistics()
	h.writeSuccess(w, stats, "")
}
