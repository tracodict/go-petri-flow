package models

// SubWorkflowLink represents a hierarchical subworkflow (substitution transition) configuration
// following CPN Tools style semantics.
type SubWorkflowLink struct {
	ID                  string            `json:"id"`                      // Unique ID of the link within parent CPN
	CPNID               string            `json:"cpnId"`                   // Target child CPN id
	CallTransitionID    string            `json:"callTransitionId"`        // Transition in parent acting as call
	AutoStart           bool              `json:"autoStart"`               // Start child automatically upon creation
	PropagateOnComplete bool              `json:"propagateOnComplete"`     // If true, parent outputs deferred until child completion
	InputMapping        map[string]string `json:"inputMapping,omitempty"`  // parentVar -> childVar
	OutputMapping       map[string]string `json:"outputMapping,omitempty"` // childVar -> parentVar
}

// Clone creates a deep copy of the sub workflow link
func (sw *SubWorkflowLink) Clone() *SubWorkflowLink {
	if sw == nil {
		return nil
	}
	inMap := make(map[string]string, len(sw.InputMapping))
	for k, v := range sw.InputMapping {
		inMap[k] = v
	}
	outMap := make(map[string]string, len(sw.OutputMapping))
	for k, v := range sw.OutputMapping {
		outMap[k] = v
	}
	return &SubWorkflowLink{
		ID:                  sw.ID,
		CPNID:               sw.CPNID,
		CallTransitionID:    sw.CallTransitionID,
		AutoStart:           sw.AutoStart,
		PropagateOnComplete: sw.PropagateOnComplete,
		InputMapping:        inMap,
		OutputMapping:       outMap,
	}
}
