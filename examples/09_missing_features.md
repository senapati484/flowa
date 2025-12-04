# Missing Features

The following features were requested but are not currently implemented in the Flowa VM:

## 1. HTTP Server
- **Status**: Not Implemented
- **Missing**: `http.listen`, `http.handle`, `http.server`
- **Current Support**: Only `http.get` (Client) is supported.

## 2. Websockets
- **Status**: Not Implemented
- **Missing**: `ws` module
- **Current Support**: None.

## 3. Email
- **Status**: Not Implemented
- **Missing**: `email` or `smtp` module
- **Current Support**: None.

## Recommendation
To add these features, the VM's `createBuiltins` function in `pkg/vm/vm.go` needs to be extended with Go's `net/http` server, `github.com/gorilla/websocket`, and `net/smtp` libraries.
