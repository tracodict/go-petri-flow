## Manual Transition Fire (Options A & B)

Demonstrates two ways to update context before OUT arcs:
- Option A: Mutate bound variable in ActionExpression.
- Option B: Supply external formData when firing manual transition; injected variables are visible to action & OUT arcs.

```sh
export FLOW_SVC=http://localhost:8082
```

### 1. Load Net for Option A (Action Mutates Bound Variable)

```sh
curl -X POST ${FLOW_SVC}/api/cpn/load -H 'Content-Type: application/json' -d '{
  "id": "manual-fire-a",
  "name": "Manual Fire A",
  "description": "Action mutates token",
  "colorSets": ["colset J = json;"],
  "places": [ {"id":"p_in","name":"In","colorSet":"J"}, {"id":"p_out","name":"Out","colorSet":"J"} ],
  "transitions": [ {"id":"t_review","name":"Review","kind":"Manual","variables":["item"], "actionExpression":"item.status = \"APPROVED\""} ],
  "arcs": [
    {"id":"a_in","sourceId":"p_in","targetId":"t_review","expression":"item","direction":"IN"},
    {"id":"a_out","sourceId":"t_review","targetId":"p_out","expression":"item","direction":"OUT"}
  ],
  "initialMarking": {"p_in":[{"value":{"id":1,"status":"PENDING"},"timestamp":0}]}
}'
```

Get enabled (binding index should be 0):
```sh
curl -s "${FLOW_SVC}/api/transitions/enabled?id=manual-fire-a" | jq '.'
```
Fire (no formData):
```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
  -d '{"cpnId":"manual-fire-a","transitionId":"t_review","bindingIndex":0}' | jq '.'
```
Check output marking contains status APPROVED:
```sh
curl -s "${FLOW_SVC}/api/marking/get?id=manual-fire-a" | jq '.places.p_out'
```

### 2. Load Net for Option B (formData Injected)

Action copies a form variable into token.

```sh
curl -X POST ${FLOW_SVC}/api/cpn/load -H 'Content-Type: application/json' -d '{
  "id": "manual-fire-b",
  "name": "Manual Fire B",
  "description": "formData injection",
  "jsonSchemas": [
    {"name":"TaskForm","schema":{"type":"object","properties":{"decision":{"type":"string"},"note":{"type":"string"}},"required":["decision"]}}
  ],
  "colorSets": ["colset T = json;"],
  "places": [ {"id":"p_in","name":"In","colorSet":"T"}, {"id":"p_out","name":"Out","colorSet":"T"} ],
  "transitions": [ {"id":"t_decide","name":"Decide","kind":"Manual","variables":["task"],"guardExpression":"task.state == \"OPEN\"","actionExpression":"task.decision = decision; task.note = note"} ],
  "arcs": [
    {"id":"a_in","sourceId":"p_in","targetId":"t_decide","expression":"task","direction":"IN"},
    {"id":"a_out","sourceId":"t_decide","targetId":"p_out","expression":"task","direction":"OUT"}
  ],
  "initialMarking": {"p_in":[{"value":{"id":42,"state":"OPEN"},"timestamp":0}]}
}'
```
Enabled transitions (should show t_decide):
```sh
curl -s "${FLOW_SVC}/api/transitions/enabled?id=manual-fire-b" | jq '.'
```
Fire with formData (decision + note supplied):
```sh
curl -X POST ${FLOW_SVC}/api/transitions/fire -H 'Content-Type: application/json' \
  -d '{"cpnId":"manual-fire-b","transitionId":"t_decide","bindingIndex":0,"formData":{"decision":"APPROVE","note":"All good"}}' | jq '.'
```
Verify updated token:
```sh
curl -s "${FLOW_SVC}/api/marking/get?id=manual-fire-b" | jq '.places.p_out'
```
Expect object containing fields decision:"APPROVE" and note:"All good".

### 3. Error Case: formData Overwrites Existing Variable

Providing a key matching an existing bound variable name will override it before the action executes. (Use cautiously.)

---
Document version: 1.0
