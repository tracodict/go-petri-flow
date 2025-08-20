## Multiset API Test Coverage (curl Examples)

These scenarios illustrate operations that rely on place multisets: duplicate token values, counting, consumption ordering via expressions, and ensuring correct removal semantics.

```sh
export FLOW_SVC=http://localhost:8082
```

### 1. Load CPN With Duplicate Integer Tokens
Loads two identical INT tokens so multiset holds multiplicity 2 under same value key.
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
		"initialMarking": {"Mix": [ {"value":7,"timestamp":0}, {"value":3,"timestamp":0}, {"value":7,"timestamp":0} ]}
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
			{"value": {"first":1,"second":"A"}, "timestamp":0},
			{"value": {"first":1,"second":"A"}, "timestamp":0},
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

---
Document version: 1.0
