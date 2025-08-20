# API Test Coverage: Example curl Commands

```sh
export FLOW_SVC=http://localhost:8082
```
## 1. Health Check
```sh
curl -X GET {$FLOW_SVC}/api/health
```

## 2. Load a CPN Definition
```sh
curl -X POST {$FLOW_SVC}/api/cpn/load \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "test-cpn",
    "name": "Test CPN",
    "description": "A test CPN for API testing",
    "colorSets": ["colset INT = int;"],
    "colorSets": [
      "colset INT = int;",
      "colset jsonMap = map;"
    ],
    "places": [
      {"id": "p1", "name": "Place1", "colorSet": "INT"},
      {"id": "p2", "name": "Place2", "colorSet": "INT"},
      {"id": "p3", "name": "JsonPlace", "colorSet": "jsonMap"}
    ],
    "transitions": [
      {"id": "t1", "name": "Transition1", "kind": "Auto"}
    ],
    "arcs": [
      {"id": "a1", "sourceId": "p1", "targetId": "t1", "expression": "x", "direction": "IN"},
      {"id": "a2", "sourceId": "t1", "targetId": "p2", "expression": "x + 1", "direction": "OUT"},
      {"id": "a3", "sourceId": "p3", "targetId": "t1", "expression": "m", "direction": "IN"}
    ],
    "initialMarking": {
      "Place1": [{"value": 5, "timestamp": 0}],
      "JsonPlace": [{"value": {"foo": "bar", "num": 42}, "timestamp": 0}]
    }
  }'
```

## 3. List All CPNs
```sh
curl -X GET {$FLOW_SVC}/api/cpn/list
```

## 4. Get Current Marking
```sh
curl -X GET "{$FLOW_SVC}/api/marking/get?id=test-cpn"
```

## 5. List Transitions (with enabled/disabled status)
```sh
curl -X GET "{$FLOW_SVC}/api/transitions/list?id=test-cpn"
```

## 6. Fire a Transition
```sh
curl -X POST {$FLOW_SVC}/api/transitions/fire \
  -H 'Content-Type: application/json' \
  -d '{
    "cpnId": "test-cpn",
    "transitionId": "t1",
    "bindingIndex": 0
  }'
```

## 7. Simulate One Step (automatic transitions)
```sh
curl -X POST "{$FLOW_SVC}/api/simulation/step?id=test-cpn"
```

## 8. Simulate Multiple Steps
```sh
curl -X POST "{$FLOW_SVC}/api/simulation/steps?id=test-cpn&steps=10"
```

## 9. Reset CPN to Initial Marking
```sh
curl -X POST "{$FLOW_SVC}/api/cpn/reset?id=test-cpn"
```

## 10. Validate a CPN (rule violations & diagnostics)
```sh
curl -X GET "${FLOW_SVC}/api/cpn/validate?id=test-cpn"
```

# Notes
- Replace `localhost:8080` with your actual server address/port if different.
- The example payloads match those used in the Go test cases.
- For endpoints requiring a CPN to be loaded, run the "Load a CPN Definition" command first.
- After a reset, the marking reverts; re-check with the marking endpoint.
- Multiple steps stop early if no auto transitions fire or net completes.
