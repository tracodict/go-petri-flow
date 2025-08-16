package api

import (
	"encoding/json"
	"net/http"
	"time"

	"go-petri-flow/internal/models"
	"go-petri-flow/internal/workitem"
)

// WorkItemHandlers contains handlers for work item management endpoints
type WorkItemHandlers struct {
	workItemManager *workitem.Manager
}

// NewWorkItemHandlers creates new work item handlers
func NewWorkItemHandlers(workItemManager *workitem.Manager) *WorkItemHandlers {
	return &WorkItemHandlers{
		workItemManager: workItemManager,
	}
}

// Request/Response structures for work item management

type CreateWorkItemRequest struct {
	ID           string `json:"id"`
	CaseID       string `json:"caseId"`
	TransitionID string `json:"transitionId"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	BindingIndex int    `json:"bindingIndex,omitempty"`
}

type UpdateWorkItemRequest struct {
	Data     map[string]interface{} `json:"data,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type SetPriorityRequest struct {
	Priority string `json:"priority"`
}

type SetDueDateRequest struct {
	DueDate *time.Time `json:"dueDate"`
}

type OfferWorkItemRequest struct {
	UserIDs []string `json:"userIds"`
}

type AllocateWorkItemRequest struct {
	UserID string `json:"userId"`
}

type WorkItemResponse struct {
	ID           string                 `json:"id"`
	CaseID       string                 `json:"caseId"`
	TransitionID string                 `json:"transitionId"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Status       string                 `json:"status"`
	Priority     string                 `json:"priority"`
	CreatedAt    time.Time              `json:"createdAt"`
	OfferedAt    *time.Time             `json:"offeredAt,omitempty"`
	AllocatedAt  *time.Time             `json:"allocatedAt,omitempty"`
	StartedAt    *time.Time             `json:"startedAt,omitempty"`
	CompletedAt  *time.Time             `json:"completedAt,omitempty"`
	DueDate      *time.Time             `json:"dueDate,omitempty"`
	AllocatedTo  string                 `json:"allocatedTo,omitempty"`
	OfferedTo    []string               `json:"offeredTo,omitempty"`
	Data         map[string]interface{} `json:"data"`
	Metadata     map[string]interface{} `json:"metadata"`
	BindingIndex int                    `json:"bindingIndex"`
	Duration     float64                `json:"duration"`   // Duration in seconds
	WaitTime     float64                `json:"waitTime"`   // Wait time in seconds
	IsOverdue    bool                   `json:"isOverdue"`
}

type WorkItemListResponse struct {
	WorkItems []WorkItemResponse `json:"workItems"`
	Total     int                `json:"total"`
	Offset    int                `json:"offset"`
	Limit     int                `json:"limit"`
}

// Helper functions

func (h *WorkItemHandlers) workItemToResponse(workItem *models.WorkItem) WorkItemResponse {
	return WorkItemResponse{
		ID:           workItem.ID,
		CaseID:       workItem.CaseID,
		TransitionID: workItem.TransitionID,
		Name:         workItem.Name,
		Description:  workItem.Description,
		Status:       string(workItem.Status),
		Priority:     string(workItem.Priority),
		CreatedAt:    workItem.CreatedAt,
		OfferedAt:    workItem.OfferedAt,
		AllocatedAt:  workItem.AllocatedAt,
		StartedAt:    workItem.StartedAt,
		CompletedAt:  workItem.CompletedAt,
		DueDate:      workItem.DueDate,
		AllocatedTo:  workItem.AllocatedTo,
		OfferedTo:    workItem.OfferedTo,
		Data:         workItem.Data,
		Metadata:     workItem.Metadata,
		BindingIndex: workItem.BindingIndex,
		Duration:     workItem.GetDuration().Seconds(),
		WaitTime:     workItem.GetWaitTime().Seconds(),
		IsOverdue:    workItem.IsOverdue(),
	}
}

func (h *WorkItemHandlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *WorkItemHandlers) writeError(w http.ResponseWriter, status int, err string, message string) {
	h.writeJSON(w, status, ErrorResponse{
		Error:   err,
		Message: message,
	})
}

func (h *WorkItemHandlers) writeSuccess(w http.ResponseWriter, data interface{}, message string) {
	h.writeJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// API Handlers

// CreateWorkItem creates a new work item
func (h *WorkItemHandlers) CreateWorkItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	var request CreateWorkItemRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_json", "Failed to parse JSON: "+err.Error())
		return
	}

	// Validate required fields
	if request.ID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_field", "Work item ID is required")
		return
	}
	if request.CaseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_field", "Case ID is required")
		return
	}
	if request.TransitionID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_field", "Transition ID is required")
		return
	}
	if request.Name == "" {
		h.writeError(w, http.StatusBadRequest, "missing_field", "Work item name is required")
		return
	}

	// Create the work item
	workItem, err := h.workItemManager.CreateWorkItem(request.ID, request.CaseID, request.TransitionID, request.Name, request.Description, request.BindingIndex)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "creation_failed", "Failed to create work item: "+err.Error())
		return
	}

	h.writeSuccess(w, h.workItemToResponse(workItem), "Work item created successfully")
}

// GetWorkItem retrieves a work item by ID
func (h *WorkItemHandlers) GetWorkItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	workItemID := r.URL.Query().Get("id")
	if workItemID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Work item ID is required")
		return
	}

	workItem, err := h.workItemManager.GetWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "workitem_not_found", err.Error())
		return
	}

	h.writeSuccess(w, h.workItemToResponse(workItem), "")
}

// UpdateWorkItem updates work item data and metadata
func (h *WorkItemHandlers) UpdateWorkItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only PUT method is allowed")
		return
	}

	workItemID := r.URL.Query().Get("id")
	if workItemID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Work item ID is required")
		return
	}

	var request UpdateWorkItemRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_json", "Failed to parse JSON: "+err.Error())
		return
	}

	err := h.workItemManager.UpdateWorkItem(workItemID, request.Data, request.Metadata)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "update_failed", err.Error())
		return
	}

	// Get updated work item
	workItem, err := h.workItemManager.GetWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.workItemToResponse(workItem), "Work item updated successfully")
}

// SetPriority sets the priority of a work item
func (h *WorkItemHandlers) SetPriority(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only PUT method is allowed")
		return
	}

	workItemID := r.URL.Query().Get("id")
	if workItemID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Work item ID is required")
		return
	}

	var request SetPriorityRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_json", "Failed to parse JSON: "+err.Error())
		return
	}

	priority := models.WorkItemPriority(request.Priority)
	if priority != models.WorkItemPriorityLow && priority != models.WorkItemPriorityNormal && 
	   priority != models.WorkItemPriorityHigh && priority != models.WorkItemPriorityUrgent {
		h.writeError(w, http.StatusBadRequest, "invalid_priority", "Invalid priority value")
		return
	}

	err := h.workItemManager.SetPriority(workItemID, priority)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "set_priority_failed", err.Error())
		return
	}

	// Get updated work item
	workItem, err := h.workItemManager.GetWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.workItemToResponse(workItem), "Work item priority updated successfully")
}

// SetDueDate sets the due date of a work item
func (h *WorkItemHandlers) SetDueDate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only PUT method is allowed")
		return
	}

	workItemID := r.URL.Query().Get("id")
	if workItemID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Work item ID is required")
		return
	}

	var request SetDueDateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_json", "Failed to parse JSON: "+err.Error())
		return
	}

	err := h.workItemManager.SetDueDate(workItemID, request.DueDate)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "set_due_date_failed", err.Error())
		return
	}

	// Get updated work item
	workItem, err := h.workItemManager.GetWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.workItemToResponse(workItem), "Work item due date updated successfully")
}

// OfferWorkItem offers a work item to specific users
func (h *WorkItemHandlers) OfferWorkItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	workItemID := r.URL.Query().Get("id")
	if workItemID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Work item ID is required")
		return
	}

	var request OfferWorkItemRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_json", "Failed to parse JSON: "+err.Error())
		return
	}

	if len(request.UserIDs) == 0 {
		h.writeError(w, http.StatusBadRequest, "missing_field", "At least one user ID is required")
		return
	}

	err := h.workItemManager.OfferWorkItem(workItemID, request.UserIDs)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "offer_failed", err.Error())
		return
	}

	// Get updated work item
	workItem, err := h.workItemManager.GetWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.workItemToResponse(workItem), "Work item offered successfully")
}

// AllocateWorkItem allocates a work item to a specific user
func (h *WorkItemHandlers) AllocateWorkItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	workItemID := r.URL.Query().Get("id")
	if workItemID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Work item ID is required")
		return
	}

	var request AllocateWorkItemRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_json", "Failed to parse JSON: "+err.Error())
		return
	}

	if request.UserID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_field", "User ID is required")
		return
	}

	err := h.workItemManager.AllocateWorkItem(workItemID, request.UserID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "allocate_failed", err.Error())
		return
	}

	// Get updated work item
	workItem, err := h.workItemManager.GetWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.workItemToResponse(workItem), "Work item allocated successfully")
}

// StartWorkItem starts a work item execution
func (h *WorkItemHandlers) StartWorkItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	workItemID := r.URL.Query().Get("id")
	if workItemID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Work item ID is required")
		return
	}

	err := h.workItemManager.StartWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "start_failed", err.Error())
		return
	}

	// Get updated work item
	workItem, err := h.workItemManager.GetWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.workItemToResponse(workItem), "Work item started successfully")
}

// CompleteWorkItem completes a work item
func (h *WorkItemHandlers) CompleteWorkItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	workItemID := r.URL.Query().Get("id")
	if workItemID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Work item ID is required")
		return
	}

	err := h.workItemManager.CompleteWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "complete_failed", err.Error())
		return
	}

	// Get updated work item
	workItem, err := h.workItemManager.GetWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.workItemToResponse(workItem), "Work item completed successfully")
}

// FailWorkItem marks a work item as failed
func (h *WorkItemHandlers) FailWorkItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	workItemID := r.URL.Query().Get("id")
	if workItemID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Work item ID is required")
		return
	}

	err := h.workItemManager.FailWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "fail_failed", err.Error())
		return
	}

	// Get updated work item
	workItem, err := h.workItemManager.GetWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.workItemToResponse(workItem), "Work item marked as failed")
}

// CancelWorkItem cancels a work item
func (h *WorkItemHandlers) CancelWorkItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	workItemID := r.URL.Query().Get("id")
	if workItemID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Work item ID is required")
		return
	}

	err := h.workItemManager.CancelWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "cancel_failed", err.Error())
		return
	}

	// Get updated work item
	workItem, err := h.workItemManager.GetWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", err.Error())
		return
	}

	h.writeSuccess(w, h.workItemToResponse(workItem), "Work item cancelled successfully")
}

// DeleteWorkItem deletes a work item
func (h *WorkItemHandlers) DeleteWorkItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only DELETE method is allowed")
		return
	}

	workItemID := r.URL.Query().Get("id")
	if workItemID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Work item ID is required")
		return
	}

	err := h.workItemManager.DeleteWorkItem(workItemID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "delete_failed", err.Error())
		return
	}

	h.writeSuccess(w, nil, "Work item deleted successfully")
}

// QueryWorkItems queries work items based on filter criteria
func (h *WorkItemHandlers) QueryWorkItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	var query models.WorkItemQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_json", "Failed to parse JSON: "+err.Error())
		return
	}

	workItems, err := h.workItemManager.QueryWorkItems(&query)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	var workItemResponses []WorkItemResponse
	for _, workItem := range workItems {
		workItemResponses = append(workItemResponses, h.workItemToResponse(workItem))
	}

	response := WorkItemListResponse{
		WorkItems: workItemResponses,
		Total:     len(workItemResponses),
		Offset:    query.Offset,
		Limit:     query.Limit,
	}

	h.writeSuccess(w, response, "")
}

// GetWorkItemsByCase returns all work items for a specific case
func (h *WorkItemHandlers) GetWorkItemsByCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	caseID := r.URL.Query().Get("caseId")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}

	workItems, err := h.workItemManager.GetWorkItemsByCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	var workItemResponses []WorkItemResponse
	for _, workItem := range workItems {
		workItemResponses = append(workItemResponses, h.workItemToResponse(workItem))
	}

	h.writeSuccess(w, workItemResponses, "")
}

// GetWorkItemsByUser returns all work items for a specific user
func (h *WorkItemHandlers) GetWorkItemsByUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	userID := r.URL.Query().Get("userId")
	if userID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "User ID is required")
		return
	}

	workItems, err := h.workItemManager.GetWorkItemsByUser(userID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	var workItemResponses []WorkItemResponse
	for _, workItem := range workItems {
		workItemResponses = append(workItemResponses, h.workItemToResponse(workItem))
	}

	h.writeSuccess(w, workItemResponses, "")
}

// GetOverdueWorkItems returns all overdue work items
func (h *WorkItemHandlers) GetOverdueWorkItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	workItems, err := h.workItemManager.GetOverdueWorkItems()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	var workItemResponses []WorkItemResponse
	for _, workItem := range workItems {
		workItemResponses = append(workItemResponses, h.workItemToResponse(workItem))
	}

	h.writeSuccess(w, workItemResponses, "")
}

// GetWorkItemStatistics returns statistics about work items
func (h *WorkItemHandlers) GetWorkItemStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	stats := h.workItemManager.GetWorkItemStatistics()
	h.writeSuccess(w, stats, "")
}

// CreateWorkItemsForCase creates work items for all manual transitions in a case
func (h *WorkItemHandlers) CreateWorkItemsForCase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	caseID := r.URL.Query().Get("caseId")
	if caseID == "" {
		h.writeError(w, http.StatusBadRequest, "missing_parameter", "Case ID is required")
		return
	}

	workItems, err := h.workItemManager.CreateWorkItemsForCase(caseID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "creation_failed", err.Error())
		return
	}

	var workItemResponses []WorkItemResponse
	for _, workItem := range workItems {
		workItemResponses = append(workItemResponses, h.workItemToResponse(workItem))
	}

	h.writeSuccess(w, workItemResponses, "Work items created successfully")
}

