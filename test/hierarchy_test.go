package test

import (
	"encoding/json"
	"testing"

	cm "go-petri-flow/internal/case"
	"go-petri-flow/internal/engine"
	"go-petri-flow/internal/models"
)

// TestHierarchyLoad ensures parser accepts subWorkflows and basic structures wire
func TestHierarchyLoad(t *testing.T) {
	parser := models.NewCPNParser()
	childJSON := []byte(`{
        "id":"child-cpn-1","name":"ChildFlow","description":"","colorSets":["colset INT = int;"],
        "places":[{"id":"c_in","name":"CIn","colorSet":"INT"},{"id":"c_out","name":"COut","colorSet":"INT"}],
        "transitions":[{"id":"t_child","name":"Child","kind":"Auto","actionExpression":"y = x * 2"}],
        "arcs":[
          {"id":"ac_in","sourceId":"c_in","targetId":"t_child","expression":"x","direction":"IN"},
          {"id":"ac_out","sourceId":"t_child","targetId":"c_out","expression":"y","direction":"OUT"}
        ],
	"initialMarking":{"c_in":[{"value":5,"timestamp":0}]},
	"endPlaces":["c_out"]
    }`)
	childCPN, err := parser.ParseCPNFromJSON(childJSON)
	if err != nil {
		t.Fatalf("child parse err: %v", err)
	}

	parentJSON := []byte(`{
        "id":"parent-cpn-1","name":"ParentFlow","description":"","colorSets":["colset INT = int;"],
        "places":[{"id":"p_start","name":"Start","colorSet":"INT"},{"id":"p_done","name":"Done","colorSet":"INT"}],
        "transitions":[{"id":"t_call","name":"Call","kind":"Manual"}],
        "arcs":[{"id":"a_in","sourceId":"p_start","targetId":"t_call","expression":"x","direction":"IN"},{"id":"a_out","sourceId":"t_call","targetId":"p_done","expression":"y","direction":"OUT"}],
        "initialMarking":{"p_start":[{"value":3,"timestamp":0}]},
	"endPlaces":["p_done"],
        "subWorkflows":[{"id":"sw1","cpnId":"child-cpn-1","callTransitionId":"t_call","autoStart":true,"propagateOnComplete":true, "inputMapping":{"x":"x"},"outputMapping":{"y":"y"}}]
    }`)
	parentCPN, err := parser.ParseCPNFromJSON(parentJSON)
	if err != nil {
		t.Fatalf("parent parse err: %v", err)
	}

	if len(parentCPN.SubWorkflows) != 1 {
		t.Fatalf("expected 1 subWorkflow")
	}
	if parentCPN.GetSubWorkflowByTransition("t_call") == nil {
		t.Fatalf("lookup subWorkflow failed")
	}

	// Roundtrip
	data, err := parser.CPNToJSON(parentCPN)
	if err != nil {
		t.Fatalf("roundtrip err: %v", err)
	}
	var j map[string]interface{}
	if err := json.Unmarshal(data, &j); err != nil {
		t.Fatalf("unmarshal back err: %v", err)
	}
	if _, ok := j["subWorkflows"]; !ok {
		t.Fatalf("subWorkflows missing in json output")
	}

	// Basic manager wiring (no actual hierarchical output yet)
	eng := engine.NewEngine()
	mgr := cm.NewManager(eng)
	mgr.RegisterCPN(childCPN)
	mgr.RegisterCPN(parentCPN)
	_, err = mgr.CreateCase("parent-case-1", parentCPN.ID, "Parent", "", nil)
	if err != nil {
		t.Fatalf("create parent case err: %v", err)
	}
	if err := mgr.StartCase("parent-case-1"); err != nil {
		t.Fatalf("start parent err: %v", err)
	}
	// Fire call transition (binding 0)
	if err := mgr.FireTransition("parent-case-1", "t_call", 0); err != nil {
		t.Fatalf("fire hierarchical call err: %v", err)
	}
	// Verify child created
	pc, _ := mgr.GetCase("parent-case-1")
	if len(pc.Children) != 1 {
		t.Fatalf("expected 1 child case, got %d", len(pc.Children))
	}
}
