# Go Petri Flow

A high-performance Colored Petri Net (CPN) simulation engine implemented in Go, featuring gopher-lua for expression evaluation and a comprehensive REST API.

## Overview

Go Petri Flow is a complete implementation of Colored Petri Nets with the following key features:

- **Colored Petri Net Support**: Full implementation of CPN semantics including places, transitions, arcs, color sets, and markings
- **Lua Expression Engine**: Uses gopher-lua for guard expressions and arc expressions, providing a powerful and flexible scripting environment
- **REST API**: Complete HTTP API for loading, managing, and simulating CPNs
- **Real-time Simulation**: Support for both automatic and manual transition firing
- **Time-aware**: Global clock management and transition delays
- **Extensible**: Modular architecture with clear separation of concerns

## Architecture

The system is organized into several key packages:

### Core Components

- **`models`**: Core CPN data structures (Token, Place, Transition, Arc, CPN, etc.)
- **`expression`**: Lua-based expression evaluation engine
- **`engine`**: CPN simulation engine with transition firing logic
- **`api`**: REST API handlers and server setup

### Key Features

1. **Color Sets**: Support for integer, string, boolean, real, unit, enumerated, and product color sets
2. **Expression Evaluation**: Lua-based guard and arc expressions with variable binding
3. **Simulation Engine**: Automatic transition firing, manual transition control, and step-by-step simulation
4. **API Layer**: RESTful endpoints for all CPN operations with CORS support

## Quick Start

### Building the Server


```bash
# Clone and build (requires Go 1.25+ for json v2 support)
cd go-petri-flow
go build -tags=json1.25 -o bin/go-petri-flow ./cmd/server

# Run the server
./bin/go-petri-flow -port 8080
```

> **Note:** This project uses Go 1.25's `encoding/json` v2. Make sure you have Go 1.25 or newer installed. The `-tags=json1.25` build tag is required to enable the new JSON implementation.

### Running Tests

```bash
# Run all tests
go test ./test/ -v

# Run specific test categories
go test ./test/ -v -run "TestAPI"
go test ./test/ -v -run "TestEngine"
go test ./test/ -v -run "TestExpression"
```

## API Documentation

The server provides a comprehensive REST API for CPN operations:

### Base URL
```
http://localhost:8080/api
```

### Endpoints

#### CPN Management
- `POST /cpn/load` - Load a CPN from JSON definition
- `GET /cpn/list` - List all loaded CPNs
- `GET /cpn/get?id={cpnId}` - Get CPN details
- `DELETE /cpn/delete?id={cpnId}` - Delete a CPN
- `POST /cpn/reset?id={cpnId}` - Reset CPN to initial marking

#### Marking and State
- `GET /marking/get?id={cpnId}` - Get current marking

#### Transitions
- `GET /transitions/list?id={cpnId}` - List transitions and their status
- `POST /transitions/fire` - Manually fire a transition

#### Simulation
- `POST /simulation/step?id={cpnId}` - Perform one simulation step
- `POST /simulation/steps?id={cpnId}&steps={n}` - Perform multiple steps

#### Utility
- `GET /health` - Health check
- `GET /docs` - API documentation

## CPN JSON Format

CPNs are defined using a JSON format that includes all necessary components:

```json
{
  "id": "example-cpn",
  "name": "Example CPN",
  "description": "A simple example",
  "colorSets": [
    "colset INT = int;",
    "colset STRING = string;"
  ],
  "places": [
    {
      "id": "p1",
      "name": "Start",
      "colorSet": "INT"
    }
  ],
  "transitions": [
    {
      "id": "t1",
      "name": "Process",
      "kind": "Auto",
      "guardExpression": "x > 0",
      "variables": ["x"]
    }
  ],
  "arcs": [
    {
      "id": "a1",
      "sourceId": "p1",
      "targetId": "t1",
      "expression": "x",
      "direction": "IN"
    }
  ],
  "initialMarking": {
    "Start": [
      {
        "value": 5,
        "timestamp": 0
      }
    ]
  },
  "endPlaces": ["End"]
}
```

## Color Sets

The system supports various color set types:

### Basic Types
```lua
colset INT = int;
colset STRING = string;
colset BOOL = bool;
colset REAL = real;
colset UNIT = unit;
```

### Ranged Integers
```lua
colset SmallInt = int[1..10];
```

### Enumerated Types
```lua
colset Color = with red | green | blue;
colset Status = with pending | approved | rejected;
```

### Product Types
```lua
colset Pair = product INT * STRING;
colset Triple = product INT * STRING * BOOL;
```

### Timed Color Sets
```lua
colset TimedInt = int timed;
colset TimedString = string timed;
```

## Lua Expressions

The system uses Lua for guard and arc expressions, providing powerful scripting capabilities:

### Guard Expressions
```lua
x > 0
x > 10 and y < 20
status == "pending"
```

### Arc Expressions
```lua
x                    -- Simple variable
x + 1               -- Arithmetic
x * 2 + 5           -- Complex arithmetic
tuple(x, y, z)      -- Tuple creation
delay(x, 5)         -- Delayed token (timestamp + 5)
```

### Built-in Functions
- `tuple(...)` - Create tuples
- `delay(value, time)` - Create delayed tokens
- `type(value)` - Get value type
- `tostring(value)` - Convert to string
- `tonumber(value)` - Convert to number

## Examples

### Simple Processing CPN

See `examples/simple_cpn.json` for a basic CPN that demonstrates:
- Automatic transitions
- Guard expressions
- Arc expressions with arithmetic
- Transition delays

### Manual Approval Process

See `examples/manual_cpn.json` for a CPN that demonstrates:
- Manual transitions
- Multiple token types
- Conditional processing based on token values

## Testing

The project includes comprehensive tests covering:

- **Unit Tests**: All core components (models, expression engine, simulation engine)
- **Integration Tests**: API endpoints and full workflow testing
- **Example Tests**: Validation of example CPNs

Run tests with:
```bash
go test ./test/ -v
```

## Development

### Project Structure
```
go-petri-flow/
├── cmd/server/          # Main server application
├── internal/
│   ├── api/            # REST API handlers
│   ├── engine/         # CPN simulation engine
│   ├── expression/     # Lua expression evaluator
│   └── models/         # Core CPN data structures
├── test/               # Test files
├── examples/           # Example CPN definitions
└── bin/               # Built binaries
```

### Adding New Features

1. **New Color Sets**: Implement the `ColorSet` interface in `models/colorset.go`
2. **New Lua Functions**: Add functions in `expression/evaluator.go`
3. **New API Endpoints**: Add handlers in `api/handlers.go` and routes in `api/server.go`

## Performance

The system is designed for high performance:

- **Efficient Data Structures**: Optimized multisets and markings
- **Minimal Memory Allocation**: Careful memory management in hot paths
- **Concurrent Safe**: Thread-safe design for concurrent API access
- **Fast Expression Evaluation**: Optimized Lua state management

## Comparison with Python Implementation

This Go implementation provides several advantages over the original Python version:

1. **Performance**: Significantly faster execution due to Go's compiled nature
2. **Memory Efficiency**: Lower memory footprint and better garbage collection
3. **Concurrency**: Built-in support for concurrent operations
4. **Type Safety**: Compile-time type checking reduces runtime errors
5. **Deployment**: Single binary deployment with no runtime dependencies
6. **Lua Integration**: More efficient Lua integration with gopher-lua

## License

This project is part of the CPN simulation framework and follows the same licensing terms as the original Python implementation.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## Support

For questions and support:
- Check the API documentation at `/api/docs`
- Review the example CPNs in the `examples/` directory
- Run the test suite to understand expected behavior
- Check the health endpoint at `/api/health` for system status

