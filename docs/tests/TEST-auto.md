## Automatic Transition Action (Lua Script) API Tests

These curl examples demonstrate defining and running an `actionExpression` on an automatic transition. The action executes once per firing after inputs are consumed (and delay applied) but before outputs are produced. Any value returned is ignored; use side effects like computing derived values for output arcs via global variables or updating counters (state kept only inside the Lua VM lifecycle—no persistence layer is implemented).

```sh
export FLOW_SVC=http://localhost:8082
```

### 1. Load CPN With Action Computing Derived Output
Action stores an intermediate result in a Lua global `tmp` used by output arc expression.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "auto-act-1",
		"name": "Auto Action Demo",
		"description": "Auto transition with actionExpression",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_in","name":"In","colorSet":"INT"},
			{"id":"p_out","name":"Out","colorSet":"INT"}
		],
		"transitions": [
			{"id":"t_proc","name":"Proc","kind":"Auto","actionExpression":"tmp = x * 10"}
		],
		"arcs": [
			{"id":"a_in","sourceId":"p_in","targetId":"t_proc","expression":"x","direction":"IN"},
			{"id":"a_out","sourceId":"t_proc","targetId":"p_out","expression":"tmp + 1","direction":"OUT"}
		],
		"initialMarking": {"In": [ {"value":3, "timestamp":0} ]}
	}'
```

Fire automatically via step (layered) or multi-step:
```sh
curl -X POST "${FLOW_SVC}/api/simulation/step?id=auto-act-1"
curl -X GET  "${FLOW_SVC}/api/marking/get?id=auto-act-1" # Expect Out token value 31
```

### 2. Action With Transition Delay
Demonstrates delay + action; action can reference `x_timestamp` (timestamp of consumed token variable x) and `global_clock` (advanced after delay).
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "auto-act-2",
		"name": "Action Delay",
		"description": "Action executed after delay",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_in","name":"In","colorSet":"INT"},
			{"id":"p_out","name":"Out","colorSet":"INT"}
		],
		"transitions": [
			{"id":"t_wait","name":"WaitAndCompute","kind":"Auto","transitionDelay":4,"actionExpression":"tmp = (x * 2) + global_clock"}
		],
		"arcs": [
			{"id":"a_in","sourceId":"p_in","targetId":"t_wait","expression":"x","direction":"IN"},
			{"id":"a_out","sourceId":"t_wait","targetId":"p_out","expression":"tmp","direction":"OUT"}
		],
		"initialMarking": {"In": [ {"value":5, "timestamp":0} ]}
	}'
curl -X POST "${FLOW_SVC}/api/simulation/step?id=auto-act-2"
curl -X GET  "${FLOW_SVC}/api/marking/get?id=auto-act-2" # Expect Out value = (5*2)+4 = 14
```

### 3. Multiple Inputs and Action Combining Them
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "auto-act-3",
		"name": "Combine",
		"description": "Combine two inputs in action",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_a","name":"A","colorSet":"INT"},
			{"id":"p_b","name":"B","colorSet":"INT"},
			{"id":"p_out","name":"Out","colorSet":"INT"}
		],
		"transitions": [
			{"id":"t_add","name":"Add","kind":"Auto","actionExpression":"sum = a + b"}
		],
		"arcs": [
			{"id":"a_in1","sourceId":"p_a","targetId":"t_add","expression":"a","direction":"IN"},
			{"id":"a_in2","sourceId":"p_b","targetId":"t_add","expression":"b","direction":"IN"},
			{"id":"a_out","sourceId":"t_add","targetId":"p_out","expression":"sum","direction":"OUT"}
		],
		"initialMarking": {"A": [ {"value":2, "timestamp":0} ], "B": [ {"value":7, "timestamp":0} ]}
	}'
curl -X POST "${FLOW_SVC}/api/simulation/step?id=auto-act-3"
curl -X GET  "${FLOW_SVC}/api/marking/get?id=auto-act-3" # Expect Out value 9
```

### 4. Validation
```sh
curl -X GET "${FLOW_SVC}/api/cpn/validate?id=auto-act-1"
curl -X GET "${FLOW_SVC}/api/cpn/validate?id=auto-act-2"
curl -X GET "${FLOW_SVC}/api/cpn/validate?id=auto-act-3"
```

### Notes
- `actionExpression` runs once per firing after input consumption & delay.
- Use it to compute intermediate globals used by output arc expressions; avoid heavy side-effects (no persistence hooks provided).
- Variables from input arcs are available (e.g. x, a, b) along with their timestamps (`x_timestamp`).
- Result value (if any) is ignored unless you store it in a global or reuse via subsequent arc expressions.
- Errors in action abort the firing.

Document version: 1.0
