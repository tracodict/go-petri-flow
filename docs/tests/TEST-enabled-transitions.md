## Enabled Transitions & Binding Candidates (curl Examples)

This test file demonstrates retrieving only enabled transitions and their binding candidates (variable assignments) for a given CPN instance.

```sh
export FLOW_SVC=http://localhost:8082
```

### 1. Load a Net With Two Potential Bindings

We model a transition needing an INT token greater than 0; two tokens qualify, one does not.

```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "enabled-cpn-1",
    "name": "Enabled Demo",
    "description": "Shows enabled transitions & bindings",
    "colorSets": ["colset I = int;"],
    "places": [
      { "id":"p_numbers", "name":"Numbers", "colorSet":"I" },
      { "id":"p_out", "name":"Out", "colorSet":"I" }
    ],
    "transitions": [
      { "id":"t_gt0", "name":"GreaterThanZero", "kind":"Auto", "guardExpression":"x > 0", "variables":["x"] }
    ],
    "arcs": [
      { "id":"a_in",  "sourceId":"p_numbers", "targetId":"t_gt0", "expression":"x", "direction":"IN" },
      { "id":"a_out", "sourceId":"t_gt0", "targetId":"p_out", "expression":"x", "direction":"OUT" }
    ],
    "initialMarking": { "p_numbers": [ {"value": -1, "timestamp":0}, {"value": 1, "timestamp":0}, {"value": 3, "timestamp":0} ] }
  }'
```

### 2. List All Transitions (Mixed Enabled Flag)

```sh
curl -X GET "${FLOW_SVC}/api/transitions/list?id=enabled-cpn-1" | jq '.[] | {id,name,enabled,bindingCount}'
```
Expect: `t_gt0` enabled with bindingCount 2.

### 3. Retrieve Only Enabled Transitions

```sh
curl -X GET "${FLOW_SVC}/api/transitions/enabled?id=enabled-cpn-1" | jq '.'
```
Expect array with a single entry for `t_gt0` and a `bindings` array of size 2.

You should also see its `guardExpression` field populated ("x > 0"). `actionExpression`, `formSchema`, `layoutSchema` will be absent here (empty) because they weren't defined.

### 4. Transition With formSchema, layoutSchema and actionExpression

Load a second net that includes these extra properties on a manual transition so the enabled endpoint demonstrates them.

```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "enabled-cpn-2",
    "name": "Enabled Demo 2",
    "description": "Shows extra transition fields",
    "jsonSchemas": [
      { "name": "FormA",   "schema": {"type":"object","properties":{"v":{"type":"integer"}},"required":["v"]} },
      { "name": "LayoutA", "schema": {"type":"object","properties":{"layout":{"type":"string"}}} }
    ],
    "colorSets": ["colset I = int;"],
    "places": [
      { "id":"p_in",  "name":"In",  "colorSet":"I" },
      { "id":"p_out", "name":"Out", "colorSet":"I" }
    ],
    "transitions": [
      { "id":"t_manual", "name":"ManualStep", "kind":"Manual", "variables":["x"], "guardExpression":"x >= 10", "actionExpression":"x = x + 1", "formSchema":"FormA", "layoutSchema":"LayoutA" }
    ],
    "arcs": [
      { "id":"a_in",  "sourceId":"p_in",  "targetId":"t_manual", "expression":"x", "direction":"IN" },
      { "id":"a_out", "sourceId":"t_manual", "targetId":"p_out", "expression":"x", "direction":"OUT" }
    ],
    "initialMarking": { "p_in": [ {"value": 10, "timestamp":0} ] }
  }'

curl -X GET "${FLOW_SVC}/api/transitions/enabled?id=enabled-cpn-2" | jq '.[] | {id,name,enabled,guardExpression,actionExpression,formSchema,layoutSchema,bindings}'
```

Expect: `t_manual` enabled with one binding; all listed fields populated.

### 5. Fire One Binding and Re-query (First Net)

```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire \
  -H 'Content-Type: application/json' \
  -d '{"cpnId":"enabled-cpn-1","transitionId":"t_gt0","bindingIndex":0}'

curl -X GET "${FLOW_SVC}/api/transitions/enabled?id=enabled-cpn-2" | jq '.[0].bindings | length'
```
Expect: remaining bindings count becomes 1.

---
Document version: 1.0
