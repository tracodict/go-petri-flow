## Multiset API Test Coverage (curl Examples)

These scenarios illustrate operations that rely on place multisets: duplicate token values, counting, consumption ordering via expressions, and ensuring correct removal semantics.

```sh
export FLOW_SVC=http://localhost:8082
```

### 1. Load CPN With Duplicate Integer Tokens
Loads two identical INT tokens so multiset holds multiplicity 2 under same value key. You can now also use the `count` shorthand instead of listing duplicates.
Verbose duplicate listing:
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "ms-cpn-1",
		"name": "Multiset Duplicates",
		"description": "Two identical int tokens",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_src","name":"Src","colorSet":"INT"},
			{"id":"p_dst","name":"Dst","colorSet":"INT"}
		],
		"transitions": [
			{"id":"t_move","name":"Move","kind":"Auto"}
		],
		"arcs": [
			{"id":"a_in","sourceId":"p_src","targetId":"t_move","expression":"x","direction":"IN"},
			{"id":"a_out","sourceId":"t_move","targetId":"p_dst","expression":"x","direction":"OUT"}
		],
		"initialMarking": {"Src": [ {"value":5,"timestamp":0}, {"value":5,"timestamp":0} ]}
	}'
```

Equivalent using count shorthand:
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "ms-cpn-1c",
		"name": "Multiset Duplicates (Count)",
		"description": "Two identical int tokens via count",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_src","name":"Src","colorSet":"INT"},
			{"id":"p_dst","name":"Dst","colorSet":"INT"}
		],
		"transitions": [ {"id":"t_move","name":"Move","kind":"Auto"} ],
		"arcs": [
			{"id":"a_in","sourceId":"p_src","targetId":"t_move","expression":"x","direction":"IN"},
			{"id":"a_out","sourceId":"t_move","targetId":"p_dst","expression":"x","direction":"OUT"}
		],
		"initialMarking": {"Src": [ {"value":5,"timestamp":0,"count":2} ]}
	}'
```
Inspect marking (expect two tokens with same value at source):
```sh
curl -X GET "${FLOW_SVC}/api/marking/get?id=ms-cpn-1"
```

### 2. Fire Once – Multiplicity Decreases
```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
	-d '{"cpnId":"ms-cpn-1","transitionId":"t_move","bindingIndex":0}'
curl -X GET "${FLOW_SVC}/api/marking/get?id=ms-cpn-1"
```
Expect Src still has one token value 5; Dst has one token value 5.

### 3. Fire Second Time – Source Empties
```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
	-d '{"cpnId":"ms-cpn-1","transitionId":"t_move","bindingIndex":0}'
curl -X GET "${FLOW_SVC}/api/marking/get?id=ms-cpn-1"
```
Expect Src empty; Dst has two tokens (value 5 multiplicity 2).

### 4. Load Mixed Tokens & Selective Consumption
Demonstrates leaving other values intact when consuming a specific one.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "ms-cpn-2",
		"name": "Selective Consumption",
		"description": "Consume only value 7 via expression",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_mix","name":"Mix","colorSet":"INT"},
			{"id":"p_out","name":"Out","colorSet":"INT"}
		],
		"transitions": [ {"id":"t_take7","name":"Take7","kind":"Auto"} ],
		"arcs": [
			{"id":"a_in","sourceId":"p_mix","targetId":"t_take7","expression":"7","direction":"IN"},
			{"id":"a_out","sourceId":"t_take7","targetId":"p_out","expression":"7","direction":"OUT"}
		],
	"initialMarking": {"Mix": [ {"value":7,"timestamp":0,"count":2}, {"value":3,"timestamp":0} ]}
	}'
```
Fire once (should remove one of the 7s):
```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
	-d '{"cpnId":"ms-cpn-2","transitionId":"t_take7","bindingIndex":0}'
curl -X GET "${FLOW_SVC}/api/marking/get?id=ms-cpn-2"
```
Expect Mix still contains one 7 and one 3.

### 5. Multiple Firings Until Value Exhausted
```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
	-d '{"cpnId":"ms-cpn-2","transitionId":"t_take7","bindingIndex":0}'
curl -X GET "${FLOW_SVC}/api/marking/get?id=ms-cpn-2"
```
After second fire Mix should no longer have value 7; only 3 remains.

### 6. Product Color Set Multiplicity
Demonstrates multiset with product (tuple) tokens.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "ms-cpn-3",
		"name": "Product Tokens",
		"description": "Duplicate tuple tokens",
		"colorSets": [
			"colset INT = int;",
			"colset STR = string;",
			"colset Pair = product INT * STR;"
		],
		"places": [
			{"id":"p_pair","name":"PairPlace","colorSet":"Pair"},
			{"id":"p_sink","name":"Sink","colorSet":"Pair"}
		],
		"transitions": [ {"id":"t_pass","name":"Pass","kind":"Auto"} ],
		"arcs": [
			{"id":"a_in","sourceId":"p_pair","targetId":"t_pass","expression":"tuple(x,y)","direction":"IN"},
			{"id":"a_out","sourceId":"t_pass","targetId":"p_sink","expression":"tuple(x,y)","direction":"OUT"}
		],
		"initialMarking": {"PairPlace": [
			{"value": {"first":1,"second":"A"}, "timestamp":0, "count":2},
			{"value": {"first":2,"second":"B"}, "timestamp":0}
		]}
	}'
```
Fire once (expect one of the (1,"A") tuples moved):
```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
	-d '{"cpnId":"ms-cpn-3","transitionId":"t_pass","bindingIndex":0}'
curl -X GET "${FLOW_SVC}/api/marking/get?id=ms-cpn-3"
```

### 7. Validation of Multiset Nets
```sh
curl -X GET "${FLOW_SVC}/api/cpn/validate?id=ms-cpn-1"
curl -X GET "${FLOW_SVC}/api/cpn/validate?id=ms-cpn-2"
curl -X GET "${FLOW_SVC}/api/cpn/validate?id=ms-cpn-3"
```

### Notes
- Engine removes a single token instance per firing based on expression result value.
- Tokens with identical value are distinguished only by pointer identity and timestamp; listing shows multiplicities.
- Product color set token JSON includes field names `first`, `second` (internal representation in current implementation may differ; adapt expression if changed).
 - NEW: `multiplicity` on an arc repeats that arc's consumption (IN) or production (OUT) expression that many times in a single firing. Example: an input arc with `multiplicity:3` and expression `x` will consume three tokens matching `x` (each iteration re-evaluates expression; use simple variable for identical values). An output arc with `multiplicity:5` will emit five identical evaluations (use timestamp / delay logic inside expression if you need variation—note each evaluation re-runs the Lua expression so you can incorporate counters external to net if desired).

### Large Token Sets Best Practice
For very large counts (e.g. thousands):
- Prefer the `count` field to keep JSON compact: `{ "value": 7, "timestamp": 0, "count": 1000 }`.
- Split extremely large counts across multiple entries if you conceptually model batches with different timestamps.
- If values must differ programmatically, consider a small bootstrap transition that expands a single seed token into many (fan-out) rather than enumerating all in initial JSON.

### Consuming Multiple Tokens Without Listing Each
Current arc expression model consumes one token per input place per binding. To remove multiple identical tokens in one firing you can:
1. Use multiple parallel input arcs from the same place each binding a variable (e.g. two arcs x and y) if color set semantics allow distinct selection (duplicates may bind to same value but distinct token instances).
2. Model an accumulator transition that fires repeatedly (via auto kind) until a guard (e.g., remaining count < N) disables it.
3. (Future extension idea) Introduce an aggregate input syntax like `x[3]` to mean 3 tokens of expression `x`. This is not implemented yet; prefer repeated firings or multiple arcs.

`multiplicity` now provides atomic batch consumption/production. A future enhancement could allow dynamic multiplicity via an expression (currently it is a static integer in the arc definition).

### 8. Input Arc Multiplicity (Single Firing Consumes Multiple Tokens)
Consumes three identical tokens in one firing using an input arc with `multiplicity:3`.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "ms-cpn-4",
		"name": "Input Multiplicity",
		"description": "Consume 3 tokens in one firing",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_src","name":"Src","colorSet":"INT"},
			{"id":"p_taken","name":"Taken","colorSet":"INT"}
		],
		"transitions": [ {"id":"t_take3","name":"Take3","kind":"Auto"} ],
		"arcs": [
			{"id":"a_in","sourceId":"p_src","targetId":"t_take3","expression":"7","direction":"IN","multiplicity":3},
			{"id":"a_out","sourceId":"t_take3","targetId":"p_taken","expression":"7","direction":"OUT"}
		],
		"initialMarking": {"Src": [ {"value":7,"timestamp":0,"count":5} ]}
	}'
curl -X GET "${FLOW_SVC}/api/marking/get?id=ms-cpn-4" # Before firing: Src has 5 x 7
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
	-d '{"cpnId":"ms-cpn-4","transitionId":"t_take3","bindingIndex":0}'
curl -X GET "${FLOW_SVC}/api/marking/get?id=ms-cpn-4" # After: Src has 2 x 7, Taken has 3 x 7
```

### 9. Output Arc Multiplicity (Single Firing Produces Multiple Tokens)
Produces four identical tokens in one firing using an output arc with `multiplicity:4`.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "ms-cpn-5",
		"name": "Output Multiplicity",
		"description": "Produce 4 tokens in one firing",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_src","name":"Src","colorSet":"INT"},
			{"id":"p_prod","name":"Prod","colorSet":"INT"}
		],
		"transitions": [ {"id":"t_emit4","name":"Emit4","kind":"Auto"} ],
		"arcs": [
			{"id":"a_in","sourceId":"p_src","targetId":"t_emit4","expression":"x","direction":"IN"},
			{"id":"a_out","sourceId":"t_emit4","targetId":"p_prod","expression":"x*10","direction":"OUT","multiplicity":4}
		],
		"initialMarking": {"Src": [ {"value":2,"timestamp":0} ]}
	}'
curl -X GET "${FLOW_SVC}/api/marking/get?id=ms-cpn-5" # Before firing: Src has one 2
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
	-d '{"cpnId":"ms-cpn-5","transitionId":"t_emit4","bindingIndex":0}'
curl -X GET "${FLOW_SVC}/api/marking/get?id=ms-cpn-5" # After: Prod has 4 x 20
```

### 10. Combined Input & Output Multiplicity
You can combine both to convert N inputs to M outputs atomically (e.g., consume 2, produce 5) by setting multiplicity on respective arcs.

---
Document version: 1.1
