# JSON Color Set Design

Goal: Allow users to define color sets whose tokens carry arbitrary JSON objects, optionally validated against a JSON Schema. Lua expressions (guards / arc expressions) must be able to read and produce JSON structures.

---

## 1. Overview

Add a new base kind: JSON (map / array structured values).  
Two variants:
1. json (untyped) – any JSON object/array.
2. json<SchemaName> – must satisfy a registered JSON Schema.

Example color set declarations (in existing `colorSets` string list):
```
colset Order = json<OrderSchema>;
colset Meta = json;
```

A companion section (new optional field in CPN JSON) provides schema definitions:
```
"jsonSchemas": [
  { "name": "OrderSchema", "schema": { "$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "required": ["id","total"], "properties": { "id": { "type": "string" }, "total": { "type": "number" }, "customer": { "type":"object", "properties": { "vip": { "type":"boolean"} } } } } }
]
```

(If no schema definitions are provided, a referenced schema name is an error.)

---

## 2. Data Structures

Extend / introduce:

```
type JsonColorSet struct {
    name        string
    schemaName  string        // empty if none
    schema      *jsonschema.Schema // compiled; nil if none
}

func (j *JsonColorSet) Name() string
func (j *JsonColorSet) IsMember(v any) bool  // schema-based or always true
func (j *JsonColorSet) Kind() ColorSetKind   // ColorSetKindJSON
```

Update `ColorSetKind` enum to add `JSON`.

Add registry in `ColorSetParser`:
```
jsonSchemas map[string]*jsonschema.Schema
```

---

## 3. Parsing Enhancements

Grammar additions (simplified):
```
JsonTypeDef := 'json' ( '<' Identifier '>' )? ';'
```

Parsing steps:
1. Detect prefix `colset` and name.
2. If right-hand side matches `json;` → create JsonColorSet (no schema).
3. If matches `json<SchemaName>;` → defer linking until schemas are loaded.
4. After all schema definitions loaded, resolve pending JsonColorSets (compile schema JSON once; cache in map).

Error cases:
- Unknown schema name.
- Invalid JSON Schema compilation.

---

## 4. CPN JSON Additions

Add optional top-level field:
```
"jsonSchemas": [
  { "name": "OrderSchema", "schema": { ... raw JSON schema ... } }
]
```

Load order:
1. Parse `jsonSchemas` → compile & store.
2. Parse `colorSets` → build color set instances; resolve schema references.

Backward compatible: existing nets without `jsonSchemas` unaffected.

---

## 5. Token Representation

Internal value type: `map[string]any` for JSON objects; `[]any` for arrays.

`Token.Value` already `any` → no struct change.

When serializing back to JSON, standard encoder handles maps/slices.

---

## 6. Validation Logic

`JsonColorSet.IsMember(v any)`:
1. Accept only `map[string]any` or `[]any` (and primitive if you decide to allow — here restrict to object/array).
2. If schema == nil → return true.
3. Run schema validation (pass raw `v`).
4. On failure return false (or richer error path if extending interface to `Validate(v any) error`).

Performance:
- Precompile schema once.
- Optionally cache last N validation successes by pointer hash (micro-optimization; skip initially).

---

## 7. Lua Integration

### 7.1 Injection
Current evaluation likely converts Go values to Lua:
- Add recursive converter:  
  map[string]any → Lua table with string keys  
  []any → array part  
  Numbers stay number (float64). Booleans & strings unchanged. Nil preserved.

### 7.2 Extraction (Output Arc)
If Lua result is:
- Table → convert back to map[string]any (if non-sequential keys) or []any (if strictly 1..N integer keys).
- Primitive → allowed only if schema-less or schema permits (object/array required otherwise).

After extraction:
- Run `IsMember` (schema validation); on failure: abort firing, report validation error.

### 7.3 Mutation Patterns
Because tables are passed by reference inside Lua, user can:
- Return modified same table.
- Or construct new table and return it.

We require an explicit return (or rely on auto-`return` wrapper for expression form).

---

## 8. Examples

### 8.1 Definition
```
"colorSets": [
  "colset Order = json<OrderSchema>;"
],
"jsonSchemas": [
  {
    "name": "OrderSchema",
    "schema": {
      "$schema": "https://json-schema.org/draft/2020-12/schema",
      "type": "object",
      "required": ["id","total"],
      "properties": {
        "id": { "type":"string" },
        "total": { "type":"number" },
        "customer": {
          "type":"object",
          "properties": { "vip": { "type":"boolean" } }
        }
      }
    }
  }
]
```

Place referencing:
```
{ "id":"p_orderIn", "name":"OrderIn", "colorSet":"Order" }
{ "id":"p_flag", "name":"Flag", "colorSet":"STRING" }
```

### 8.2 Initial Token
```
"initialMarking": {
  "OrderIn": [
    { "value": { "id":"A123", "total": 250.0, "customer": { "vip": true } }, "timestamp": 0 }
  ]
}
```

### 8.3 Guard Expression (read JSON)
```
order.customer and order.customer.vip == true and order.total > 100
```
If no `return`, wrapper adds `return`.

### 8.4 Output Arc Transform
Input arc expression (bind variable):
```
"expression": "order"
```
Output arc expression (apply discount & annotate):
```
local o = order          -- copy reference
if o.total > 200 then
  o.total = o.total * 0.9
  o.discount = "10%"
end
return o
```

### 8.5 Derive Classification
Output arc to STRING place:
```
(order.total > 500 and "BIG") or (order.total > 100 and "MEDIUM") or "SMALL"
```

### 8.6 Building a Derived JSON
```
return {
  id = order.id,
  vip = (order.customer and order.customer.vip) or false,
  bucket = (order.total > 500 and "PLATINUM") or (order.total > 250 and "GOLD") or "STANDARD"
}
```

---

## 9. Error Reporting

Enhance validation result structure:
```
{
  "valid": false,
  "violations": [
    { "code": "token_schema_violation", "placeId":"p_orderIn", "detail":"missing required property: total" }
  ]
}
```

During transition firing:
- If output JSON fails schema: transition fails atomically; engine returns error: `output_arc_validation_failed`.

---

## 10. API / Parser Changes Summary

1. Add `jsonSchemas` array to CPNDefinitionJSON.
2. Extend parser:
   - Parse `jsonSchemas` first (compile).
   - Parse `colorSets` (resolve json<...> references).
3. Register new endpoint doc update (if needed to list schemas).
4. Update `/api/cpn/get` to include `jsonSchemas` back (optional).
5. Update validation endpoint to test sample JSON token via `IsMember`.

---

## 11. Backward Compatibility

- Nets without JSON color sets unaffected.
- `jsonMap` (earlier ad-hoc) can be aliased to `json`.
- If both used, deprecate `jsonMap` in favor of `json`.

---

## 12. Security Considerations

- JSON Schema recursion limits to avoid DoS (set max depth / compiled schema size).
- Lua sandbox already in place: ensure JSON tables do not inject metatables.
- Prevent extremely large JSON tokens (enforce configurable size limit).

---

## 13. Future Extensions

- Schema versioning: `json<OrderSchema@v2>`.
- Inline schema: `colset Order = json{ ...inline schema... };` (escaped).
- Typed access helpers in Lua (utility library for deep get / default).

---

## 14. Implementation Steps (Checklist)

- [ ] Add ColorSetKindJSON.
- [ ] Implement JsonColorSet.
- [ ] Extend parser for jsonSchemas + json<SchemaName>.
- [ ] Integrate schema compiler (choose lib, e.g., github.com/santhosh-tekuri/jsonschema/v5).
- [ ] Add Lua table (de)serialization helpers.
- [ ] Enforce validation on:
  - Loading initial marking.
  - Transition consumption (optional).
  - Transition production (mandatory).
- [ ] Extend validation endpoint to report schema violations.
- [ ] Add tests (valid token, invalid token, transform passes/fails).
- [ ] Update docs & examples.

---

## 15. Minimal Code Sketch (Interfaces)

```go
type JsonColorSet struct {
    name       string
    schemaName string
    schema     *jsonschema.Schema
}

func (j *JsonColorSet) Name() string { return j.name }
func (j *JsonColorSet) IsMember(v any) bool {
    switch v.(type) {
    case map[string]any, []any:
        if j.schema == nil { return true }
        return j.schema.Validate(v) == nil
    default:
        return false
    }
}
```

Lua conversion (outline):
```go
func goToLua(L *lua.LState, v any) lua.LValue { /* recursive */ }
func luaToGo(v lua.LValue) any { /* reverse */ }
```

---

## 16. Testing Scenarios

1. Load CPN with json color set + schema → success.
2. Initial token violates schema → load fails (or marked invalid).
3. Transition creates invalid JSON → firing rejected.
4. Guard reads nested JSON fields → correct enablement.
5. Performance: many validations cached.

---