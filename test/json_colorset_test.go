package test

import (
	"encoding/json"
	"testing"

	"go-petri-flow/internal/engine"
	"go-petri-flow/internal/models"
)

func TestJSONColorSet_Untyped(t *testing.T) {
	parser := models.NewCPNParser()
	def := models.CPNDefinitionJSON{
		ID:             "json-untyped",
		Name:           "json-untyped",
		ColorSets:      []string{"colset Meta = json;"},
		Places:         []models.PlaceJSON{{ID: "p1", Name: "MetaIn", ColorSet: "Meta"}},
		Transitions:    []models.TransitionJSON{},
		Arcs:           []models.ArcJSON{},
		InitialMarking: map[string][]models.TokenJSON{"p1": {{Value: map[string]interface{}{"k": "v", "n": 1}, Timestamp: 0}}},
	}
	_, err := parser.ParseCPNFromDefinition(&def)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestJSONColorSet_WithSchemaValid(t *testing.T) {
	parser := models.NewCPNParser()
	schema := map[string]interface{}{"type": "object", "required": []interface{}{"id", "total"}, "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "total": map[string]interface{}{"type": "number"}}}
	def := models.CPNDefinitionJSON{
		ID:             "json-schema",
		Name:           "json-schema",
		JsonSchemas:    []models.JsonSchemaDef{{Name: "OrderSchema", Schema: schema}},
		ColorSets:      []string{"colset Order = json<OrderSchema>;"},
		Places:         []models.PlaceJSON{{ID: "p1", Name: "Orders", ColorSet: "Order"}},
		Transitions:    []models.TransitionJSON{},
		Arcs:           []models.ArcJSON{},
		InitialMarking: map[string][]models.TokenJSON{"p1": {{Value: map[string]interface{}{"id": "A1", "total": 10.5}, Timestamp: 0}}},
	}
	_, err := parser.ParseCPNFromDefinition(&def)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestJSONColorSet_WithSchemaInvalid(t *testing.T) {
	parser := models.NewCPNParser()
	schema := map[string]interface{}{"type": "object", "required": []interface{}{"id", "total"}, "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "total": map[string]interface{}{"type": "number"}}}
	def := models.CPNDefinitionJSON{
		ID:             "json-schema-bad",
		Name:           "json-schema-bad",
		JsonSchemas:    []models.JsonSchemaDef{{Name: "OrderSchema", Schema: schema}},
		ColorSets:      []string{"colset Order = json<OrderSchema>;"},
		Places:         []models.PlaceJSON{{ID: "p1", Name: "Orders", ColorSet: "Order"}},
		Transitions:    []models.TransitionJSON{},
		Arcs:           []models.ArcJSON{},
		InitialMarking: map[string][]models.TokenJSON{"p1": {{Value: map[string]interface{}{"id": "A1"}, Timestamp: 0}}},
	}
	_, err := parser.ParseCPNFromDefinition(&def)
	if err == nil {
		t.Fatalf("expected schema validation error")
	}
}

func TestJSONColorSet_AliasMap(t *testing.T) {
	parser := models.NewCPNParser()
	def := models.CPNDefinitionJSON{
		ID:             "json-map-alias",
		Name:           "json-map-alias",
		ColorSets:      []string{"colset Legacy = map;"},
		Places:         []models.PlaceJSON{{ID: "p1", Name: "Legacy", ColorSet: "Legacy"}},
		Transitions:    []models.TransitionJSON{},
		Arcs:           []models.ArcJSON{},
		InitialMarking: map[string][]models.TokenJSON{"p1": {{Value: map[string]interface{}{"hello": "world"}, Timestamp: 0}}},
	}
	_, err := parser.ParseCPNFromDefinition(&def)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

// Test transformation on output arc preserves schema validity and adds field
func TestJSONColorSet_OutputTransformation(t *testing.T) {
	parser := models.NewCPNParser()
	schema := map[string]interface{}{"type": "object", "required": []interface{}{"id", "total"}, "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "total": map[string]interface{}{"type": "number"}, "flag": map[string]interface{}{"type": "string"}}}
	def := models.CPNDefinitionJSON{
		ID:             "json-transform",
		Name:           "json-transform",
		JsonSchemas:    []models.JsonSchemaDef{{Name: "OrderSchema", Schema: schema}},
		ColorSets:      []string{"colset Order = json<OrderSchema>;"},
		Places:         []models.PlaceJSON{{ID: "p_in", Name: "In", ColorSet: "Order"}, {ID: "p_out", Name: "Out", ColorSet: "Order"}},
		Transitions:    []models.TransitionJSON{{ID: "t1", Name: "T", Kind: "Auto"}},
		Arcs:           []models.ArcJSON{{ID: "a_in", SourceID: "p_in", TargetID: "t1", Expression: "order", Direction: "IN"}, {ID: "a_out", SourceID: "t1", TargetID: "p_out", Expression: "local o=order; o.flag=\"X\"; return o", Direction: "OUT"}},
		InitialMarking: map[string][]models.TokenJSON{"p_in": {{Value: map[string]interface{}{"id": "A", "total": 5}, Timestamp: 0}}},
	}
	cpn, err := parser.ParseCPNFromDefinition(&def)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	eng := engine.NewEngine()
	defer eng.Close()
	marking := models.NewMarking()
	// seed initial marking
	for placeID, toks := range cpn.InitialMarking {
		for _, tk := range toks {
			marking.AddToken(placeID, tk)
		}
	}
	enabled, bindings, err := eng.GetEnabledTransitions(cpn, marking)
	if err != nil || len(enabled) != 1 {
		t.Fatalf("expected 1 enabled, got %v err %v", len(enabled), err)
	}
	if err := eng.FireTransition(cpn, enabled[0], bindings[enabled[0].ID][0], marking); err != nil {
		t.Fatalf("fire failed: %v", err)
	}
	// verify new token has flag
	out := marking.Places["p_out"].GetAllTokens()
	if len(out) != 1 {
		t.Fatalf("expected 1 token in Out")
	}
	val, _ := out[0].Value.(map[string]interface{})
	if _, ok := val["flag"]; !ok {
		t.Fatalf("expected flag field added")
	}
}

// Test guard using JSON field
func TestJSONColorSet_Guard(t *testing.T) {
	parser := models.NewCPNParser()
	schema := map[string]interface{}{"type": "object", "required": []interface{}{"id", "total"}, "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "total": map[string]interface{}{"type": "number"}}}
	def := models.CPNDefinitionJSON{
		ID:             "json-guard",
		Name:           "json-guard",
		JsonSchemas:    []models.JsonSchemaDef{{Name: "OrderSchema", Schema: schema}},
		ColorSets:      []string{"colset Order = json<OrderSchema>;"},
		Places:         []models.PlaceJSON{{ID: "p_in", Name: "In", ColorSet: "Order"}},
		Transitions:    []models.TransitionJSON{{ID: "t1", Name: "T", Kind: "Manual", GuardExpression: "order.total > 10", Variables: []string{"order"}}},
		Arcs:           []models.ArcJSON{{ID: "a_in", SourceID: "p_in", TargetID: "t1", Expression: "order", Direction: "IN"}},
		InitialMarking: map[string][]models.TokenJSON{"p_in": {{Value: map[string]interface{}{"id": "A", "total": 5}, Timestamp: 0}}},
	}
	cpn, err := parser.ParseCPNFromDefinition(&def)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	eng := engine.NewEngine()
	defer eng.Close()
	marking := models.NewMarking()
	for placeID, toks := range cpn.InitialMarking {
		for _, tk := range toks {
			marking.AddToken(placeID, tk)
		}
	}
	enabled, _, err := eng.GetEnabledTransitions(cpn, marking)
	if err != nil {
		t.Fatalf("enabled err: %v", err)
	}
	if len(enabled) != 0 {
		t.Fatalf("guard should block transition")
	}
}

// Test output schema violation (adds wrong type) should fail
func TestJSONColorSet_OutputSchemaViolation(t *testing.T) {
	parser := models.NewCPNParser()
	schema := map[string]interface{}{"type": "object", "required": []interface{}{"id", "total"}, "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}, "total": map[string]interface{}{"type": "number"}}}
	def := models.CPNDefinitionJSON{
		ID:             "json-out-bad",
		Name:           "json-out-bad",
		JsonSchemas:    []models.JsonSchemaDef{{Name: "OrderSchema", Schema: schema}},
		ColorSets:      []string{"colset Order = json<OrderSchema>;"},
		Places:         []models.PlaceJSON{{ID: "p_in", Name: "In", ColorSet: "Order"}, {ID: "p_out", Name: "Out", ColorSet: "Order"}},
		Transitions:    []models.TransitionJSON{{ID: "t1", Name: "T", Kind: "Auto"}},
		Arcs:           []models.ArcJSON{{ID: "a_in", SourceID: "p_in", TargetID: "t1", Expression: "order", Direction: "IN"}, {ID: "a_out", SourceID: "t1", TargetID: "p_out", Expression: "return { id = order.id, total = \"oops\" }", Direction: "OUT"}},
		InitialMarking: map[string][]models.TokenJSON{"p_in": {{Value: map[string]interface{}{"id": "A", "total": 5}, Timestamp: 0}}},
	}
	cpn, err := parser.ParseCPNFromDefinition(&def)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	eng := engine.NewEngine()
	defer eng.Close()
	marking := models.NewMarking()
	for pn, toks := range cpn.InitialMarking {
		for _, tk := range toks {
			marking.AddToken(pn, tk)
		}
	}
	enabled, bindings, err := eng.GetEnabledTransitions(cpn, marking)
	if err != nil {
		t.Fatalf("enabled err: %v", err)
	}
	if len(enabled) != 1 {
		t.Fatalf("expected 1 enabled")
	}
	if err := eng.FireTransition(cpn, enabled[0], bindings[enabled[0].ID][0], marking); err == nil {
		t.Fatalf("expected firing error due to schema violation")
	}
}

// Test array output for untyped json
func TestJSONColorSet_ArrayOutputUntyped(t *testing.T) {
	parser := models.NewCPNParser()
	def := models.CPNDefinitionJSON{
		ID:             "json-array",
		Name:           "json-array",
		ColorSets:      []string{"colset J = json;"},
		Places:         []models.PlaceJSON{{ID: "p_in", Name: "In", ColorSet: "J"}, {ID: "p_out", Name: "Out", ColorSet: "J"}},
		Transitions:    []models.TransitionJSON{{ID: "t1", Name: "T", Kind: "Auto"}},
		Arcs:           []models.ArcJSON{{ID: "a_in", SourceID: "p_in", TargetID: "t1", Expression: "x", Direction: "IN"}, {ID: "a_out", SourceID: "t1", TargetID: "p_out", Expression: "return {1,2,3}", Direction: "OUT"}},
		InitialMarking: map[string][]models.TokenJSON{"p_in": {{Value: []interface{}{map[string]interface{}{"dummy": true}}, Timestamp: 0}}},
	}
	cpn, err := parser.ParseCPNFromDefinition(&def)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	eng := engine.NewEngine()
	defer eng.Close()
	marking := models.NewMarking()
	for pn, toks := range cpn.InitialMarking {
		for _, tk := range toks {
			marking.AddToken(pn, tk)
		}
	}
	enabled, bindings, err := eng.GetEnabledTransitions(cpn, marking)
	if err != nil {
		t.Fatalf("enabled err: %v", err)
	}
	if len(enabled) != 1 {
		t.Fatalf("expected 1 enabled")
	}
	if err := eng.FireTransition(cpn, enabled[0], bindings[enabled[0].ID][0], marking); err != nil {
		t.Fatalf("fire failed: %v", err)
	}
	if marking.Places["p_out"].Size() == 0 {
		t.Fatalf("expected token array output")
	}
}

// Utility to pretty print for debugging if needed
func toJSON(v interface{}) string { b, _ := json.Marshal(v); return string(b) }
