## Hierarchical CPN (Substitution Transition Semantics – CPN Tools Convention)

Status: MVP IMPLEMENTED (child spawning, input mapping, deferred output emission on child completion). This doc now uses place IDs (not names) for initialMarking and endPlaces.

Key Semantics (CPN Tools aligned):
1. Substitution Transition ("call") consumes input tokens (binding) on its socket input arcs and creates a child net instance (subpage) – one instance per firing.
2. Input Mapping: Only the consumed tokens are mapped to child port/input places according to `inputMapping` (parent variable -> child variable). Child initial marking = child net's own declared initial marking PLUS mapped tokens.
3. Child Execution: If `autoStart:true`, child instance immediately executes automatic transitions (cascading) but may leave manual transitions pending. Parent continues executing other transitions concurrently.
4. Blocking / Completion: Parent substitution transition output tokens are deferred until the child reaches its terminal marking (all required end places marked) when `propagateOnComplete:true`.
5. Output Mapping: Upon child completion, tokens present in child output port places bound to variables in `outputMapping` are transferred to parent context as new tokens bound to mapped parent variables; then parent output arc expressions (from call transition) are evaluated to produce tokens.
6. Multiple Instances: Each firing spawns a distinct child case (ID pattern: `parentCaseID:subLinkID:seq`). Multiple children may run in parallel.
7. Asynchronous: Parent net is free to fire unrelated transitions while child runs; only production of call transition outputs waits.
8. Failure / Deadlock: If child deadlocks (no enabled transitions, not terminal) the parent call remains pending (no outputs). Recovery requires model design (timeouts/escalation not in first cut).
9. Isolation: No global variable leakage; only tokens via mappings. Lua state for parent and child nets are independent evaluators.
10. First Implementation Scope: Input/output token mapping, child case lifecycle, deferred outputs, parallel child instances; no UI endpoints yet for listing children (temporary exposure via parent case metadata `children`).

```sh
export FLOW_SVC=http://localhost:8082
```

### 1. Load Child CPN
```sh
# Example payload:
curl -s -X POST ${FLOW_SVC}/api/cpn/load \
  -H 'Content-Type: application/json' \
  -d '{
    "id":"child-cpn-1",
    "name":"ChildFlow",
    "colorSets":["colset INT = int;"],
    "places":[{"id":"c_in","name":"CIn","colorSet":"INT"},{"id":"c_out","name":"COut","colorSet":"INT"}],
    "transitions":[{"id":"t_child","name":"Child","kind":"Auto","actionExpression":"y = x * 2"}],
    "arcs":[
      {"id":"ac_in","sourceId":"c_in","targetId":"t_child","expression":"x","direction":"IN"},
      {"id":"ac_out","sourceId":"t_child","targetId":"c_out","expression":"y","direction":"OUT"}
    ],
  "initialMarking":{"c_in":[{"value":5,"timestamp":0}]},
  "endPlaces":["c_out"]
  }'
```

### 2. Load Parent CPN Referencing Child
```sh
# Uses subWorkflows extension (implemented):
curl -s -X POST ${FLOW_SVC}/api/cpn/load \
  -H 'Content-Type: application/json' \
  -d '{
    "id":"parent-cpn-1",
    "name":"ParentFlow",
    "colorSets":["colset INT = int;"],
    "places":[
      {"id":"p_start","name":"Start","colorSet":"INT"},
      {"id":"p_wait","name":"Wait","colorSet":"INT"},
      {"id":"p_done","name":"Done","colorSet":"INT"}
    ],
    "transitions":[
      {"id":"t_call_child","name":"CallChild","kind":"Manual"},
      {"id":"t_finalize","name":"Finalize","kind":"Auto"}
    ],
    "arcs":[
      {"id":"ap_in","sourceId":"p_start","targetId":"t_call_child","expression":"a","direction":"IN"},
      {"id":"ap_wait_out","sourceId":"t_call_child","targetId":"p_wait","expression":"b","direction":"OUT"},
      {"id":"ap_wait_in","sourceId":"p_wait","targetId":"t_finalize","expression":"b","direction":"IN"},
      {"id":"ap_out","sourceId":"t_finalize","targetId":"p_done","expression":"b","direction":"OUT"}
    ],
  "initialMarking":{"p_start":[{"value":3,"timestamp":0}]},
  "endPlaces":["p_done"],
    "subWorkflows":[
      {
        "id":"sw1",
        "cpnId":"child-cpn-1",
        "callTransitionId":"t_call_child",
        "autoStart":true,
        "propagateOnComplete":true,
        "inputMapping":{"x":"a"},
  "outputMapping":{"y":"b"}
      }
    ]
  }'
```

Explanation of Parent Example:
- Parent consumes token bound as `a` on `ap_in` when firing `t_call_child`.
- Input mapping sends that token as `x` into child net input place (via child arc expecting `x`).
- Child doubles value (y = x * 2) producing token `y` in its output place.
- On child completion, outputMapping y->b binds `b` in parent context; the deferred output arc from `t_call_child` to `p_wait` with expression `b` materializes token `b` (value 2*a).
- `t_finalize` then moves token from `p_wait` to `p_done`.

### 3. Create & Start Parent Case
```sh
curl -s -X POST ${FLOW_SVC}/api/cases/create -H 'Content-Type: application/json' -d '{"id":"parent-case-1","cpnId":"parent-cpn-1","name":"Parent Case"}'
curl -s -X POST "${FLOW_SVC}/api/cases/start?id=parent-case-1"
```

### 4. Fire Call Transition & Monitor Child
```sh
curl -s -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' -d '{"cpnId":"parent-cpn-1","transitionId":"t_call_child","bindingIndex":0}'
# Expect engine to instantiate child case (future endpoint might list it)
```

### 5. Execute Automatic Steps Until Child Completion & Propagation
```sh
# Run automatic transitions (child autoStart + any parent autos)
curl -s -X POST "${FLOW_SVC}/api/cases/executeall?id=parent-case-1"

# Verify propagated token now exists in p_wait (value should be 2 * original a)
curl -s "${FLOW_SVC}/api/marking/get?id=parent-cpn-1" | jq '.places.p_wait'
```

### 6. Retrieve Parent & Child State
```sh
curl -s "${FLOW_SVC}/api/cases/get?id=parent-case-1"
# Potential future: /api/cases/children?parentId=parent-case-1
```

### 7. (Alternate) Direct CPN Workflow (No Case Manager)
This variant uses only CPN-level endpoints to demonstrate hierarchical call transition firing (child still managed internally for deferred outputs):
```sh
# 7.1 List CPNs
curl -s ${FLOW_SVC}/api/cpn/list | jq '.cpns[] | {id,name}'

# 7.2 Inspect parent CPN definition (shows subWorkflows)
curl -s "${FLOW_SVC}/api/cpn/get?id=parent-cpn-1" | jq '.subWorkflows'

# 7.3 View initial marking
curl -s "${FLOW_SVC}/api/marking/get?id=parent-cpn-1" | jq

# 7.4 Check enabled transitions (expect t_call_child manual)
curl -s "${FLOW_SVC}/api/transitions/list?id=parent-cpn-1" | jq '.[] | {id,name,enabled,kind}'

# 7.5 Fire call transition (spawns child, defers outputs)
curl -s -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
  -d '{"cpnId":"parent-cpn-1","transitionId":"t_call_child","bindingIndex":0}' | jq

# 7.6 Execute automatic transitions to quiescence (child autoStart + any parent autos)
curl -s -X POST "${FLOW_SVC}/api/simulation/step?id=parent-cpn-1" | jq '.transitionsFired'
# or repeatedly until 0 fired or end marking reached.

# 7.7 (Optional) Multiple-step batch
curl -s -X POST "${FLOW_SVC}/api/simulation/steps?id=parent-cpn-1&steps=20" | jq '.transitionsFired'

# 7.8 Marking after child completion (expect token in p_wait due to propagation, then finalize)
curl -s "${FLOW_SVC}/api/marking/get?id=parent-cpn-1" | jq

# 7.9 If finalize still pending (manual chain), list transitions again
curl -s "${FLOW_SVC}/api/transitions/list?id=parent-cpn-1" | jq '.[] | {id,enabled}'

# 7.10 Fire finalize if needed
curl -s -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
  -d '{"cpnId":"parent-cpn-1","transitionId":"t_finalize","bindingIndex":0}' | jq

# 7.11 Final marking (token in p_done)
curl -s "${FLOW_SVC}/api/marking/get?id=parent-cpn-1" | jq
```

### Status
- MVP hierarchy working: spawning child, autoStart, deferred parent outputs, output mapping to parent on child completion. Missing pieces: multi-variable mapping into child token places (currently via case variables), explicit child listing endpoint, timeout/error escalation.

### Next Steps to Realize Feature
1. Extend models: add SubWorkflowLink struct & slice on CPN.
2. Parser & JSON roundtrip for subWorkflows.
3. Case manager: on firing callTransitionId create child case (naming convention parentID:linkID:seq).
4. Mapping: before child start, map parent vars/tokens per inputMapping; after child completion, map outputs.
5. Propagation & completion: block parent transition completion until child done if propagateOnComplete.
6. Add endpoints to list child cases for a parent.

### Future Implementation Checklist
1. Add `SubWorkflowLink` model and slice on `CPN`.
2. Extend JSON parser / serializer to handle `subWorkflows`.
3. Track parent/child case relations; add metadata `children` list.
4. Intercept `FireTransition` for call transitions: create child case; if `autoStart`, run auto cascade.
5. Defer output arcs until child completion (`propagateOnComplete`).
6. Apply input/output mappings; merge with child initial marking.
7. Auto cascade and manual continuation for child cases via existing case APIs.
8. Add optional endpoint `/api/cases/children?id=parentCase` (later).

### Warning
Model currently maps values through case variables (not injecting tokens into child input place yet if absent). Adjust child net to read variables in expressions if needed until direct token injection enhancement lands.
