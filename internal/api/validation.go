package api

import (
	"net/http"
	"strings"
)

// ValidationViolation represents a failed validation rule.
type ValidationViolation struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// TransitionDiagnostic provides per-transition enablement info.
type TransitionDiagnostic struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Enabled bool     `json:"enabled"`
	Reasons []string `json:"reasons,omitempty"`
	Kind    string   `json:"kind"`
	Guard   string   `json:"guard"`
}

// ValidateCPN validates a CPN definition and current marking; GET /api/cpn/validate?id=... .
// Treats empty / whitespace guard as true.
func (s *Server) ValidateCPN(w http.ResponseWriter, r *http.Request) {
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

	violations := []ValidationViolation{}

	// Duplicate place names
	nameCount := map[string]int{}
	for _, p := range cpn.Places {
		nameCount[p.Name]++
	}
	for n, c := range nameCount {
		if c > 1 {
			violations = append(violations, ValidationViolation{Code: "duplicate_place_name", Message: "Duplicate place name detected", Context: map[string]interface{}{"name": n, "count": c}})
		}
	}

	// Missing color sets
	for _, p := range cpn.Places {
		if p.ColorSet == nil {
			violations = append(violations, ValidationViolation{Code: "missing_color_set", Message: "Place has no color set", Context: map[string]interface{}{"placeId": p.ID, "placeName": p.Name}})
		}
	}

	// Initial marking referencing unknown places or bad tokens
	for placeName, multi := range cpn.InitialMarking {
		place := cpn.GetPlaceByName(placeName)
		if place == nil {
			violations = append(violations, ValidationViolation{Code: "initial_marking_unknown_place", Message: "Initial marking references unknown place", Context: map[string]interface{}{"placeName": placeName}})
			continue
		}
		for _, tok := range multi {
			if place.ColorSet != nil && !place.ColorSet.IsMember(tok.Value) {
				violations = append(violations, ValidationViolation{Code: "token_color_mismatch", Message: "Token value not member of place color set", Context: map[string]interface{}{"placeName": placeName, "value": tok.Value, "colorSet": place.ColorSet.Name()}})
			}
		}
	}

	diagnostics := []TransitionDiagnostic{}
	enabledTransitions, _, _ := s.engine.GetEnabledTransitions(cpn, marking)
	enabledSet := map[string]bool{}
	for _, t := range enabledTransitions {
		enabledSet[t.ID] = true
	}
	for _, t := range cpn.Transitions {
		diag := TransitionDiagnostic{ID: t.ID, Name: t.Name, Enabled: enabledSet[t.ID], Kind: string(t.Kind), Guard: t.GuardExpression}
		if !diag.Enabled {
			inputArcs := cpn.GetInputArcs(t.ID)
			missingToken := false
			for _, arc := range inputArcs {
				place := cpn.GetPlace(arc.SourceID)
				if place == nil {
					diag.Reasons = append(diag.Reasons, "missing_input_place")
					continue
				}
				ms := marking.Places[place.Name]
				if ms == nil || ms.Size() == 0 {
					diag.Reasons = append(diag.Reasons, "no_tokens_in_"+place.Name)
					missingToken = true
				}
			}
			g := strings.TrimSpace(t.GuardExpression)
			if !missingToken && g != "" {
				diag.Reasons = append(diag.Reasons, "guard_or_binding_not_satisfied")
			}
		}
		diagnostics = append(diagnostics, diag)
	}

	if len(enabledSet) == 0 {
		violations = append(violations, ValidationViolation{Code: "deadlock", Message: "No transitions are currently enabled"})
	}

	valid := len(violations) == 0
	result := map[string]interface{}{
		"cpnId":       cpnID,
		"valid":       valid,
		"violations":  violations,
		"transitions": diagnostics,
	}
	s.writeSuccess(w, result, "Validation completed")
}
