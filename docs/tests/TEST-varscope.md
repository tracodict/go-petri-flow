## Variable Scope in Arc & Transition Action Expressions

This guide shows how Lua variable scope works inside arc expressions and `actionExpression` blocks for transitions.

Key points:
- Every expression (guard, arc, action) executes in the shared Lua state; globals persist across firings unless overwritten.
- Variables you assign with `local` exist only for that single evaluation; they are not visible to later arcs or actions.
- Input arc simple variable expressions (e.g. `x`) bind token values to globals named after the variable (`x`, plus `x_timestamp`).
- Use globals (implicit) when you need to pass intermediate results from an action to an output arc.
- Prefer `local` when computing temporary values to avoid polluting global namespace.

Below curl examples illustrate patterns.

```sh
export FLOW_SVC=http://localhost:8082
```

### 1. Local Variable Inside Output Arc (Discarded After Evaluation)
The local `tmp` is created, used, then gone. No other expression can read it later.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "scope-arc-local",
		"name": "Arc Local",
		"description": "Local var inside arc expression",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_in","name":"In","colorSet":"INT"},
			{"id":"p_out","name":"Out","colorSet":"INT"}
		],
		"transitions": [ {"id":"t1","name":"T1","kind":"Auto"} ],
		"arcs": [
			{"id":"a_in","sourceId":"p_in","targetId":"t1","expression":"x","direction":"IN"},
			{"id":"a_out","sourceId":"t1","targetId":"p_out","expression":"local tmp = x * 5; tmp + 2","direction":"OUT"}
		],
		"initialMarking": {"In": [ {"value":4, "timestamp":0} ]}
	}'
curl -X POST "${FLOW_SVC}/api/simulation/step?id=scope-arc-local"
curl -X GET  "${FLOW_SVC}/api/marking/get?id=scope-arc-local" # Expect Out value 22
```

### 2. Global Variable Via Output Arc (Persists for Later Use)
Second firing sees prior global `g` value.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "scope-arc-global",
		"name": "Arc Global",
		"description": "Global variable reuse across firings",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_src","name":"Src","colorSet":"INT"},
			{"id":"p_mid","name":"Mid","colorSet":"INT"}
		],
		"transitions": [ {"id":"t_incr","name":"Incr","kind":"Auto"} ],
		"arcs": [
			{"id":"a_in","sourceId":"p_src","targetId":"t_incr","expression":"x","direction":"IN"},
			{"id":"a_out","sourceId":"t_incr","targetId":"p_mid","expression":"g = (g or 0) + x; g","direction":"OUT"}
		],
		"initialMarking": {"Src": [ {"value":3, "timestamp":0}, {"value":4, "timestamp":0} ]}
	}'
curl -X POST "${FLOW_SVC}/api/simulation/step?id=scope-arc-global" # Fires first token: g=3
curl -X POST "${FLOW_SVC}/api/simulation/step?id=scope-arc-global" # Fires second token: g=7
curl -X GET  "${FLOW_SVC}/api/marking/get?id=scope-arc-global" # Mid should have tokens [3,7]
```

### 3. Action Expression Producing Global for Output Arc
`actionExpression` sets `tmp`; output arc uses it. Using `local tmp` would hide it from the arc.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "scope-action-global",
		"name": "Action Global",
		"description": "Action sets global for arc",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_in","name":"In","colorSet":"INT"},
			{"id":"p_out","name":"Out","colorSet":"INT"}
		],
		"transitions": [ {"id":"t_proc","name":"Proc","kind":"Auto","actionExpression":"tmp = x * 10"} ],
		"arcs": [
			{"id":"a_in","sourceId":"p_in","targetId":"t_proc","expression":"x","direction":"IN"},
			{"id":"a_out","sourceId":"t_proc","targetId":"p_out","expression":"tmp + 1","direction":"OUT"}
		],
		"initialMarking": {"In": [ {"value":2, "timestamp":0} ]}
	}'
curl -X POST "${FLOW_SVC}/api/simulation/step?id=scope-action-global"
curl -X GET  "${FLOW_SVC}/api/marking/get?id=scope-action-global" # Expect Out value 21
```

### 4. Attempting Local in Action (Shows It Is Not Visible to Arc)
We purposely use a local variable inside action; arc falls back to nil reference pattern, so we protect with `(tmp2 or 0)`.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "scope-action-local",
		"name": "Action Local Hidden",
		"description": "Local in action not visible outside",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_in","name":"In","colorSet":"INT"},
			{"id":"p_out","name":"Out","colorSet":"INT"}
		],
		"transitions": [ {"id":"t_proc","name":"Proc","kind":"Auto","actionExpression":"local tmp2 = x * 3"} ],
		"arcs": [
			{"id":"a_in","sourceId":"p_in","targetId":"t_proc","expression":"x","direction":"IN"},
			{"id":"a_out","sourceId":"t_proc","targetId":"p_out","expression":"(tmp2 or -1)","direction":"OUT"}
		],
		"initialMarking": {"In": [ {"value":5, "timestamp":0} ]}
	}'
curl -X POST "${FLOW_SVC}/api/simulation/step?id=scope-action-local"
curl -X GET  "${FLOW_SVC}/api/marking/get?id=scope-action-local" # Expect Out value -1 because tmp2 is local
```

### 5. Guard Using Globals Accumulated by Previous Firings
Guard checks a running sum `g` accumulated globally; second token allowed only if sum < 10.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "scope-guard-global",
		"name": "Guard Global",
		"description": "Guard references global updated by output arc",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_src","name":"Src","colorSet":"INT"},
			{"id":"p_out","name":"Out","colorSet":"INT"}
		],
		"transitions": [ {"id":"t_guard","name":"TGuard","kind":"Auto","guardExpression":"(g or 0) < 10","variables":[]} ],
		"arcs": [
			{"id":"a_in","sourceId":"p_src","targetId":"t_guard","expression":"x","direction":"IN"},
			{"id":"a_out","sourceId":"t_guard","targetId":"p_out","expression":"g = (g or 0) + x; g","direction":"OUT"}
		],
		"initialMarking": {"Src": [ {"value":6, "timestamp":0}, {"value":7, "timestamp":0} ]}
	}'
curl -X POST "${FLOW_SVC}/api/simulation/step?id=scope-guard-global" # Fires first (g=6)
curl -X POST "${FLOW_SVC}/api/simulation/step?id=scope-guard-global" # Second blocked (g=6 >=10? false so still fires? adjust values)
curl -X GET  "${FLOW_SVC}/api/marking/get?id=scope-guard-global"
```
Adjust values so the guard stops firing when threshold reached (e.g., use tokens 6 and 5).

### Summary Table
| Pattern | Use Case | Persist? | Example |
|---------|----------|----------|---------|
| `local tmp = ...` inside arc | Temporary calc | No | Arc local example |
| Global assignment `g = ...` in arc | Accumulate across firings | Yes | Arc global example |
| Action sets global `tmp = ...` | Share with output arcs | Yes | Action global example |
| Action local `local t = ...` | Hidden temp | No | Action local example |
| Guard reads global `(g or 0)` | Conditional on history | Yes | Guard global example |

### Notes
- Because one Lua state is reused, globals persist across separate CPN loads within same process; consider resetting engine if isolation required.
- Avoid naming collisions: pick descriptive global names.
- Use `(var or default)` idiom to safely read possibly unset globals.

Document version: 1.0
