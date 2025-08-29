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
		"initialMarking": {"In": [ {"value": 1 ,"timestamp":0, "count": 100} ]},
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


### Performance

Primary bottleneck sources (likely): per-token Lua eval (setup + string compile), per-token Go<->Lua value conversion, naive multiset expansion into 10k heap objects, repeated enabled-binding recomputation, and sequential firing (one token per transition pass).

Improve performance (apply top items first):

Compressed multiset
Keep (value,timestamp,count) aggregates; don’t materialize 10k *models.Token.
Update marking ops to decrement/increment counts.
Fire transition: if arc expression is variable passthrough or simple linear map (e.g. x, x+const, x.."suffix"), apply once and transfer count.
Arc/action precompilation
On CPN load, precompile each guard/arc/action into a lua.Function (DoString once).
Evaluation: push already-built function, set only changed globals, call.
Minimize Lua state setup
Reuse one L per transition firing; avoid re-running setup for each arc expression.
Cache lua tables representing immutable token values (numbers/strings skip conversion entirely; write directly as lua.LNumber/lua.LString).
Fast path for primitive expressions
Detect patterns: x, x + <number>, x - <number>, x .. "literal".
Execute in Go without Lua; only fall back to Lua for complex constructs (math.random, function calls, table ops).
Batch firing
If transition input arc multiplicity count > 1 and independent (no guard using per-token differences), consume all available identical tokens in one call, produce outputs in bulk (count preserve).
Offer Engine.FireAllAuto(cpn, marking) that loops internal while auto enabled (you added execute-all; extend inside to batch per transition).
Binding search pruning
For large places, short-circuit once enough bindings chosen (only need first for Auto under current semantics).
Replace full slice copies with index references; avoid cloning tokens during binding enumeration.
Memory reuse
Object pool for EvaluationContext and temporary token slices (sync.Pool).
Avoid map allocations per firing: reuse map[string]*Token (clear entries).
Reduce conversions
Provide Evaluator.GetNumber(varName) for numeric tokens to read without converting through generic interface.
When writing back mutated globals, skip unchanged primitives.
Optional concurrency
Parallelize independent auto transitions in a step (careful with shared marking: stage diffs then apply under lock).
math.random hot path
If randomness not essential per token, generate once per batch.
Or switch to pre-filled random slice.
Guard purity enforcement
If guard doesn’t reference variable values that differ per token, evaluate once.
Profiling & metrics
Add simple timing around FireTransition and Lua eval; use go tool pprof to identify real hot spots before deeper changes.
Incremental implementation order:

(1) compressed multiset
(2) precompile expressions
(4) fast-path simple expressions
(3) shared context (micro-optimization) Then profile again.
Data model change sketch (compressed multiset): type MultiToken struct { Value interface{}; Timestamp int; Count int } Marking.Places: map[string][]*MultiToken Consumption: decrement Count; remove when 0. Production: merge with existing identical (value,timestamp) entry.

Arc eval adaptation:

If passthrough: reuse value; just adjust counts.
If deterministic pure function (no math.random, no globals), compute once.
Lua precompile sketch: type Compiled struct { fn *lua.LFunction; kind ExprKind; raw string; pure bool; simplePattern SimpleKind } Store in Arc struct / Transition.

Expect speed-up >10x for large homogeneous multisets.

Let me know if you want me to start by implementing compressed multiset or expression precompilation next.