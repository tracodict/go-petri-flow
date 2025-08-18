## JSON Color Set Tests

### 1. Load CPN with untyped json color set
Definition snippet:
```
"colorSets": [
	"colset Meta = json;"
],
"places": [ { "id":"p1", "name":"MetaIn", "colorSet":"Meta" } ],
"initialMarking": { "MetaIn": [ { "value": { "k":"v", "n": 1 }, "timestamp": 0 } ] }
```
Expected: load succeeds, token accepted.

### 2. Load CPN with schema bound json<OrderSchema>
```
"jsonSchemas": [
	{ "name": "OrderSchema", "schema": { "type": "object", "required": ["id","total"], "properties": { "id": {"type":"string"}, "total": {"type":"number"} } } }
],
"colorSets": [ "colset Order = json<OrderSchema>;" ],
"places": [ { "id":"p2", "name":"Orders", "colorSet":"Order" } ],
"initialMarking": { "Orders": [ { "value": { "id":"A1", "total": 10.5 }, "timestamp":0 } ] }
```
Expected: load succeeds.

### 3. Reject invalid initial token (missing required)
Modify initial token to `{ "id":"A1" }` (no total). Expect: parser error referencing schema.

### 4. Transformation via Lua on output arc
Input place: Orders schema as above. Output arc expression:
```
local o = order
if o.total > 100 then o.flag = "BIG" end
return o
```
Expected: token still valid (schema allows extra properties by default). Add assertion order.flag present when total>100.

### 5. Deprecated alias 'map'
`colset Legacy = map;` should work but produce deprecation notice (future enhancement). For now just accepts.

### 6. Guard accessing nested JSON
Guard: `order.total > 50` enabling transition only when true.

### 7. json<UnknownSchema>
Expect: load fails with unknown json schema error.

### 8. Output arc producing array
`return {1,2,3}` for color set `json` accepted; for `json<OrderSchema>` rejected (schema mismatch).

### 9. Validation endpoint
After loading invalid token (case 3), /api/cpn/validate lists `token_schema_violation` (future enhancement once integrated).
