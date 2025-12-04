# Flowa Examples

This directory contains organized examples demonstrating all Flowa features.

## Examples by Topic

Each topic has a `basic_*.flowa` and `advanced_*.flowa` example:

### Authentication & JWT (`auth/`)
- `basic_auth.flowa` - Password hashing and verification
- `advanced_auth.flowa` - Full login system with JWT tokens

### Email (`email/`)
- `basic_email.flowa` - Send simple text emails
- `advanced_email.flowa` - HTML templates and notifications

### HTTP Server (`http/`)
- `basic_http.flowa` - Simple routes and handlers
- `advanced_http.flowa` - REST API with middleware

### WebSocket (`websocket/`)
- `basic_websocket.flowa` - Echo server
- `advanced_websocket.flowa` - Chat server

### Loops (`loops/`)
- `basic_loops.flowa` - For and while loops
- `advanced_loops.flowa` - Nested loops and complex patterns

### Functions (`functions/`)
- `basic_functions.flowa` - Function definition and calls
- `advanced_functions.flowa` - Recursion and higher-order functions

### Data Structures (`data/`)
- `basic_data.flowa` - Arrays and maps
- `advanced_data.flowa` - Complex nested structures

### JSON (`json/`)
- `basic_json.flowa` - Encode and decode
- `advanced_json.flowa` - API integration

## Original Examples

The numbered examples (`01_basics.flowa` through `09_advanced_features.flowa`) provide a sequential tutorial.

## Running Examples

```bash
# Run any example
./flowa examples/auth/basic_auth.flowa

# Run HTTP server example (runs in foreground)
./flowa examples/http/basic_http.flowa

# Test with curl (in another terminal)
curl http://localhost:8080/
```

## Testing

See `examples/run_all_tests.sh` for automated testing of all examples.
