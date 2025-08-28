## JSON Color Set Tests (curl Examples)

End-to-end API scenarios covering untyped `json`, schema-bound `json<Schema>`, validation failures, guards, transformations, arrays, and legacy alias.

```sh
export FLOW_SVC=http://localhost:8082
```

### 1. Load CPN with Untyped json Color Set
Loads a net whose place accepts any JSON value.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "json-cpn-1",
		"name": "Untyped JSON",
		"description": "Any JSON accepted",
		"colorSets": ["colset Meta = json;"],
		"places": [ { "id":"p_meta", "name":"MetaIn", "colorSet":"Meta" } ],
		"transitions": [ { "id":"t_passthru", "name":"Pass", "kind":"Auto" } ],
		"arcs": [
			{ "id":"a_in",  "sourceId":"p_meta", "targetId":"t_passthru", "expression":"x", "direction":"IN"},
			{ "id":"a_out", "sourceId":"t_passthru", "targetId":"p_meta", "expression":"x", "direction":"OUT"}
		],
		"initialMarking": { "p_meta": [ { "value": { "k":"v", "n": 1 }, "timestamp": 0 } ] }
	}'
curl -X GET "${FLOW_SVC}/api/cpn/get?id=json-cpn-1"
```
Expect: load succeeds; token present.

### 2. Load CPN with Schema-Bound json<OrderSchema>
Defines a JSON Schema and binds it to a color set.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "json-cpn-2",
		"name": "Schema Orders",
		"description": "Orders with schema",
		"jsonSchemas": [
			{ "name": "OrderSchema", "schema": {"type":"object","required":["id","total"],"properties": {"id":{"type":"string"},"total":{"type":"number"}} } },
			{ "name": "OrderSchema1", "schema": {"type":"object","required":["id","total"],"properties": {"id":{"type":"string"},"total":{"type":"number"}} } }
		],
		"colorSets": ["colset Order = json<OrderSchema>;"],
		"places": [ { "id":"p_orders", "name":"Orders", "colorSet":"Order" } ],
		"transitions": [ { "id":"t_hold", "name":"Hold", "kind":"Auto" } ],
		"arcs": [ { "id":"a_in", "sourceId":"p_orders", "targetId":"t_hold", "expression":"order", "direction":"IN" },
							 { "id":"a_out","sourceId":"t_hold","targetId":"p_orders","expression":"order","direction":"OUT" } ],
		"initialMarking": { "p_orders": [ { "value": { "id":"A1", "total": 10.5 }, "timestamp":0 } ] }
	}'
curl -X GET "${FLOW_SVC}/api/cpn/get?id=json-cpn-2"
```
Expect: load succeeds.

### 3. Reject Invalid Initial Token (Schema Missing Required Field)
Missing `total` should cause load error.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "json-cpn-3-bad",
		"name": "Bad Order",
		"description": "Invalid order token",
		"jsonSchemas": [
			{ "name": "OrderSchema", "schema": {"type":"object","required":["id","total"],"properties": {"id":{"type":"string"},"total":{"type":"number"}} } }
		],
		"colorSets": ["colset Order = json<OrderSchema>;"],
		"places": [ { "id":"p_orders", "name":"Orders", "colorSet":"Order" } ],
		"transitions": [],
		"arcs": [],
		"initialMarking": { "p_orders": [ { "value": { "id":"A1" }, "timestamp":0 } ] }
	}'
```
Expect: HTTP 400 with schema validation error message.

### 4. Transformation via Lua on Output Arc
Adds a flag when total > 100; token must still validate.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "json-cpn-4",
		"name": "Transform Order",
		"description": "Add flag in action",
		"jsonSchemas": [
			{ "name": "OrderSchema", "schema": {"type":"object","required":["id","total"],"properties": {"id":{"type":"string"},"total":{"type":"number"}} } }
		],
		"colorSets": ["colset Order = json<OrderSchema>;"],
		"places": [ { "id":"p_in", "name":"In", "colorSet":"Order" }, { "id":"p_out", "name":"Out", "colorSet":"Order" } ],
		"transitions": [ { "id":"t_proc", "name":"Proc", "kind":"Auto" } ],
		"arcs": [
			{ "id":"a_in",  "sourceId":"p_in",  "targetId":"t_proc", "expression":"order", "direction":"IN" },
			{ "id":"a_out", "sourceId":"t_proc", "targetId":"p_out", "expression":"local o = order; if o.total > 100 then o.flag=\"BIG\" end; return o", "direction":"OUT" }
		],
		"initialMarking": { "p_in": [ { "value": { "id":"B1", "total": 150 }, "timestamp":0 } ] }
	}'
curl -X POST ${FLOW_SVC}/api/simulation/step?id=json-cpn-4
curl -X GET  "${FLOW_SVC}/api/marking/get?id=json-cpn-4" # Expect token with flag = BIG in p_out
```

### 5. Deprecated Alias 'map'
Current behavior: treat `map` same as untyped `json` (no warning yet).
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "json-cpn-5",
		"name": "Legacy Alias",
		"description": "Legacy map alias",
		"colorSets": ["colset Legacy = map;"],
		"places": [ { "id":"p_map", "name":"Map", "colorSet":"Legacy" } ],
		"transitions": [],
		"arcs": [],
		"initialMarking": { "p_map": [ { "value": { "x":1 }, "timestamp":0 } ] }
	}'
```
Expect: load succeeds.

### 6. Guard Accessing Nested JSON
Transition only fires when `order.total > 50`.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "json-cpn-6",
		"name": "Guarded Order",
		"description": "Guard uses nested field",
		"jsonSchemas": [ { "name": "OrderSchema", "schema": {"type":"object","required":["id","total"],"properties": {"id":{"type":"string"},"total":{"type":"number"}} } } ],
		"colorSets": ["colset Order = json<OrderSchema>;"],
		"places": [ { "id":"p_src", "name":"Src", "colorSet":"Order" }, { "id":"p_sink", "name":"Sink", "colorSet":"Order" } ],
		"transitions": [ { "id":"t_pass", "name":"Pass", "kind":"Auto", "guardExpression":"order.total > 50", "variables":["order"] } ],
		"arcs": [
			{ "id":"a_in","sourceId":"p_src","targetId":"t_pass","expression":"order","direction":"IN" },
			{ "id":"a_out","sourceId":"t_pass","targetId":"p_sink","expression":"order","direction":"OUT" }
		],
		"initialMarking": { "p_src": [ { "value": { "id":"G1", "total": 75 }, "timestamp":0 } ] }
	}'
curl -X POST ${FLOW_SVC}/api/simulation/step?id=json-cpn-6
curl -X GET  "${FLOW_SVC}/api/marking/get?id=json-cpn-6" # Expect token moved to p_sink
```

### 7. Unknown Schema Reference
Expect failure due to `UnknownSchema` not defined in `jsonSchemas`.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "json-cpn-7-bad",
		"name": "Unknown Schema",
		"description": "Bad schema ref",
		"colorSets": ["colset X = json<UnknownSchema>;"],
		"places": [],
		"transitions": [],
		"arcs": [],
		"initialMarking": {}
	}'
```
Expect: HTTP 400 unknown schema error.

### 8. Output Arc Producing Array (Accepted vs Rejected)
Untyped json accepts array token:
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "json-cpn-8a",
		"name": "Array OK",
		"description": "Array output for untyped json",
		"colorSets": ["colset J = json;"],
		"places": [ { "id":"p_in", "name":"In", "colorSet":"J" }, { "id":"p_out","name":"Out","colorSet":"J" } ],
		"transitions": [ { "id":"t_arr","name":"Arr","kind":"Auto" } ],
		"arcs": [ { "id":"a_in","sourceId":"p_in","targetId":"t_arr","expression":"j","direction":"IN" },
							 { "id":"a_out","sourceId":"t_arr","targetId":"p_out","expression":"return {1,2,3}","direction":"OUT" } ],
		"initialMarking": { "p_in": [ { "value": { "a":1 }, "timestamp":0 } ] }
	}'
curl -X POST ${FLOW_SVC}/api/simulation/step?id=json-cpn-8a
curl -X GET  "${FLOW_SVC}/api/marking/get?id=json-cpn-8a" # Expect array token in p_out
```
Same attempt with schema-bound set should fail (array not matching object schema):
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "json-cpn-8b-bad",
		"name": "Array Bad",
		"description": "Array violates schema",
		"jsonSchemas": [ { "name":"OrderSchema", "schema": {"type":"object","required":["id","total"],"properties": {"id":{"type":"string"},"total":{"type":"number"}} } } ],
		"colorSets": ["colset Order = json<OrderSchema>;"],
		"places": [ { "id":"p_src","name":"Src","colorSet":"Order" }, { "id":"p_dst","name":"Dst","colorSet":"Order" } ],
		"transitions": [ { "id":"t_make","name":"Make","kind":"Auto" } ],
		"arcs": [ { "id":"a_in","sourceId":"p_src","targetId":"t_make","expression":"order","direction":"IN" },
							 { "id":"a_out","sourceId":"t_make","targetId":"p_dst","expression":"return {1,2,3}","direction":"OUT" } ],
		"initialMarking": { "p_src": [ { "value": { "id":"Z1", "total": 1 }, "timestamp":0 } ] }
	}'
```
Expect: error when producing invalid array (future enhancement may defer to runtime validation).

### 9. Validation Endpoint (Future Token Schema Violations)
After a failed load (case 3) or future runtime schema violation, you can inspect:
```sh
curl -X GET "${FLOW_SVC}/api/cpn/validate?id=json-cpn-2"
```
Current implementation may not yet list `token_schema_violation`; planned for enhancement.

---
Document version: 1.0
