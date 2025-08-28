## Manual Transition Tests (curl Examples)

This document demonstrates loading and retrieving manual transitions that declare UI schemas. A manual transition should expose two additional properties:

	formSchema   (reference to a schema name declared in `jsonSchemas`)
	layoutSchema (reference to a schema name declared in `jsonSchemas`)

These schemas are supplied at workflow (CPN) load time inside the `jsonSchemas` array. The transition then references them by name. Round‑trip retrieval via `/api/cpn/get` should include `formSchema` and `layoutSchema` in the transition object (backend support required).

Environment variable for convenience:
```sh
export FLOW_SVC=http://localhost:8082
```

### 1. Load CPN With Single Manual Transition + Form/Layout Schemas

Defines two JSON Schemas: one for the user form data (`UserFormSchema`) and one describing layout (`UserLayoutSchema`). The manual transition `t_review` references both.

```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "manual-cpn-1",
		"name": "Manual Review Flow",
		"description": "Single manual step with form & layout",
		"jsonSchemas": [
			{ "name": "UserFormSchema", "schema": {
					"type": "object",
					"required": ["firstName","lastName","age"],
					"properties": {
						"firstName": {"type":"string"},
						"lastName":  {"type":"string"},
						"age":       {"type":"integer","minimum":0}
					}
				}
			},
			{ "name": "UserLayoutSchema", "schema": {
					"type": "object",
					"properties": {
						"sections": {"type":"array","items": {"type":"object","properties": {"title":{"type":"string"},"fields":{"type":"array","items":{"type":"string"}}},"required":["title","fields"]}}
					},
					"required": ["sections"]
				}
			}
		],
		"colorSets": ["colset USER = json<UserFormSchema>;"],
		"places": [
			{ "id":"p_in",  "name":"In",  "colorSet":"USER" },
			{ "id":"p_out", "name":"Out", "colorSet":"USER" }
		],
		"transitions": [
			{ "id":"t_review", "name":"Review", "kind":"Manual", "formSchema":"UserFormSchema", "layoutSchema":"UserLayoutSchema" }
		],
		"arcs": [
			{ "id":"a_in",  "sourceId":"p_in",  "targetId":"t_review", "expression":"user", "direction":"IN" },
			{ "id":"a_out", "sourceId":"t_review", "targetId":"p_out", "expression":"user", "direction":"OUT" }
		],
		"initialMarking": { "p_in": [ { "value": { "firstName":"Ada","lastName":"Lovelace","age": 36 }, "timestamp":0 } ] }
	}'

# Retrieve definition (expect transition includes formSchema & layoutSchema)
curl -X GET "${FLOW_SVC}/api/cpn/get?id=manual-cpn-1" | jq '.transitions[] | select(.id=="t_review")'
```

Expected (schematic) snippet:
```json
{
	"id": "t_review",
	"name": "Review",
	"kind": "Manual",
	"formSchema": "UserFormSchema",
	"layoutSchema": "UserLayoutSchema"
}
```

### 2. Multiple Manual Transitions Reusing Schemas

Two manual transitions share the same form schema but have distinct layout schemas.

```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "manual-cpn-2",
		"name": "Multi Manual",
		"description": "Two manual steps with layouts",
		"jsonSchemas": [
			{ "name": "BaseForm",   "schema": {"type":"object","properties":{"fieldA":{"type":"string"}},"required":["fieldA"]} },
			{ "name": "LayoutOne",  "schema": {"type":"object","properties":{"panels":{"type":"array"}},"required":["panels"]} },
			{ "name": "LayoutTwo",  "schema": {"type":"object","properties":{"tabs":{"type":"array"}},  "required":["tabs"]} }
		],
		"colorSets": ["colset DATA = json<BaseForm>;"],
		"places": [ { "id":"p0","name":"Start","colorSet":"DATA" }, { "id":"p1","name":"Mid","colorSet":"DATA" }, { "id":"p2","name":"End","colorSet":"DATA" } ],
		"transitions": [
			{ "id":"t_step1", "name":"Step1", "kind":"Manual", "formSchema":"BaseForm", "layoutSchema":"LayoutOne" },
			{ "id":"t_step2", "name":"Step2", "kind":"Manual", "formSchema":"BaseForm", "layoutSchema":"LayoutTwo" }
		],
		"arcs": [
			{ "id":"a0", "sourceId":"p0", "targetId":"t_step1", "expression":"d", "direction":"IN" },
			{ "id":"a1", "sourceId":"t_step1", "targetId":"p1", "expression":"d", "direction":"OUT" },
			{ "id":"a2", "sourceId":"p1", "targetId":"t_step2", "expression":"d", "direction":"IN" },
			{ "id":"a3", "sourceId":"t_step2", "targetId":"p2", "expression":"d", "direction":"OUT" }
		],
		"initialMarking": { "p0": [ { "value": { "fieldA":"hello" }, "timestamp":0 } ] }
	}'

curl -X GET "${FLOW_SVC}/api/cpn/get?id=manual-cpn-2" | jq '.transitions[] | {id,name,formSchema,layoutSchema}'
```

### 3. Error: Unknown formSchema Reference

Attempt to load a manual transition referencing a non-existent schema name. Expect HTTP 400 (backend must validate).

```sh
curl -X POST ${FLOW_SVC}/api/cpn/load \
	-H 'Content-Type: application/json' \
	-d '{
		"id": "manual-cpn-3-bad",
		"name": "Bad Manual",
		"description": "Unknown form schema ref",
		"jsonSchemas": [ { "name": "KnownSchema", "schema": {"type":"object"} } ],
		"colorSets": ["colset X = json<KnownSchema>;"],
		"places": [ { "id":"p1","name":"P1","colorSet":"X" } ],
		"transitions": [ { "id":"t_bad","name":"Bad","kind":"Manual","formSchema":"MissingSchema","layoutSchema":"KnownSchema" } ],
		"arcs": [],
		"initialMarking": {}
	}'
```

Expected: Error mentioning unknown schema reference for `formSchema`.

### 4. Work Item Creation After Case Start (Future)

Once case endpoints create work items for manual transitions, you can (after loading e.g. `manual-cpn-1`):

```sh
# (Example – adjust if case creation endpoint differs)
curl -X POST ${FLOW_SVC}/api/case/start -d '{"id":"case-1","cpnId":"manual-cpn-1"}' -H 'Content-Type: application/json'
curl -X GET  ${FLOW_SVC}/api/workitems/by-case?caseId=case-1 | jq '.workItems[] | {id,transitionId,data}'
```

Expect: Work item linked to transition `t_review`, front-end can fetch schemas by name from the original CPN definition.

---
Document version: 1.0

