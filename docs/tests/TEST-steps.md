## Simulation Steps API Test (curl Examples)

End-to-end scenarios exercising `/api/simulation/step` and `/api/simulation/steps` with a net that has multiple initial tokens producing a processing pipeline.

```sh
export FLOW_SVC=http://localhost:8082
```

### Network: Parallel Tokens Through Two Sequential Auto Transitions
Definition: Two integer tokens enter `Stage1`; each pass through `Inc` then `Double` producing results in `Done`.

Pipeline logic:
- Inc: out = x + 1
- Double: out = x * 2
So each initial token v becomes (v+1)*2.

#### 1. Load the CPN
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "steps-cpn-1",
		"name": "Steps Pipeline",
		"description": "Two-stage pipeline with multiple initial tokens",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_in","name":"Stage1","colorSet":"INT"},
			{"id":"p_mid","name":"Stage2","colorSet":"INT"},
			{"id":"p_done","name":"Done","colorSet":"INT"}
		],
		"transitions": [
			{"id":"t_inc","name":"Inc","kind":"Auto"},
			{"id":"t_double","name":"Double","kind":"Auto"}
		],
		"arcs": [
			{"id":"a1","sourceId":"p_in","targetId":"t_inc","expression":"x","direction":"IN"},
			{"id":"a2","sourceId":"t_inc","targetId":"p_mid","expression":"x+1","direction":"OUT"},
			{"id":"a3","sourceId":"p_mid","targetId":"t_double","expression":"x","direction":"IN"},
			{"id":"a4","sourceId":"t_double","targetId":"p_done","expression":"x*2","direction":"OUT"}
		],
		"initialMarking": {"Stage1": [
			{"value":3,"timestamp":0},
			{"value":5,"timestamp":0},
			{"value":10,"timestamp":0}
		]}
	}'
```

#### 2. Baseline Marking & Transitions
```sh
curl -X GET "${FLOW_SVC}/api/marking/get?id=steps-cpn-1"
curl -X GET "${FLOW_SVC}/api/transitions/list?id=steps-cpn-1"
```
Expect three enabled bindings for `Inc` and none for `Double` initially.

#### 3. Single Simulation Step
Each simulation step now fires at most one binding for each currently enabled automatic transition (layered). Newly enabled transitions created by those firings are deferred to the next step. After the first step in this pipeline all three Inc firings occur, producing three intermediate tokens in Stage2. Double does not fire until the next step.
```sh
curl -X POST "${FLOW_SVC}/api/simulation/step?id=steps-cpn-1"
curl -X GET "${FLOW_SVC}/api/marking/get?id=steps-cpn-1"
```
Expected outcome after first step: `Stage1` empty, `Stage2` has 3 tokens 4,6,11; `Done` empty.

#### 4. Second Step
Run another step to move Stage2 tokens to Done (Double fires for each token):
```sh
curl -X POST "${FLOW_SVC}/api/simulation/step?id=steps-cpn-1"
curl -X GET "${FLOW_SVC}/api/marking/get?id=steps-cpn-1"
```
Expect `Done` to now contain values {8,12,22}.

#### 5. Multiple Steps in One Call
Reset and then execute multi-step.
```sh
curl -X POST "${FLOW_SVC}/api/cpn/reset?id=steps-cpn-1"
curl -X POST "${FLOW_SVC}/api/simulation/steps?id=steps-cpn-1&steps=5"
curl -X GET "${FLOW_SVC}/api/marking/get?id=steps-cpn-1"
```
Expect final Done marking to match set {8,12,22}. Response from multi-step includes `transitionsFired` count.

#### 6. Validate After Completion
```sh
curl -X GET "${FLOW_SVC}/api/cpn/validate?id=steps-cpn-1"
```
No structural violations expected.

### Extended Scenario: Partial Consumption with Manual Stop
Use manual variant for second stage to show difference.
```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "steps-cpn-2",
		"name": "Manual Second Stage",
		"description": "Second stage manual to illustrate step boundaries",
		"colorSets": ["colset INT = int;"],
		"places": [
			{"id":"p_in","name":"Stage1","colorSet":"INT"},
			{"id":"p_mid","name":"Stage2","colorSet":"INT"},
			{"id":"p_done","name":"Done","colorSet":"INT"}
		],
		"transitions": [
			{"id":"t_inc","name":"Inc","kind":"Auto"},
			{"id":"t_double","name":"Double","kind":"Manual"}
		],
		"arcs": [
			{"id":"a1","sourceId":"p_in","targetId":"t_inc","expression":"x","direction":"IN"},
			{"id":"a2","sourceId":"t_inc","targetId":"p_mid","expression":"x+1","direction":"OUT"},
			{"id":"a3","sourceId":"p_mid","targetId":"t_double","expression":"x","direction":"IN"},
			{"id":"a4","sourceId":"t_double","targetId":"p_done","expression":"x*2","direction":"OUT"}
		],
		"initialMarking": {"Stage1": [ {"value":2,"timestamp":0}, {"value":7,"timestamp":0} ]}
	}'
```
Run one automatic simulation step (only Inc fires):
```sh
curl -X POST "${FLOW_SVC}/api/simulation/step?id=steps-cpn-2"
curl -X GET "${FLOW_SVC}/api/marking/get?id=steps-cpn-2"
curl -X GET "${FLOW_SVC}/api/transitions/list?id=steps-cpn-2" # Double should appear manual & enabled
```
Manually fire `Double` for first binding (repeat for remaining if desired):
```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
	-d '{"cpnId":"steps-cpn-2","transitionId":"t_double","bindingIndex":0}'
curl -X GET "${FLOW_SVC}/api/marking/get?id=steps-cpn-2"
```

### Notes
- Record actual engine step behavior (layered vs full chain) to keep docs accurate.
- Multi-step endpoint stops early if no transitions fire in an iteration or net completes.
- Reset returns to initial multiset of tokens enabling reproducible runs.

---


### Invidual token in multiset

Arc multiplicity today just loops the same arc logic N times; it doesn’t bind “individual slot” variables for those N consumed tokens. So you cannot distinguish them inside a single firing via something like x1, x2, x3 automatically.

Current options:

1. If you need to refer to each token separately, use multiple parallel input arcs (each with its own variable name: x1, x2, x3) instead of one arc with multiplicity=3. That gives you distinct bindings you can use in guard/output expressions.
2. If tokens are indistinguishable (all the same value) and you only need the count, multiplicity is fine—just consume them; you don’t get per‑token variables.
3. If you need to process a batch collectively, first aggregate tokens into a product/tuple or JSON array token via repeated firings of a “collector” transition, then operate on that single structured token.
4. For future enhancement you could add (not implemented): automatic variables like x_1…x_n or an iteration index (e.g. _i) during multiplicity expansion.