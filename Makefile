# Flowa Language Makefile

# Variables
BINARY_NAME=flowa
BUILD_DIR=.
INSTALL_PATH=/usr/local/bin

# Build the binary
build:
	@echo "Building Flowa..."
	go build -o $(BINARY_NAME) ./cmd/flowa
	@echo "✓ Build complete: $(BINARY_NAME)"

# Install globally (requires sudo on Linux/macOS)
install: build
	@echo "Installing Flowa to $(INSTALL_PATH)..."
	@sudo cp $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✓ Flowa installed successfully!"
	@echo "You can now use 'flowa' from anywhere"

# Uninstall
uninstall:
	@echo "Uninstalling Flowa..."
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✓ Flowa uninstalled"

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@echo "✓ Clean complete"

# Run examples
examples: build
	@echo "Running examples..."
	@./$(BINARY_NAME) examples/hello.flowa
	@echo "---"
	@./$(BINARY_NAME) examples/pipeline.flowa
	@echo "---"
	@./$(BINARY_NAME) examples/fibonacci.flowa
	@echo "✓ All examples passed"

# Development: build and run a file
run: build
	@./$(BINARY_NAME) $(FILE)

.PHONY: build install uninstall test clean examples run
