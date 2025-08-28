package expression

import (
	"fmt"
	"strconv"
	"strings"

	"go-petri-flow/internal/models"

	lua "github.com/yuin/gopher-lua"
)

// EvaluationContext holds the context for expression evaluation
type EvaluationContext struct {
	TokenBindings map[string]*models.Token   // Variable name -> Token
	GlobalClock   int                        // Current global clock
	PlaceTokens   map[string][]*models.Token // Place name -> Available tokens
	ColorSets     map[string]models.ColorSet // Color set registry
}

// NewEvaluationContext creates a new evaluation context
func NewEvaluationContext() *EvaluationContext {
	return &EvaluationContext{
		TokenBindings: make(map[string]*models.Token),
		GlobalClock:   0,
		PlaceTokens:   make(map[string][]*models.Token),
		ColorSets:     make(map[string]models.ColorSet),
	}
}

// Evaluator handles expression evaluation using gopher-lua
type Evaluator struct {
	luaState *lua.LState
}

// NewEvaluator creates a new expression evaluator
func NewEvaluator() *Evaluator {
	L := lua.NewState()

	evaluator := &Evaluator{
		luaState: L,
	}

	// Register CPN-specific functions
	evaluator.registerCPNFunctions()

	return evaluator
}

// Close closes the Lua state
func (e *Evaluator) Close() {
	if e.luaState != nil {
		e.luaState.Close()
	}
}

// GetGlobalValue returns the current Lua global value for a variable name converted to Go.
// Returns nil if the variable is not defined.
func (e *Evaluator) GetGlobalValue(varName string) interface{} {
	if e.luaState == nil { return nil }
	lv := e.luaState.GetGlobal(varName)
	if lv == lua.LNil { return nil }
	return e.luaValueToGo(lv)
}

// EvaluateGuard evaluates a guard expression and returns true/false
func (e *Evaluator) EvaluateGuard(expression string, context *EvaluationContext) (bool, error) {
	if expression == "" {
		return true, nil // Empty guard is always true
	}

	// Set up the Lua environment with context
	if err := e.setupLuaContext(context); err != nil {
		return false, fmt.Errorf("failed to setup Lua context: %v", err)
	}

	// Evaluate the expression
	result, err := e.evaluateLuaExpression(expression)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate guard expression '%s': %v", expression, err)
	}

	// Convert result to boolean
	boolResult, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("guard expression '%s' did not return a boolean value, got %T", expression, result)
	}

	return boolResult, nil
}

// EvaluateArcExpression evaluates an arc expression and returns the result
func (e *Evaluator) EvaluateArcExpression(expression string, context *EvaluationContext) (interface{}, error) {
	if expression == "" {
		return nil, fmt.Errorf("arc expression cannot be empty")
	}

	// Set up the Lua environment with context
	if err := e.setupLuaContext(context); err != nil {
		return nil, fmt.Errorf("failed to setup Lua context: %v", err)
	}

	result, err := e.evaluateLuaExpression(expression)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate arc expression '%s': %v", expression, err)
	}

	return result, nil
}

// EvaluateAction executes an action expression that may contain statements (assignments, loops, etc.).
// It doesn't enforce a return value. Any final expression result is ignored.
func (e *Evaluator) EvaluateAction(action string, context *EvaluationContext) error {
	if action == "" {
		return nil
	}
	if err := e.setupLuaContext(context); err != nil {
		return fmt.Errorf("failed to setup Lua context: %v", err)
	}
	// For actions we allow full Lua chunks. Ensure it compiles by leaving as-is.
	// Provide implicit do-end wrapper so single line assignment still works uniformly.
	chunk := action
	if !strings.HasPrefix(strings.TrimSpace(action), "do") && strings.Contains(action, "=") {
		// wrap only if it's a plain assignment without control keywords or return
		lowered := strings.ToLower(action)
		if !strings.Contains(lowered, "return") && !strings.Contains(lowered, "if ") && !strings.Contains(lowered, "for ") && !strings.Contains(lowered, "while ") {
			chunk = "do " + action + " end"
		}
	}
	if err := e.luaState.DoString(chunk); err != nil {
		return fmt.Errorf("Lua action execution error: %v", err)
	}
	return nil
}

// setupLuaContext sets up the Lua environment with the evaluation context
func (e *Evaluator) setupLuaContext(context *EvaluationContext) error {
	L := e.luaState

	// Set global clock
	L.SetGlobal("global_clock", lua.LNumber(context.GlobalClock))

	// Set token bindings as variables
	for varName, token := range context.TokenBindings {
		luaValue, err := e.goValueToLua(token.Value)
		if err != nil {
			return fmt.Errorf("failed to convert token value for variable %s: %v", varName, err)
		}
		L.SetGlobal(varName, luaValue)

		// Also set timestamp for the variable
		L.SetGlobal(varName+"_timestamp", lua.LNumber(token.Timestamp))
	}

	// Set place tokens (for more complex expressions that might need to access place contents)
	placeTable := L.NewTable()
	for placeName, tokens := range context.PlaceTokens {
		tokenTable := L.NewTable()
		for i, token := range tokens {
			tokenLuaTable := L.NewTable()

			valueLua, err := e.goValueToLua(token.Value)
			if err != nil {
				return fmt.Errorf("failed to convert token value for place %s: %v", placeName, err)
			}

			tokenLuaTable.RawSetString("value", valueLua)
			tokenLuaTable.RawSetString("timestamp", lua.LNumber(token.Timestamp))

			tokenTable.RawSetInt(i+1, tokenLuaTable) // Lua arrays are 1-indexed
		}
		placeTable.RawSetString(placeName, tokenTable)
	}
	L.SetGlobal("places", placeTable)

	return nil
}

// evaluateLuaExpression evaluates a Lua expression and returns the result
func (e *Evaluator) evaluateLuaExpression(expression string) (interface{}, error) {
	L := e.luaState

	// Wrap the expression in a return statement if it doesn't already have one
	luaCode := expression
	if !strings.Contains(strings.ToLower(expression), "return") {
		luaCode = "return " + expression
	}

	// Execute the Lua code
	if err := L.DoString(luaCode); err != nil {
		return nil, fmt.Errorf("Lua execution error: %v", err)
	}

	// Get the result from the stack
	result := L.Get(-1)
	L.Pop(1)

	// Convert Lua value back to Go value
	return e.luaValueToGo(result), nil
}

// goValueToLua converts a Go value to a Lua value
func (e *Evaluator) goValueToLua(value interface{}) (lua.LValue, error) {
	switch v := value.(type) {
	case nil:
		return lua.LNil, nil
	case bool:
		return lua.LBool(v), nil
	case int:
		return lua.LNumber(v), nil
	case int32:
		return lua.LNumber(v), nil
	case int64:
		return lua.LNumber(v), nil
	case float32:
		return lua.LNumber(v), nil
	case float64:
		return lua.LNumber(v), nil
	case string:
		return lua.LString(v), nil
	case []interface{}:
		// Convert slice to Lua table
		table := e.luaState.NewTable()
		for i, item := range v {
			luaItem, err := e.goValueToLua(item)
			if err != nil {
				return nil, fmt.Errorf("failed to convert slice item %d: %v", i, err)
			}
			table.RawSetInt(i+1, luaItem) // Lua arrays are 1-indexed
		}
		return table, nil
	case map[string]interface{}:
		// Convert map to Lua table
		table := e.luaState.NewTable()
		for key, val := range v {
			luaVal, err := e.goValueToLua(val)
			if err != nil {
				return nil, fmt.Errorf("failed to convert map value for key %s: %v", key, err)
			}
			table.RawSetString(key, luaVal)
		}
		return table, nil
	default:
		// For other types, try to convert to string
		return lua.LString(fmt.Sprintf("%v", v)), nil
	}
}

// luaValueToGo converts a Lua value to a Go value
func (e *Evaluator) luaValueToGo(value lua.LValue) interface{} {
	switch v := value.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		// Try to determine if it's an integer or float
		num := float64(v)
		if num == float64(int64(num)) {
			return int(num)
		}
		return num
	case lua.LString:
		return string(v)
	case *lua.LTable:
		// Check if it's an array or a map
		if e.isLuaArray(v) {
			return e.luaTableToSlice(v)
		}
		return e.luaTableToMap(v)
	default:
		return v.String()
	}
}

// isLuaArray checks if a Lua table is an array (consecutive integer keys starting from 1)
func (e *Evaluator) isLuaArray(table *lua.LTable) bool {
	length := table.Len()
	if length == 0 {
		return false
	}

	for i := 1; i <= length; i++ {
		if table.RawGetInt(i) == lua.LNil {
			return false
		}
	}

	// Check if there are any non-integer keys
	hasNonIntegerKeys := false
	table.ForEach(func(key, value lua.LValue) {
		if _, ok := key.(lua.LNumber); !ok {
			hasNonIntegerKeys = true
		} else {
			keyNum := int(key.(lua.LNumber))
			if keyNum < 1 || keyNum > length {
				hasNonIntegerKeys = true
			}
		}
	})

	return !hasNonIntegerKeys
}

// luaTableToSlice converts a Lua table to a Go slice
func (e *Evaluator) luaTableToSlice(table *lua.LTable) []interface{} {
	length := table.Len()
	result := make([]interface{}, length)

	for i := 1; i <= length; i++ {
		result[i-1] = e.luaValueToGo(table.RawGetInt(i))
	}

	return result
}

// luaTableToMap converts a Lua table to a Go map
func (e *Evaluator) luaTableToMap(table *lua.LTable) map[string]interface{} {
	result := make(map[string]interface{})

	table.ForEach(func(key, value lua.LValue) {
		keyStr := e.luaValueToGo(key)
		valueGo := e.luaValueToGo(value)
		result[fmt.Sprintf("%v", keyStr)] = valueGo
	})

	return result
}

// registerCPNFunctions registers CPN-specific functions in the Lua environment
func (e *Evaluator) registerCPNFunctions() {
	L := e.luaState

	// Register utility functions
	L.SetGlobal("print", L.NewFunction(e.luaPrint))
	L.SetGlobal("type", L.NewFunction(e.luaType))
	L.SetGlobal("tostring", L.NewFunction(e.luaToString))
	L.SetGlobal("tonumber", L.NewFunction(e.luaToNumber))

	// Register CPN-specific functions
	L.SetGlobal("token", L.NewFunction(e.luaCreateToken))
	L.SetGlobal("tuple", L.NewFunction(e.luaCreateTuple))
	L.SetGlobal("delay", L.NewFunction(e.luaDelay))
}

// Lua function implementations

func (e *Evaluator) luaPrint(L *lua.LState) int {
	// Simple print function for debugging
	args := make([]string, L.GetTop())
	for i := 1; i <= L.GetTop(); i++ {
		args[i-1] = L.Get(i).String()
	}
	fmt.Println(strings.Join(args, "\t"))
	return 0
}

func (e *Evaluator) luaType(L *lua.LState) int {
	value := L.Get(1)
	L.Push(lua.LString(value.Type().String()))
	return 1
}

func (e *Evaluator) luaToString(L *lua.LState) int {
	value := L.Get(1)
	L.Push(lua.LString(value.String()))
	return 1
}

func (e *Evaluator) luaToNumber(L *lua.LState) int {
	value := L.Get(1)
	switch v := value.(type) {
	case lua.LNumber:
		L.Push(v)
	case lua.LString:
		if num, err := strconv.ParseFloat(string(v), 64); err == nil {
			L.Push(lua.LNumber(num))
		} else {
			L.Push(lua.LNil)
		}
	default:
		L.Push(lua.LNil)
	}
	return 1
}

func (e *Evaluator) luaCreateToken(L *lua.LState) int {
	// Create a token value (just return the value for now)
	value := L.Get(1)
	L.Push(value)
	return 1
}

func (e *Evaluator) luaCreateTuple(L *lua.LState) int {
	// Create a tuple from arguments
	args := L.GetTop()
	table := L.NewTable()

	for i := 1; i <= args; i++ {
		table.RawSetInt(i, L.Get(i))
	}

	L.Push(table)
	return 1
}

func (e *Evaluator) luaDelay(L *lua.LState) int {
	// Create a delayed token (for arc expressions with @+delay syntax)
	value := L.Get(1)
	delay := L.Get(2)

	// For now, just return a table with value and delay
	table := L.NewTable()
	table.RawSetString("value", value)
	table.RawSetString("delay", delay)

	L.Push(table)
	return 1
}

// Helper functions for token binding

// BindVariable binds a variable to a token in the evaluation context
func (ctx *EvaluationContext) BindVariable(varName string, token *models.Token) {
	ctx.TokenBindings[varName] = token
}

// UnbindVariable removes a variable binding
func (ctx *EvaluationContext) UnbindVariable(varName string) {
	delete(ctx.TokenBindings, varName)
}

// ClearBindings clears all variable bindings
func (ctx *EvaluationContext) ClearBindings() {
	ctx.TokenBindings = make(map[string]*models.Token)
}

// SetPlaceTokens sets the available tokens for a place
func (ctx *EvaluationContext) SetPlaceTokens(placeName string, tokens []*models.Token) {
	ctx.PlaceTokens[placeName] = tokens
}

// SetGlobalClock sets the global clock value
func (ctx *EvaluationContext) SetGlobalClock(clock int) {
	ctx.GlobalClock = clock
}

// RegisterColorSet registers a color set in the context
func (ctx *EvaluationContext) RegisterColorSet(colorSet models.ColorSet) {
	ctx.ColorSets[colorSet.Name()] = colorSet
}

// SetValue sets or overrides a variable binding with a raw value (used for external form data)
func (ctx *EvaluationContext) SetValue(varName string, value interface{}) {
	// Represent raw value as a token with timestamp = 0 (non-timed semantics). If already a token, keep.
	if token, ok := value.(*models.Token); ok {
		ctx.TokenBindings[varName] = token
		return
	}
	ctx.TokenBindings[varName] = models.NewToken(value, 0)
}

// Clone creates a copy of the evaluation context
func (ctx *EvaluationContext) Clone() *EvaluationContext {
	clone := &EvaluationContext{
		TokenBindings: make(map[string]*models.Token),
		GlobalClock:   ctx.GlobalClock,
		PlaceTokens:   make(map[string][]*models.Token),
		ColorSets:     make(map[string]models.ColorSet),
	}

	// Copy token bindings
	for varName, token := range ctx.TokenBindings {
		clone.TokenBindings[varName] = token.Clone()
	}

	// Copy place tokens
	for placeName, tokens := range ctx.PlaceTokens {
		clonedTokens := make([]*models.Token, len(tokens))
		for i, token := range tokens {
			clonedTokens[i] = token.Clone()
		}
		clone.PlaceTokens[placeName] = clonedTokens
	}

	// Copy color sets (shallow copy is fine as they're immutable)
	for name, cs := range ctx.ColorSets {
		clone.ColorSets[name] = cs
	}

	return clone
}
