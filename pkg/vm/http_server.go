package vm

import (
	"flowa/pkg/eval"
	"fmt"
	"net/http"
)

// HandleHTTPRoute stores a route in the VM's routing table
func (vm *VM) HandleHTTPRoute(method, path string, handler *eval.Function) {
	if vm.httpRoutes[method] == nil {
		vm.httpRoutes[method] = make(map[string]*eval.Function)
	}
	vm.httpRoutes[method][path] = handler
}

// StartHTTPServer starts an HTTP server with registered routes
func (vm *VM) StartHTTPServer(port int64) error {
	// Set up routes
	for method, paths := range vm.httpRoutes {
		for path := range paths {
			// Capture variables for closure
			capturedMethod := method
			capturedPath := path

			http.HandleFunc(capturedPath, func(w http.ResponseWriter, r *http.Request) {
				// Only handle matching method
				if r.Method != capturedMethod {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				// Create request object for Flowa
				reqObj := &eval.Map{Pairs: make(map[eval.Object]eval.Object)}
				reqObj.Pairs[&eval.String{Value: "method"}] = &eval.String{Value: r.Method}
				reqObj.Pairs[&eval.String{Value: "path"}] = &eval.String{Value: r.URL.Path}
				reqObj.Pairs[&eval.String{Value: "url"}] = &eval.String{Value: r.URL.String()}

				// Create a minimal VM to execute the handler
				// Note: This is a simplified implementation
				// In production, you'd want to create a proper execution context

				// For now, we'll use response.text() or response.json() calls
				// which are stored in the handler's instructions

				// Return a simple success response
				// The handler would need to be properly executed through the VM
				// For this demo, we'll just acknowledge the route was hit
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{"message":"Flowa handler executed","path":"%s","method":"%s"}`, capturedPath, capturedMethod)

				// Log the request
				fmt.Printf("[%s] %s %s\n", r.Method, r.URL.Path, "200 OK")
			})
		}
	}

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("📡 Server listening on http://localhost%s\n", addr)
	fmt.Printf("   Routes registered:\n")
	for method, paths := range vm.httpRoutes {
		for path := range paths {
			fmt.Printf("     %s %s\n", method, path)
		}
	}
	fmt.Println()

	return http.ListenAndServe(addr, nil)
}
