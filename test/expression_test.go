package test

import (
	"testing"
	"go-petri-flow/internal/expression"
	"go-petri-flow/internal/models"
)

func TestBasicExpressionEvaluation(t *testing.T) {
	evaluator := expression.NewEvaluator()
	defer evaluator.Close()

	context := expression.NewEvaluationContext()

	// Test simple arithmetic
	result, err := evaluator.EvaluateArcExpression("2 + 3", context)
	if err != nil {
		t.Fatalf("Failed to evaluate arithmetic expression: %v", err)
	}
	if result != 5 {
		t.Errorf("Expected 5, got %v", result)
	}

	// Test string operations
	result, err = evaluator.EvaluateArcExpression("'hello' .. ' world'", context)
	if err != nil {
		t.Fatalf("Failed to evaluate string expression: %v", err)
	}
	if result != "hello world" {
		t.Errorf("Expected 'hello world', got %v", result)
	}

	// Test boolean operations
	guardResult, err := evaluator.EvaluateGuard("true and false", context)
	if err != nil {
		t.Fatalf("Failed to evaluate boolean expression: %v", err)
	}
	if guardResult != false {
		t.Errorf("Expected false, got %v", guardResult)
	}
}

func TestVariableBinding(t *testing.T) {
	evaluator := expression.NewEvaluator()
	defer evaluator.Close()

	context := expression.NewEvaluationContext()

	// Bind a variable
	token := models.NewToken(42, 10)
	context.BindVariable("x", token)

	// Test using the variable in an expression
	result, err := evaluator.EvaluateArcExpression("x + 1", context)
	if err != nil {
		t.Fatalf("Failed to evaluate expression with variable: %v", err)
	}
	if result != 43 {
		t.Errorf("Expected 43, got %v", result)
	}

	// Test using variable timestamp
	result, err = evaluator.EvaluateArcExpression("x_timestamp", context)
	if err != nil {
		t.Fatalf("Failed to evaluate timestamp expression: %v", err)
	}
	if result != 10 {
		t.Errorf("Expected 10, got %v", result)
	}

	// Test guard with variable
	guardResult, err := evaluator.EvaluateGuard("x > 40", context)
	if err != nil {
		t.Fatalf("Failed to evaluate guard with variable: %v", err)
	}
	if !guardResult {
		t.Error("Expected true, got false")
	}

	guardResult, err = evaluator.EvaluateGuard("x < 40", context)
	if err != nil {
		t.Fatalf("Failed to evaluate guard with variable: %v", err)
	}
	if guardResult {
		t.Error("Expected false, got true")
	}
}

func TestGlobalClock(t *testing.T) {
	evaluator := expression.NewEvaluator()
	defer evaluator.Close()

	context := expression.NewEvaluationContext()
	context.SetGlobalClock(100)

	// Test accessing global clock
	result, err := evaluator.EvaluateArcExpression("global_clock", context)
	if err != nil {
		t.Fatalf("Failed to evaluate global clock expression: %v", err)
	}
	if result != 100 {
		t.Errorf("Expected 100, got %v", result)
	}

	// Test using global clock in calculation
	result, err = evaluator.EvaluateArcExpression("global_clock + 5", context)
	if err != nil {
		t.Fatalf("Failed to evaluate global clock calculation: %v", err)
	}
	if result != 105 {
		t.Errorf("Expected 105, got %v", result)
	}
}

func TestTupleCreation(t *testing.T) {
	evaluator := expression.NewEvaluator()
	defer evaluator.Close()

	context := expression.NewEvaluationContext()

	// Test creating a tuple
	result, err := evaluator.EvaluateArcExpression("tuple(1, 'hello', true)", context)
	if err != nil {
		t.Fatalf("Failed to evaluate tuple expression: %v", err)
	}

	// Check if result is a slice
	slice, ok := result.([]interface{})
	if !ok {
		t.Fatalf("Expected slice, got %T", result)
	}

	if len(slice) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(slice))
	}

	if slice[0] != 1 {
		t.Errorf("Expected first element to be 1, got %v", slice[0])
	}
	if slice[1] != "hello" {
		t.Errorf("Expected second element to be 'hello', got %v", slice[1])
	}
	if slice[2] != true {
		t.Errorf("Expected third element to be true, got %v", slice[2])
	}
}

func TestComplexExpressions(t *testing.T) {
	evaluator := expression.NewEvaluator()
	defer evaluator.Close()

	context := expression.NewEvaluationContext()

	// Bind multiple variables
	context.BindVariable("x", models.NewToken(10, 0))
	context.BindVariable("y", models.NewToken("test", 5))

	// Test complex arithmetic
	result, err := evaluator.EvaluateArcExpression("x * 2 + 5", context)
	if err != nil {
		t.Fatalf("Failed to evaluate complex arithmetic: %v", err)
	}
	if result != 25 {
		t.Errorf("Expected 25, got %v", result)
	}

	// Test complex guard
	guardResult, err := evaluator.EvaluateGuard("x > 5 and y_timestamp < 10", context)
	if err != nil {
		t.Fatalf("Failed to evaluate complex guard: %v", err)
	}
	if !guardResult {
		t.Error("Expected true, got false")
	}

	// Test string manipulation
	result, err = evaluator.EvaluateArcExpression("y .. '_suffix'", context)
	if err != nil {
		t.Fatalf("Failed to evaluate string manipulation: %v", err)
	}
	if result != "test_suffix" {
		t.Errorf("Expected 'test_suffix', got %v", result)
	}
}

func TestErrorHandling(t *testing.T) {
	evaluator := expression.NewEvaluator()
	defer evaluator.Close()

	context := expression.NewEvaluationContext()

	// Test syntax error
	_, err := evaluator.EvaluateArcExpression("2 +", context)
	if err == nil {
		t.Error("Expected syntax error, got nil")
	}

	// Test undefined variable (Lua will return nil, not error)
	result, err := evaluator.EvaluateArcExpression("undefined_var", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil for undefined variable, got %v", result)
	}

	// Test guard returning non-boolean
	_, err = evaluator.EvaluateGuard("42", context)
	if err == nil {
		t.Error("Expected non-boolean guard error, got nil")
	}
}

func TestEmptyGuard(t *testing.T) {
	evaluator := expression.NewEvaluator()
	defer evaluator.Close()

	context := expression.NewEvaluationContext()

	// Test empty guard (should return true)
	guardResult, err := evaluator.EvaluateGuard("", context)
	if err != nil {
		t.Fatalf("Failed to evaluate empty guard: %v", err)
	}
	if !guardResult {
		t.Error("Empty guard should return true")
	}
}

func TestContextCloning(t *testing.T) {
	context := expression.NewEvaluationContext()
	context.SetGlobalClock(50)
	context.BindVariable("x", models.NewToken(100, 20))

	// Clone the context
	clone := context.Clone()

	// Verify clone has same values
	if clone.GlobalClock != 50 {
		t.Errorf("Expected global clock 50, got %d", clone.GlobalClock)
	}

	token, exists := clone.TokenBindings["x"]
	if !exists {
		t.Error("Variable x should exist in clone")
	} else {
		if token.Value != 100 {
			t.Errorf("Expected token value 100, got %v", token.Value)
		}
		if token.Timestamp != 20 {
			t.Errorf("Expected token timestamp 20, got %d", token.Timestamp)
		}
	}

	// Modify original and verify clone is unaffected
	context.SetGlobalClock(75)
	context.BindVariable("y", models.NewToken(200, 30))

	if clone.GlobalClock != 50 {
		t.Error("Clone should not be affected by changes to original")
	}

	if _, exists := clone.TokenBindings["y"]; exists {
		t.Error("Clone should not have variable y")
	}
}

func TestLuaBuiltinFunctions(t *testing.T) {
	evaluator := expression.NewEvaluator()
	defer evaluator.Close()

	context := expression.NewEvaluationContext()

	// Test type function
	result, err := evaluator.EvaluateArcExpression("type(42)", context)
	if err != nil {
		t.Fatalf("Failed to evaluate type function: %v", err)
	}
	if result != "number" {
		t.Errorf("Expected 'number', got %v", result)
	}

	// Test tostring function
	result, err = evaluator.EvaluateArcExpression("tostring(42)", context)
	if err != nil {
		t.Fatalf("Failed to evaluate tostring function: %v", err)
	}
	if result != "42" {
		t.Errorf("Expected '42', got %v", result)
	}

	// Test tonumber function
	result, err = evaluator.EvaluateArcExpression("tonumber('123')", context)
	if err != nil {
		t.Fatalf("Failed to evaluate tonumber function: %v", err)
	}
	if result != 123 { // tonumber returns integer when possible
		t.Errorf("Expected 123, got %v", result)
	}
}

