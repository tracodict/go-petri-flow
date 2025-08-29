package test

import (
	case_manager "go-petri-flow/internal/case"
	"go-petri-flow/internal/engine"
	"go-petri-flow/internal/models"
	"testing"
)

// TestHierarchyPropagateOnComplete verifies that a child CPN completion with propagateOnComplete=true
// produces deferred parent output tokens using outputMapping.
func TestHierarchyPropagateOnComplete(t *testing.T) {
	eng := engine.NewEngine()
	defer eng.Close()

	mgr := case_manager.NewManager(eng)

	// Color set
	intCS := models.NewIntegerColorSet("INT", false)

	// Child CPN: c_in ->(x)-> t_child (y = x * 2) ->(y)-> c_out
	child := models.NewCPN("child-prop-cpn", "ChildProp", "Child for propagation test")
	cIn := models.NewPlace("c_in", "CIn", intCS)
	cOut := models.NewPlace("c_out", "COut", intCS)
	child.AddPlace(cIn)
	child.AddPlace(cOut)
	tChild := models.NewTransition("t_child", "Child")
	tChild.SetKind(models.TransitionKindAuto)
	tChild.SetAction("y = x * 2")
	child.AddTransition(tChild)
	child.AddArc(models.NewInputArc("ac_in", "c_in", "t_child", "x"))
	child.AddArc(models.NewOutputArc("ac_out", "t_child", "c_out", "y"))
	// Initial marking: token value 5 so result expected 10
	child.AddInitialToken("c_in", models.NewToken(5, 0))
	child.SetEndPlaces([]string{"c_out"}) // name accepted

	// Parent CPN: p_start ->(a)-> t_call ->(b)-> p_wait
	parent := models.NewCPN("parent-prop-cpn", "ParentProp", "Parent for propagation test")
	pStart := models.NewPlace("p_start", "Start", intCS)
	pWait := models.NewPlace("p_wait", "Wait", intCS)
	parent.AddPlace(pStart)
	parent.AddPlace(pWait)
	tCall := models.NewTransition("t_call", "CallChild")
	tCall.SetKind(models.TransitionKindManual)
	parent.AddTransition(tCall)
	parent.AddArc(models.NewInputArc("ap_in", "p_start", "t_call", "a"))
	parent.AddArc(models.NewOutputArc("ap_out", "t_call", "p_wait", "b"))
	parent.AddInitialToken("p_start", models.NewToken(5, 0))
	parent.SetEndPlaces([]string{"p_wait"})

	// SubWorkflow link (propagateOnComplete = true, autoStart = true)
	parent.SubWorkflows = append(parent.SubWorkflows, &models.SubWorkflowLink{
		ID:                  "sw1",
		CPNID:               child.ID,
		CallTransitionID:    tCall.ID,
		AutoStart:           true,
		PropagateOnComplete: true,
		InputMapping:        map[string]string{"x": "a"},
		OutputMapping:       map[string]string{"y": "b"},
	})

	// Register CPNs
	mgr.RegisterCPN(child)
	mgr.RegisterCPN(parent)

	// Create and start parent case
	if _, err := mgr.CreateCase("parent-prop-case", parent.ID, "Parent Case", "", nil); err != nil {
		t.Fatalf("failed to create parent case: %v", err)
	}
	if err := mgr.StartCase("parent-prop-case"); err != nil {
		t.Fatalf("failed to start parent case: %v", err)
	}

	// Fire call transition (binding index 0)
	if err := mgr.FireTransition("parent-prop-case", tCall.ID, 0); err != nil {
		t.Fatalf("failed to fire hierarchical call transition: %v", err)
	}

	// After firing, child autoStart should have completed (single auto transition), triggering propagation.
	caseState, err := mgr.GetCase("parent-prop-case")
	if err != nil {
		t.Fatalf("failed to get parent case: %v", err)
	}

	// Expect token in p_wait with value 10 (5 * 2)
	tokens := caseState.Marking.GetTokens("p_wait")
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token in p_wait after propagation, got %d", len(tokens))
	}
	if tokens[0].Value != 10 {
		t.Fatalf("expected propagated token value 10, got %v", tokens[0].Value)
	}
}
