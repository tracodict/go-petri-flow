## Timed Net API Test Coverage (curl Examples)

```sh
export FLOW_SVC=http://localhost:8082
```

### 1. Load Timed CPN (Transition Delay)
Loads a net where transition `t_delay` has `transitionDelay=5`.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "timed-cpn-1",
		"name": "Timed Delay Example",
		"description": "Single transition delay of 5",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_in","name":"In","colorSet":"INT"},
			{"id":"p_out","name":"Out","colorSet":"INT"}
		],
		"transitions": [
			{"id":"t_delay","name":"DelayT","kind":"Auto","transitionDelay":5}
		],
		"arcs": [
			{"id":"a_in","sourceId":"p_in","targetId":"t_delay","expression":"x","direction":"IN"},
			{"id":"a_out","sourceId":"t_delay","targetId":"p_out","expression":"x","direction":"OUT"}
		],
		"initialMarking": {"In": [{"value": 42, "timestamp": 0}]}
	}'
```

### 2. List Transitions & Marking Before Firing
```sh
curl -X GET "${FLOW_SVC}/api/transitions/list?id=timed-cpn-1"
curl -X GET "${FLOW_SVC}/api/marking/get?id=timed-cpn-1"
```

### 3. Fire Delayed Transition
```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire \
	-H 'Content-Type: application/json' \
	-d '{"cpnId":"timed-cpn-1","transitionId":"t_delay","bindingIndex":0}'
```
Expect response globalClock increased to 5 and token moved to Out.

### 4. Net With Output Arc Token Delay (No Transition Delay)
Output arc expression introduces future token using `delay` field.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "timed-cpn-2",
		"name": "Arc Delay Example",
		"description": "Arc expression delays token by 7",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p1","name":"P1","colorSet":"INT"},
			{"id":"p2","name":"P2","colorSet":"INT"}
		],
		"transitions": [
			{"id":"t1","name":"T1","kind":"Auto","transitionDelay":0}
		],
		"arcs": [
			{"id":"a1","sourceId":"p1","targetId":"t1","expression":"x","direction":"IN"},
			{"id":"a2","sourceId":"t1","targetId":"p2","expression":"return { value = x, delay = 7 }","direction":"OUT"}
		],
		"initialMarking": {"P1": [{"value": 1, "timestamp": 0}]}
	}'
```

Fire and inspect marking (global clock should stay 0, produced token timestamp 7):
```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
	-d '{"cpnId":"timed-cpn-2","transitionId":"t1","bindingIndex":0}'
curl -X GET "${FLOW_SVC}/api/marking/get?id=timed-cpn-2"
```

### 5. Advance Simulation Step (Automatic Transitions Chain)
Load chained delays: T1 delay 3 then T2 delay 2.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "timed-cpn-3",
		"name": "Chained Delays",
		"description": "Two transitions with cumulative delays",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p1","name":"P1","colorSet":"INT"},
			{"id":"p2","name":"P2","colorSet":"INT"},
			{"id":"p3","name":"P3","colorSet":"INT"}
		],
		"transitions": [
			{"id":"t1","name":"T1","kind":"Auto","transitionDelay":3},
			{"id":"t2","name":"T2","kind":"Auto","transitionDelay":2}
		],
		"arcs": [
			{"id":"a1","sourceId":"p1","targetId":"t1","expression":"x","direction":"IN"},
			{"id":"a2","sourceId":"t1","targetId":"p2","expression":"x+1","direction":"OUT"},
			{"id":"a3","sourceId":"p2","targetId":"t2","expression":"x","direction":"IN"},
			{"id":"a4","sourceId":"t2","targetId":"p3","expression":"x+1","direction":"OUT"}
		],
		"initialMarking": {"P1": [{"value":1,"timestamp":0}]}
	}'
```
Run a first simulation step (layered semantics: only T1 fires producing token in P2 at time 3):
```sh
curl -X POST "${FLOW_SVC}/api/simulation/step?id=timed-cpn-3"
curl -X GET "${FLOW_SVC}/api/marking/get?id=timed-cpn-3" # Expect token in P2 timestamp 3, none in P3
```
Run second step to fire T2 (advances to time 5, token moved to P3):
```sh
curl -X POST "${FLOW_SVC}/api/simulation/step?id=timed-cpn-3"
curl -X GET "${FLOW_SVC}/api/marking/get?id=timed-cpn-3"
```

### 6. Multiple Steps API (steps query)
```sh
curl -X POST "${FLOW_SVC}/api/simulation/steps?id=timed-cpn-3&steps=5"
```

### 7. Validate Timed Net
```sh
curl -X GET "${FLOW_SVC}/api/cpn/validate?id=timed-cpn-3"
```

### 8. Manual Transition With Delay (Not Auto-Fired)
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "timed-cpn-4",
		"name": "Manual Delay",
		"description": "Manual transition with delay",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p1","name":"P1","colorSet":"INT"},
			{"id":"p2","name":"P2","colorSet":"INT"}
		],
		"transitions": [
			{"id":"t_manual","name":"TManual","kind":"Manual","transitionDelay":4}
		],
		"arcs": [
			{"id":"a1","sourceId":"p1","targetId":"t_manual","expression":"x","direction":"IN"},
			{"id":"a2","sourceId":"t_manual","targetId":"p2","expression":"x","direction":"OUT"}
		],
		"initialMarking": {"P1": [{"value":7,"timestamp":0}]}
	}'
```
Check auto step (should not fire manual):
```sh
curl -X POST "${FLOW_SVC}/api/simulation/step?id=timed-cpn-4"
```
Fire manually then inspect marking:
```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
	-d '{"cpnId":"timed-cpn-4","transitionId":"t_manual","bindingIndex":0}'
curl -X GET "${FLOW_SVC}/api/marking/get?id=timed-cpn-4"
```

### 9. Output Arc Delay + Guard Time Blocking Example
Token produced for future time; until clock advances, guarded transition depending on it stays disabled.
```sh
# Load producer net
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "timed-cpn-5",
		"name": "Guard Time Block",
		"description": "Future token blocks guard until time",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_src","name":"Src","colorSet":"INT"},
			{"id":"p_mid","name":"Mid","colorSet":"INT"},
			{"id":"p_sink","name":"Sink","colorSet":"INT"}
		],
		"transitions": [
			{"id":"t_prod","name":"Prod","kind":"Auto","transitionDelay":0},
			{"id":"t_cons","name":"Cons","kind":"Auto","transitionDelay":0,"guardExpression":"x > 0","variables":["x"]}
		],
		"arcs": [
			{"id":"a1","sourceId":"p_src","targetId":"t_prod","expression":"x","direction":"IN"},
			{"id":"a2","sourceId":"t_prod","targetId":"p_mid","expression":"return { value = x, delay = 6 }","direction":"OUT"},
			{"id":"a3","sourceId":"p_mid","targetId":"t_cons","expression":"x","direction":"IN"},
			{"id":"a4","sourceId":"t_cons","targetId":"p_sink","expression":"x","direction":"OUT"}
		],
		"initialMarking": {"Src": [{"value":5,"timestamp":0}]}
	}'
```
Produce future token:
```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
	-d '{"cpnId":"timed-cpn-5","transitionId":"t_prod","bindingIndex":0}'
curl -X GET "${FLOW_SVC}/api/transitions/list?id=timed-cpn-5" # t_cons should be disabled until clock advances
```
Run simulation steps until consumer fires (depends on engine clock advancement strategy):
```sh
curl -X POST "${FLOW_SVC}/api/simulation/steps?id=timed-cpn-5&steps=10"
curl -X GET "${FLOW_SVC}/api/marking/get?id=timed-cpn-5"
```

### 10. Validate Any Timed CPN
```sh
curl -X GET "${FLOW_SVC}/api/cpn/validate?id=timed-cpn-5"
```

### 11. Timed Color Set (TimedInt) Production & Consumption
Defines an explicit timed integer color set; tokens carry timestamps but transition has no delay. We manually create a future token via arc delay and then consume it.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "timed-cpn-6",
		"name": "TimedInt Example",
		"description": "Explicit timed int color set",
		"colorSets": ["colset TimedInt = int timed;"],
		"places": [
			{"id":"p_src","name":"Src","colorSet":"TimedInt"},
			{"id":"p_mid","name":"Mid","colorSet":"TimedInt"},
			{"id":"p_out","name":"Out","colorSet":"TimedInt"}
		],
		"transitions": [
			{"id":"t_emit","name":"Emit","kind":"Auto","transitionDelay":0},
			{"id":"t_consume","name":"Consume","kind":"Auto","transitionDelay":0,"guardExpression":"x > 0","variables":["x"]}
		],
		"arcs": [
			{"id":"a1","sourceId":"p_src","targetId":"t_emit","expression":"x","direction":"IN"},
			{"id":"a2","sourceId":"t_emit","targetId":"p_mid","expression":"return { value = x+1, delay = 4 }","direction":"OUT"},
			{"id":"a3","sourceId":"p_mid","targetId":"t_consume","expression":"x","direction":"IN"},
			{"id":"a4","sourceId":"t_consume","targetId":"p_out","expression":"x","direction":"OUT"}
		],
		"initialMarking": {"Src": [{"value":5,"timestamp":0}]}
	}'
```
Fire producer and inspect state (token in Mid has future timestamp 4):
```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
	-d '{"cpnId":"timed-cpn-6","transitionId":"t_emit","bindingIndex":0}'
curl -X GET "${FLOW_SVC}/api/marking/get?id=timed-cpn-6"
```
Advance simulation steps until consumer fires:
```sh
curl -X POST "${FLOW_SVC}/api/simulation/steps?id=timed-cpn-6&steps=10"
curl -X GET "${FLOW_SVC}/api/marking/get?id=timed-cpn-6"
```

### 12. TimedString Color Set With Transition Delay
Combines timed string color set and transition delay. Delay advances global clock; produced token inherits that time.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "timed-cpn-7",
		"name": "TimedString Delay",
		"description": "Timed string with transition delay",
		"colorSets": ["colset TimedString = string timed;"],
		"places": [
			{"id":"ps","name":"Src","colorSet":"TimedString"},
			{"id":"pd","name":"Dst","colorSet":"TimedString"}
		],
		"transitions": [
			{"id":"t_delay_str","name":"DelayStr","kind":"Auto","transitionDelay":3}
		],
		"arcs": [
			{"id":"as1","sourceId":"ps","targetId":"t_delay_str","expression":"s","direction":"IN"},
			{"id":"as2","sourceId":"t_delay_str","targetId":"pd","expression":"s .. \"_done\"","direction":"OUT"}
		],
		"initialMarking": {"Src": [{"value":"job","timestamp":0}]}
	}'
```
Fire delayed transition and inspect new timestamp (expected global clock 3, token timestamp >=3):
```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
	-d '{"cpnId":"timed-cpn-7","transitionId":"t_delay_str","bindingIndex":0}'
curl -X GET "${FLOW_SVC}/api/marking/get?id=timed-cpn-7"
```

### 13. Validate Timed Color Set Nets
```sh
curl -X GET "${FLOW_SVC}/api/cpn/validate?id=timed-cpn-6"
curl -X GET "${FLOW_SVC}/api/cpn/validate?id=timed-cpn-7"
```

### Notes
- Engine currently advances global clock on firing using transition delay; future token timestamps may require additional stepping to become consumable.
- Only one token produced per output arc evaluation; complex multi-token lists are not supported.
- Guard evaluation occurs only when token timestamps are <= global clock.

### Run Timed Tests (Go)
```sh
go test ./test -count=1 -run Timed
```

---
Document version: 2.1
