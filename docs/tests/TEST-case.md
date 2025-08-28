## Case API End-to-End Examples

This document demonstrates usage of the case management endpoints (`/api/cases/*`) with curl.

Prerequisites:
```sh
export FLOW_SVC=http://localhost:8082
```

### All in one

```sh
curl -s -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "case-demo-cpn-1",
		"name": "Case Demo CPN",
		"description": "CPN for case API demo",
		"colorSets": ["colset REAL = real;"],
		"places": [
			{"id":"p_in","name":"In","colorSet":"REAL"},
			{"id":"p_mid","name":"Mid","colorSet":"REAL"},
			{"id":"p_out","name":"Out","colorSet":"REAL"}
		],
		"transitions": [
			{"id":"t_auto","name":"Auto","kind":"Auto"},
			{"id":"t_auto1","name":"Auto1","kind":"Auto","actionExpression":"local tmp = x * (1+ math.random()); return tmp;"}
		],
		"arcs": [
			{"id":"a_in","sourceId":"p_in","targetId":"t_auto","expression":"x","direction":"IN"},
			{"id":"a_a_mid","sourceId":"t_auto","targetId":"p_mid","expression":"x + math.random()","direction":"OUT"},
			{"id":"a_mid_in","sourceId":"p_mid","targetId":"t_auto1","expression":"y","direction":"IN"},
			{"id":"a_out","sourceId":"t_auto1","targetId":"p_out","expression":"y - math.random()","direction":"OUT"}
		],
		"initialMarking": {"In": [ {"value": 1 ,"timestamp":0, "count": 10000} ]},
		"endPlaces": ["Out"]
	}'

curl -s -X POST ${FLOW_SVC}/api/cases/create \
	-H 'Content-Type: application/json' \
	-d '{
		"id":"case-212",
		"cpnId":"case-demo-cpn-1",
		"name":"Case 1",
		"description":"Demo case",
		"variables": {"priority":"high","owner":"alice"}
	}' | jq
curl -s -X POST "${FLOW_SVC}/api/cases/start?id=case-212" | jq
curl -s -X POST "${FLOW_SVC}/api/cases/executeall?id=case-212" | jq
curl -s -X POST "${FLOW_SVC}/api/cases/abort?id=case-212" | jq
curl -s -X DELETE "${FLOW_SVC}/api/cases/delete?id=case-212" | jq
```

### 1. Load a CPN to Use for Cases
We'll use a simple net with a manual transition to show firing APIs.
```sh
curl -s -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "case-demo-cpn",
		"name": "Case Demo CPN",
		"description": "CPN for case API demo",
		"colorSets": ["colset STRING = string;"],
		"places": [
			{"id":"p_in","name":"In","colorSet":"STRING"},
			{"id":"p_mid","name":"Mid","colorSet":"STRING"},
			{"id":"p_out","name":"Out","colorSet":"STRING"}
		],
		"transitions": [
			{"id":"t_auto","name":"Auto","kind":"Auto"},
			{"id":"t_manual","name":"Manual","kind":"Auto","actionExpression":"note = (note or \"\") .. \"-handled\""}
		],
		"arcs": [
			{"id":"a_in","sourceId":"p_in","targetId":"t_auto","expression":"x","direction":"IN"},
			{"id":"a_a_mid","sourceId":"t_auto","targetId":"p_mid","expression":"x .. \"-proc\"","direction":"OUT"},
			{"id":"a_mid_in","sourceId":"p_mid","targetId":"t_manual","expression":"y","direction":"IN"},
			{"id":"a_out","sourceId":"t_manual","targetId":"p_out","expression":"(note or y)","direction":"OUT"}
		],
		"initialMarking": {"In": [ {"value":"item1","timestamp":0, "count": 10} ]},
		"endPlaces": ["Out"]
	}'
```

### 2. Create a Case
```sh
curl -s -X POST ${FLOW_SVC}/api/cases/create \
	-H 'Content-Type: application/json' \
	-d '{
		"id":"case-1",
		"cpnId":"case-demo-cpn",
		"name":"Case 1",
		"description":"Demo case",
		"variables": {"priority":"high","owner":"alice"}
	}' | jq
```

### 3. Get Case
```sh
curl -s "${FLOW_SVC}/api/cases/get?id=case-1" | jq
```

### 4. Update Case Variables & Metadata
```sh
curl -s -X PUT "${FLOW_SVC}/api/cases/update?id=case-1" \
	-H 'Content-Type: application/json' \
	-d '{"variables":{"sla":"24h"},"metadata":{"source":"import"}}' | jq
```

### 5. Start Case (Initial Marking Copied)
```sh
curl -s -X POST "${FLOW_SVC}/api/cases/start?id=case-1" | jq
```

### 6. Inspect Case Marking
```sh
curl -s "${FLOW_SVC}/api/cases/marking?id=case-1" | jq
```

### 7. Execute a Step (Auto Transition Fires)
```sh
curl -s -X POST "${FLOW_SVC}/api/cases/execute?id=case-1" | jq
curl -s "${FLOW_SVC}/api/cases/marking?id=case-1" | jq  # token should be in Mid
```

### 7b. Execute All Automatic Transitions (Cascade Until Quiescent)
If remaining automatic transitions enable others, this fires them all in sequence.
```sh
curl -s -X POST "${FLOW_SVC}/api/cases/executeall?id=case-1" | jq
```

### 8. List Enabled Transitions For Case
```sh
curl -s "${FLOW_SVC}/api/cases/transitions?id=case-1" | jq
```

### 9. Fire Manual Transition (Binding Index 0)
```sh
curl -s -X POST "${FLOW_SVC}/api/cases/fire?id=case-1" \
	-H 'Content-Type: application/json' \
	-d '{"transitionId":"t_manual","bindingIndex":0}' | jq
curl -s "${FLOW_SVC}/api/cases/marking?id=case-1" | jq  # token should be in Out
```

### 10. Suspend and Resume Case
```sh
curl -s -X POST "${FLOW_SVC}/api/cases/suspend?id=case-1" | jq
curl -s -X POST "${FLOW_SVC}/api/cases/resume?id=case-1" | jq
```

### 11. Abort Case
```sh
curl -s -X POST "${FLOW_SVC}/api/cases/abort?id=case-1" | jq
```

### 12. Query Cases (Filter & Sort)
```sh
curl -s -X POST ${FLOW_SVC}/api/cases/query \
	-H 'Content-Type: application/json' \
	-d '{"filter":{"cpnId":"case-demo-cpn"},"sort":{"by":"CreatedAt","ascending":false}}' | jq
```

### 13. Statistics
```sh
curl -s "${FLOW_SVC}/api/cases/statistics" | jq
```

### 14. Delete Case (must be completed/aborted)
```sh
curl -s -X DELETE "${FLOW_SVC}/api/cases/delete?id=case-1" | jq
```

### 15. Error Examples
```sh
# Missing case id
curl -s "${FLOW_SVC}/api/cases/get" | jq

# Fire missing transition
curl -s -X POST "${FLOW_SVC}/api/cases/fire?id=case-1" -H 'Content-Type: application/json' -d '{"transitionId":"nope"}' | jq
```

### Notes
- Quoting: Because the entire JSON is in a single-quoted shell string, inner Lua string literals use escaped double quotes (e.g. `x .. \"-proc\"`) to avoid shell termination issues that caused errors like `__unm undefined`.
- `bindingIndex` defaults to 0 when only one binding; if multiple bindings, enumerate via `/api/cases/transitions`.
- `execute` performs one automatic simulation step (fires all enabled auto transitions once based on engine logic).
- `execute-all` keeps firing automatic transitions until no more are enabled.
- Aborting transitions marks case terminated; deletion only allowed after termination.

Document version: 1.0
