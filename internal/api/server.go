package api

import (
	"log"
	"net/http"
)

// SetupRoutes sets up the HTTP routes for the API server
func (s *Server) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Add CORS middleware to all routes

	// CPN Management
	mux.HandleFunc("/api/cpn/load", s.corsMiddleware(s.LoadCPN))
	mux.HandleFunc("/api/cpn/list", s.corsMiddleware(s.ListCPNs))
	mux.HandleFunc("/api/cpn/get", s.corsMiddleware(s.GetCPN))
	mux.HandleFunc("/api/cpn/delete", s.corsMiddleware(s.DeleteCPN))
	mux.HandleFunc("/api/cpn/reset", s.corsMiddleware(s.ResetCPN))
	mux.HandleFunc("/api/cpn/validate", s.corsMiddleware(s.ValidateCPN))

	// Marking
	mux.HandleFunc("/api/marking/get", s.corsMiddleware(s.GetMarking))

	// Transitions
	mux.HandleFunc("/api/transitions/list", s.corsMiddleware(s.GetTransitions))
	mux.HandleFunc("/api/transitions/enabled", s.corsMiddleware(s.GetEnabledTransitions))
	mux.HandleFunc("/api/transitions/fire", s.corsMiddleware(s.FireTransition))

	// Simulation
	mux.HandleFunc("/api/simulation/step", s.corsMiddleware(s.SimulateStep))
	mux.HandleFunc("/api/simulation/steps", s.corsMiddleware(s.SimulateSteps))

	// Case Management
	mux.HandleFunc("/api/cases/create", s.corsMiddleware(s.caseHandlers.CreateCase))
	mux.HandleFunc("/api/cases/get", s.corsMiddleware(s.caseHandlers.GetCase))
	mux.HandleFunc("/api/cases/update", s.corsMiddleware(s.caseHandlers.UpdateCase))
	mux.HandleFunc("/api/cases/delete", s.corsMiddleware(s.caseHandlers.DeleteCase))
	mux.HandleFunc("/api/cases/start", s.corsMiddleware(s.caseHandlers.StartCase))
	mux.HandleFunc("/api/cases/suspend", s.corsMiddleware(s.caseHandlers.SuspendCase))
	mux.HandleFunc("/api/cases/resume", s.corsMiddleware(s.caseHandlers.ResumeCase))
	mux.HandleFunc("/api/cases/abort", s.corsMiddleware(s.caseHandlers.AbortCase))
	mux.HandleFunc("/api/cases/execute", s.corsMiddleware(s.caseHandlers.ExecuteStep))
	mux.HandleFunc("/api/cases/executeall", s.corsMiddleware(s.caseHandlers.ExecuteAll))
	mux.HandleFunc("/api/cases/fire", s.corsMiddleware(s.caseHandlers.FireTransition))
	mux.HandleFunc("/api/cases/marking", s.corsMiddleware(s.caseHandlers.GetCaseMarking))
	mux.HandleFunc("/api/cases/transitions", s.corsMiddleware(s.caseHandlers.GetCaseTransitions))
	mux.HandleFunc("/api/cases/query", s.corsMiddleware(s.caseHandlers.QueryCases))
	mux.HandleFunc("/api/cases/statistics", s.corsMiddleware(s.caseHandlers.GetCaseStatistics))

	// Work Item Management
	mux.HandleFunc("/api/workitems/create", s.corsMiddleware(s.workItemHandlers.CreateWorkItem))
	mux.HandleFunc("/api/workitems/get", s.corsMiddleware(s.workItemHandlers.GetWorkItem))
	mux.HandleFunc("/api/workitems/update", s.corsMiddleware(s.workItemHandlers.UpdateWorkItem))
	mux.HandleFunc("/api/workitems/delete", s.corsMiddleware(s.workItemHandlers.DeleteWorkItem))
	mux.HandleFunc("/api/workitems/priority", s.corsMiddleware(s.workItemHandlers.SetPriority))
	mux.HandleFunc("/api/workitems/duedate", s.corsMiddleware(s.workItemHandlers.SetDueDate))
	mux.HandleFunc("/api/workitems/offer", s.corsMiddleware(s.workItemHandlers.OfferWorkItem))
	mux.HandleFunc("/api/workitems/allocate", s.corsMiddleware(s.workItemHandlers.AllocateWorkItem))
	mux.HandleFunc("/api/workitems/start", s.corsMiddleware(s.workItemHandlers.StartWorkItem))
	mux.HandleFunc("/api/workitems/complete", s.corsMiddleware(s.workItemHandlers.CompleteWorkItem))
	mux.HandleFunc("/api/workitems/fail", s.corsMiddleware(s.workItemHandlers.FailWorkItem))
	mux.HandleFunc("/api/workitems/cancel", s.corsMiddleware(s.workItemHandlers.CancelWorkItem))
	mux.HandleFunc("/api/workitems/query", s.corsMiddleware(s.workItemHandlers.QueryWorkItems))
	mux.HandleFunc("/api/workitems/bycase", s.corsMiddleware(s.workItemHandlers.GetWorkItemsByCase))
	mux.HandleFunc("/api/workitems/byuser", s.corsMiddleware(s.workItemHandlers.GetWorkItemsByUser))
	mux.HandleFunc("/api/workitems/overdue", s.corsMiddleware(s.workItemHandlers.GetOverdueWorkItems))
	mux.HandleFunc("/api/workitems/statistics", s.corsMiddleware(s.workItemHandlers.GetWorkItemStatistics))
	mux.HandleFunc("/api/workitems/createforcase", s.corsMiddleware(s.workItemHandlers.CreateWorkItemsForCase))

	// Health check endpoint
	mux.HandleFunc("/api/health", s.corsMiddleware(s.HealthCheck))

	// API documentation endpoint
	mux.HandleFunc("/api/docs", s.corsMiddleware(s.APIDocs))

	return mux
}

// corsMiddleware adds CORS headers to allow cross-origin requests
func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler
		next(w, r)
	}
}

// HealthCheck returns the health status of the API
func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	status := map[string]interface{}{
		"status":  "healthy",
		"service": "go-petri-flow",
		"version": "1.0.0",
		"cpns":    len(s.cpns),
		"engine":  "gopher-lua",
	}

	s.writeSuccess(w, status, "Service is healthy")
}

// APIDocs returns API documentation
func (s *Server) APIDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	docs := map[string]interface{}{
		"title":       "Go Petri Flow API",
		"version":     "1.0.0",
		"description": "REST API for Colored Petri Net simulation using gopher-lua",
		"endpoints": map[string]interface{}{
			"CPN Management": map[string]interface{}{
				"POST /api/cpn/load":     "Load a CPN from JSON definition",
				"GET /api/cpn/list":      "List all loaded CPNs",
				"GET /api/cpn/get":       "Get CPN details by ID",
				"DELETE /api/cpn/delete": "Delete a CPN by ID",
				"POST /api/cpn/reset":    "Reset CPN to initial marking",
				"GET /api/cpn/validate":  "Validate a CPN and return rule violations and transition diagnostics",
			},
			"Marking": map[string]interface{}{
				"GET /api/marking/get": "Get current marking of a CPN",
			},
			"Transitions": map[string]interface{}{
				"GET /api/transitions/list":  "List transitions and their status",
				"POST /api/transitions/fire": "Manually fire a transition",
			},
			"Simulation": map[string]interface{}{
				"POST /api/simulation/step":  "Perform one simulation step",
				"POST /api/simulation/steps": "Perform multiple simulation steps",
			},
			"Utility": map[string]interface{}{
				"GET /api/health": "Health check",
				"GET /api/docs":   "API documentation",
			},
		},
		"examples": map[string]interface{}{
			"load_cpn": map[string]interface{}{
				"method": "POST",
				"url":    "/api/cpn/load",
				"body": map[string]interface{}{
					"id":          "simple-cpn",
					"name":        "Simple CPN",
					"description": "A simple test CPN",
					"colorSets":   []string{"colset INT = int;"},
					"places": []map[string]interface{}{
						{"id": "p1", "name": "Place1", "colorSet": "INT"},
						{"id": "p2", "name": "Place2", "colorSet": "INT"},
					},
					"transitions": []map[string]interface{}{
						{"id": "t1", "name": "Transition1", "kind": "Auto"},
					},
					"arcs": []map[string]interface{}{
						{"id": "a1", "sourceId": "p1", "targetId": "t1", "expression": "x", "direction": "IN"},
						{"id": "a2", "sourceId": "t1", "targetId": "p2", "expression": "x + 1", "direction": "OUT"},
					},
					"initialMarking": map[string]interface{}{
						"Place1": []map[string]interface{}{
							{"value": 1, "timestamp": 0},
						},
					},
				},
			},
			"fire_transition": map[string]interface{}{
				"method": "POST",
				"url":    "/api/transitions/fire",
				"body": map[string]interface{}{
					"cpnId":        "simple-cpn",
					"transitionId": "t1",
					"bindingIndex": 0,
				},
			},
		},
	}

	s.writeSuccess(w, docs, "")
}

// StartServer starts the HTTP server
func (s *Server) StartServer(port string) error {
	mux := s.SetupRoutes()

	log.Printf("Starting Go Petri Flow API server on port %s", port)
	log.Printf("API documentation available at: http://localhost:%s/api/docs", port)
	log.Printf("Health check available at: http://localhost:%s/api/health", port)

	return http.ListenAndServe("0.0.0.0:"+port, mux)
}
