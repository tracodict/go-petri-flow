package engine

import (
	"fmt"
	"sort"

	"go-petri-flow/internal/expression"
	"go-petri-flow/internal/models"
)

// Engine represents the CPN simulation engine
type Engine struct {
	evaluator *expression.Evaluator
}

// NewEngine creates a new CPN simulation engine
func NewEngine() *Engine {
	return &Engine{
		evaluator: expression.NewEvaluator(),
	}
}

// Close closes the engine and releases resources
func (e *Engine) Close() {
	if e.evaluator != nil {
		e.evaluator.Close()
	}
}

// TokenBinding represents a binding of variables to tokens
type TokenBinding map[string]*models.Token

// IsEnabled checks if a transition is enabled given the current marking
func (e *Engine) IsEnabled(cpn *models.CPN, transition *models.Transition, marking *models.Marking) (bool, []TokenBinding, error) {
	// Get input arcs for the transition
	inputArcs := cpn.GetInputArcs(transition.ID)
	if len(inputArcs) == 0 {
		// Transition with no input arcs is always enabled if guard passes
		guardPassed, err := e.checkGuard(transition, TokenBinding{}, marking)
		if err != nil {
			return false, nil, err
		}
		if guardPassed {
			return true, []TokenBinding{{}}, nil
		}
		return false, nil, nil
	}

	// Find all possible token bindings
	bindings, err := e.findTokenBindings(cpn, transition, inputArcs, marking)
	if err != nil {
		return false, nil, fmt.Errorf("failed to find token bindings: %v", err)
	}

	if len(bindings) == 0 {
		return false, nil, nil
	}

	// Filter bindings that satisfy the guard
	var validBindings []TokenBinding
	for _, binding := range bindings {
		guardPassed, err := e.checkGuard(transition, binding, marking)
		if err != nil {
			return false, nil, fmt.Errorf("failed to check guard: %v", err)
		}
		if guardPassed {
			validBindings = append(validBindings, binding)
		}
	}

	return len(validBindings) > 0, validBindings, nil
}

// FireTransition fires a transition with the given binding
func (e *Engine) FireTransition(cpn *models.CPN, transition *models.Transition, binding TokenBinding, marking *models.Marking) error {
	// Verify the transition is enabled with this binding
	enabled, _, err := e.IsEnabled(cpn, transition, marking)
	if err != nil {
		return fmt.Errorf("failed to check if transition is enabled: %v", err)
	}
	if !enabled {
		return fmt.Errorf("transition %s is not enabled", transition.Name)
	}

	// Create evaluation context
	context := e.createEvaluationContext(binding, marking)

	// Process input arcs (consume tokens)
	inputArcs := cpn.GetInputArcs(transition.ID)
	for _, arc := range inputArcs {
		count := arc.Multiplicity
		if count <= 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			if err := e.processInputArc(cpn, arc, context, marking); err != nil {
				return fmt.Errorf("failed to process input arc %s (instance %d/%d): %v", arc.ID, i+1, count, err)
			}
		}
	}

	// Advance global clock if transition has delay
	if transition.TransitionDelay > 0 {
		marking.AdvanceGlobalClock(marking.GlobalClock + transition.TransitionDelay)
	}

	// Execute transition action (side-effect expression) if present and capture mutated variables
	if transition.HasAction() {
		if err := e.evaluator.EvaluateAction(transition.ActionExpression, context); err != nil {
			return fmt.Errorf("failed to execute action for transition %s: %v", transition.Name, err)
		}
		// After action, pull back Lua globals for each bound variable
		for varName, tk := range context.TokenBindings {
			if tk == nil { continue }
			if goVal := e.evaluator.GetGlobalValue(varName); goVal != nil { tk.Value = goVal }
		}
	}

	// Process output arcs (produce tokens)
	outputArcs := cpn.GetOutputArcs(transition.ID)
	for _, arc := range outputArcs {
		count := arc.Multiplicity
		if count <= 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			if err := e.processOutputArc(cpn, arc, context, marking); err != nil {
				return fmt.Errorf("failed to process output arc %s (instance %d/%d): %v", arc.ID, i+1, count, err)
			}
		}
	}

	// Increment step counter for each successful transition firing
	marking.StepCounter++

	return nil
}

// FireTransitionWithData fires a transition injecting external formData variables into the evaluation context
func (e *Engine) FireTransitionWithData(cpn *models.CPN, transition *models.Transition, binding TokenBinding, marking *models.Marking, formData map[string]interface{}) error {
	// Basic path if no extra data
	if len(formData) == 0 {
		return e.FireTransition(cpn, transition, binding, marking)
	}

	// Verify enabled
	enabled, _, err := e.IsEnabled(cpn, transition, marking)
	if err != nil {
		return fmt.Errorf("failed to check if transition is enabled: %v", err)
	}
	if !enabled {
		return fmt.Errorf("transition %s is not enabled", transition.Name)
	}

	// Create evaluation context with existing binding
	context := e.createEvaluationContext(binding, marking)

	// Inject form data as variable bindings
	if len(formData) > 0 {
		for k, v := range formData {
			context.SetValue(k, v)
		}
	}

	// Process input arcs
	inputArcs := cpn.GetInputArcs(transition.ID)
	for _, arc := range inputArcs {
		count := arc.Multiplicity
		if count <= 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			if err := e.processInputArc(cpn, arc, context, marking); err != nil {
				return fmt.Errorf("failed to process input arc %s (instance %d/%d): %v", arc.ID, i+1, count, err)
			}
		}
	}

	if transition.TransitionDelay > 0 {
		marking.AdvanceGlobalClock(marking.GlobalClock + transition.TransitionDelay)
	}

	if transition.HasAction() {
		if err := e.evaluator.EvaluateAction(transition.ActionExpression, context); err != nil {
			return fmt.Errorf("failed to execute action for transition %s: %v", transition.Name, err)
		}
		for varName, tk := range context.TokenBindings {
			if tk == nil { continue }
			if goVal := e.evaluator.GetGlobalValue(varName); goVal != nil { tk.Value = goVal }
		}
	}

	outputArcs := cpn.GetOutputArcs(transition.ID)
	for _, arc := range outputArcs {
		count := arc.Multiplicity
		if count <= 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			if err := e.processOutputArc(cpn, arc, context, marking); err != nil {
				return fmt.Errorf("failed to process output arc %s (instance %d/%d): %v", arc.ID, i+1, count, err)
			}
		}
	}

	marking.StepCounter++
	return nil
}

// AdvanceGlobalClock advances the global clock to the next earliest token timestamp
func (e *Engine) AdvanceGlobalClock(marking *models.Marking) {
	earliest := marking.GetEarliestTimestamp()
	if earliest > marking.GlobalClock {
		marking.AdvanceGlobalClock(earliest)
	}
}

// findTokenBindings finds all possible token bindings for a transition
func (e *Engine) findTokenBindings(cpn *models.CPN, transition *models.Transition, inputArcs []*models.Arc, marking *models.Marking) ([]TokenBinding, error) {
	if len(inputArcs) == 0 {
		return []TokenBinding{{}}, nil
	}

	// Start with the first arc and recursively build bindings
	return e.findBindingsRecursive(cpn, inputArcs, 0, TokenBinding{}, marking)
}

// findBindingsRecursive recursively finds token bindings for input arcs
func (e *Engine) findBindingsRecursive(cpn *models.CPN, arcs []*models.Arc, arcIndex int, currentBinding TokenBinding, marking *models.Marking) ([]TokenBinding, error) {
	if arcIndex >= len(arcs) {
		// Base case: all arcs processed
		return []TokenBinding{currentBinding}, nil
	}

	arc := arcs[arcIndex]
	place := cpn.GetPlace(arc.GetPlaceID())
	if place == nil {
		return nil, fmt.Errorf("place %s not found", arc.GetPlaceID())
	}

	// Get available tokens in the place
	availableTokens := marking.GetAvailableTokensAtTime(place.ID, marking.GlobalClock)
	if len(availableTokens) == 0 {
		return []TokenBinding{}, nil
	}

	var allBindings []TokenBinding

	// Try each available token
	for _, token := range availableTokens {
		// Create a new binding with this token
		newBinding := e.cloneBinding(currentBinding)

		// Evaluate the arc expression to see if this token matches
		context := e.createEvaluationContext(newBinding, marking)
		context.BindVariable("token", token) // Bind the current token for evaluation

		result, err := e.evaluator.EvaluateArcExpression(arc.Expression, context)
		if err != nil {
			continue // Skip tokens that cause evaluation errors
		}

		// Check if the result matches the token value
		if e.tokenMatches(token, result) {
			// Extract variable bindings from the expression
			if err := e.extractVariableBindings(arc.Expression, token, newBinding); err != nil {
				continue // Skip if variable extraction fails
			}

			// Recursively process remaining arcs
			subBindings, err := e.findBindingsRecursive(cpn, arcs, arcIndex+1, newBinding, marking)
			if err != nil {
				continue // Skip if recursive binding fails
			}

			allBindings = append(allBindings, subBindings...)
		}
	}

	return allBindings, nil
}

// extractVariableBindings extracts variable bindings from an arc expression
func (e *Engine) extractVariableBindings(expression string, token *models.Token, binding TokenBinding) error {
	// For simple expressions like "x", bind the variable to the token
	// For complex expressions, this would need more sophisticated parsing

	// Simple case: if expression is just a variable name
	if isSimpleVariable(expression) {
		binding[expression] = token
		return nil
	}

	// For more complex expressions, we would need to parse and match patterns
	// For now, we'll handle the most common cases

	return nil
}

// tokenMatches checks if a token matches the result of an arc expression
func (e *Engine) tokenMatches(token *models.Token, result interface{}) bool {
	// For simple variable expressions, we want to bind the variable to the token
	// So we return true to indicate a match
	return true
}

// isSimpleVariable checks if an expression is just a simple variable name
func isSimpleVariable(expression string) bool {
	// Simple heuristic: if it's a single word with no operators
	return len(expression) > 0 &&
		expression[0] >= 'a' && expression[0] <= 'z' &&
		!containsOperators(expression)
}

// containsOperators checks if an expression contains operators
func containsOperators(expression string) bool {
	operators := []string{"+", "-", "*", "/", "(", ")", ".", ",", " "}
	for _, op := range operators {
		if contains(expression, op) {
			return true
		}
	}
	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(len(substr) == 0 ||
			(len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}

// checkGuard evaluates the guard expression for a transition
func (e *Engine) checkGuard(transition *models.Transition, binding TokenBinding, marking *models.Marking) (bool, error) {
	if !transition.HasGuard() {
		return true, nil
	}

	context := e.createEvaluationContext(binding, marking)
	return e.evaluator.EvaluateGuard(transition.GuardExpression, context)
}

// processInputArc processes an input arc (consumes tokens)
func (e *Engine) processInputArc(cpn *models.CPN, arc *models.Arc, context *expression.EvaluationContext, marking *models.Marking) error {
	place := cpn.GetPlace(arc.GetPlaceID())
	if place == nil {
		return fmt.Errorf("place %s not found", arc.GetPlaceID())
	}

	// Evaluate the arc expression to determine which tokens to consume
	result, err := e.evaluator.EvaluateArcExpression(arc.Expression, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate input arc expression: %v", err)
	}

	// Remove the token from the place
	token := marking.RemoveTokenByValue(place.ID, result)
	if token == nil {
		return fmt.Errorf("no token with value %v found in place %s", result, place.Name)
	}

	return nil
}

// processOutputArc processes an output arc (produces tokens)
func (e *Engine) processOutputArc(cpn *models.CPN, arc *models.Arc, context *expression.EvaluationContext, marking *models.Marking) error {
	place := cpn.GetPlace(arc.GetPlaceID())
	if place == nil {
		return fmt.Errorf("place %s not found", arc.GetPlaceID())
	}

	// Evaluate the arc expression to determine what tokens to produce
	result, err := e.evaluator.EvaluateArcExpression(arc.Expression, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate output arc expression: %v", err)
	}

	// Handle delayed tokens (if result contains delay information)
	timestamp := marking.GlobalClock
	value := result

	// Check if result is a delayed token (table with value and delay)
	if resultMap, ok := result.(map[string]interface{}); ok {
		if delayVal, hasDelay := resultMap["delay"]; hasDelay {
			if delay, ok := delayVal.(int); ok {
				timestamp += delay
			}
		}
		if val, hasValue := resultMap["value"]; hasValue {
			value = val
		}
	}

	// Create and add the new token
	newToken := models.NewToken(value, timestamp)

	// Validate the token against the place's color set
	if err := place.ValidateToken(newToken); err != nil {
		return fmt.Errorf("invalid token for place %s: %v", place.Name, err)
	}

	marking.AddToken(place.ID, newToken)
	return nil
}

// createEvaluationContext creates an evaluation context from a token binding and marking
func (e *Engine) createEvaluationContext(binding TokenBinding, marking *models.Marking) *expression.EvaluationContext {
	context := expression.NewEvaluationContext()
	context.SetGlobalClock(marking.GlobalClock)

	// Add token bindings
	for varName, token := range binding {
		context.BindVariable(varName, token)
	}

	// Add place tokens for complex expressions
	for placeID := range marking.Places {
		tokens := marking.GetTokens(placeID)
		context.SetPlaceTokens(placeID, tokens)
	}

	return context
}

// cloneBinding creates a copy of a token binding
func (e *Engine) cloneBinding(binding TokenBinding) TokenBinding {
	clone := make(TokenBinding)
	for varName, token := range binding {
		clone[varName] = token
	}
	return clone
}

// GetEnabledTransitions returns all enabled transitions in the CPN
func (e *Engine) GetEnabledTransitions(cpn *models.CPN, marking *models.Marking) ([]*models.Transition, map[string][]TokenBinding, error) {
	var enabledTransitions []*models.Transition
	bindingsMap := make(map[string][]TokenBinding)

	for _, transition := range cpn.Transitions {
		enabled, bindings, err := e.IsEnabled(cpn, transition, marking)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to check if transition %s is enabled: %v", transition.Name, err)
		}

		if enabled {
			enabledTransitions = append(enabledTransitions, transition)
			bindingsMap[transition.ID] = bindings
		}
	}

	return enabledTransitions, bindingsMap, nil
}

// FireEnabledTransitions fires all enabled automatic transitions
func (e *Engine) FireEnabledTransitions(cpn *models.CPN, marking *models.Marking) (int, error) {
	firedCount := 0

	for {
		enabledTransitions, bindingsMap, err := e.GetEnabledTransitions(cpn, marking)
		if err != nil {
			return firedCount, fmt.Errorf("failed to get enabled transitions: %v", err)
		}

		// Filter only automatic transitions
		var automaticTransitions []*models.Transition
		for _, transition := range enabledTransitions {
			if transition.IsAuto() {
				automaticTransitions = append(automaticTransitions, transition)
			}
		}

		if len(automaticTransitions) == 0 {
			break // No more automatic transitions to fire
		}

		// Sort transitions by priority (for deterministic execution)
		sort.Slice(automaticTransitions, func(i, j int) bool {
			return automaticTransitions[i].ID < automaticTransitions[j].ID
		})

		// Fire the first enabled automatic transition
		transition := automaticTransitions[0]
		bindings := bindingsMap[transition.ID]

		if len(bindings) > 0 {
			// Use the first available binding
			if err := e.FireTransition(cpn, transition, bindings[0], marking); err != nil {
				return firedCount, fmt.Errorf("failed to fire transition %s: %v", transition.Name, err)
			}
			firedCount++
		}
	}

	return firedCount, nil
}

// SimulateStep performs one simulation step (fire all enabled automatic transitions)
func (e *Engine) SimulateStep(cpn *models.CPN, marking *models.Marking) (int, error) {
	// Advance global clock if needed (bring earliest future tokens into scope)
	e.AdvanceGlobalClock(marking)

	fired := 0
	// Capture snapshot of enabled automatic transitions at start of step (layer)
	enabled, bindingsMap, err := e.GetEnabledTransitions(cpn, marking)
	if err != nil {
		return 0, fmt.Errorf("failed to get enabled transitions: %v", err)
	}
	// Deterministic ordering
	sort.Slice(enabled, func(i, j int) bool { return enabled[i].ID < enabled[j].ID })
	for _, t := range enabled {
		if !t.IsAuto() { // skip manual in step auto firing
			continue
		}
		bindings := bindingsMap[t.ID]
		if len(bindings) == 0 {
			continue
		}
		// Fire only first binding for this transition in this layer
		if err := e.FireTransition(cpn, t, bindings[0], marking); err != nil {
			return fired, fmt.Errorf("failed to fire transition %s: %v", t.Name, err)
		}
		fired++
	}
	return fired, nil
}

// IsCompleted checks if the CPN execution is completed
func (e *Engine) IsCompleted(cpn *models.CPN, marking *models.Marking) bool {
	return cpn.IsCompleted(marking)
}

// GetManualTransitions returns all enabled manual transitions
func (e *Engine) GetManualTransitions(cpn *models.CPN, marking *models.Marking) ([]*models.Transition, map[string][]TokenBinding, error) {
	enabledTransitions, bindingsMap, err := e.GetEnabledTransitions(cpn, marking)
	if err != nil {
		return nil, nil, err
	}

	var manualTransitions []*models.Transition
	manualBindingsMap := make(map[string][]TokenBinding)

	for _, transition := range enabledTransitions {
		if transition.IsManual() {
			manualTransitions = append(manualTransitions, transition)
			manualBindingsMap[transition.ID] = bindingsMap[transition.ID]
		}
	}

	return manualTransitions, manualBindingsMap, nil
}
