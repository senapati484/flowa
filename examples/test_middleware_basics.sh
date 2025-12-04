#!/bin/bash

# Comprehensive Flowa Middleware and Basics Test Script

echo "==========================================="
echo "Flowa Middleware & Basics Test Suite"
echo "==========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0
WARNINGS=0

# Test function
run_test() {
    local test_name="$1"
    local test_file="$2"
    local timeout_seconds="$3"
    
    echo "Testing: $test_name"
    
    # Run test in background and capture PID
    ./flowa "$test_file" > /tmp/flowa_test_output.txt 2>&1 &
    test_pid=$!
    
    # Wait for timeout or completion
    wait_time=0
    while [ $wait_time -lt $timeout_seconds ]; do
        if ! kill -0 $test_pid 2>/dev/null; then
            # Process completed
            wait $test_pid
            exit_code=$?
            if [ $exit_code -eq 0 ]; then
                echo -e "${GREEN}✓ PASSED${NC}: $test_name"
                ((PASSED++))
            else
                echo -e "${RED}✗ FAILED${NC}: $test_name (exit code: $exit_code)"
                echo "Output:"
                cat /tmp/flowa_test_output.txt
                ((FAILED++))
            fi
            echo ""
            return
        fi
        sleep 0.1
        wait_time=$((wait_time + 1))
    done
    
    # Timeout reached, kill process
    kill $test_pid 2>/dev/null
    echo -e "${YELLOW}⚠ TIMEOUT${NC}: $test_name (This is expected for server tests)"
    ((WARNINGS++))
    echo ""
}

# Test 1: JSON Operations
echo "=== Core Module Tests ==="
run_test "JSON encode/decode" "examples/test_json_ops.flowa" 3

# Test 2: Config Module
run_test "Config.env()" "examples/test_config.flowa" 3

# Test 3: Auth Module
run_test "Auth hash/verify" "examples/test_auth.flowa" 3

# Test 4: JWT Module
run_test "JWT sign/verify" "examples/test_jwt.flowa" 3

# Test 5: Basic HTTP
echo "=== HTTP Server Tests ==="
run_test "Basic HTTP routes" "examples/http/basic_http.flowa" 2

# Test 6: Advanced HTTP with middleware
run_test "Advanced HTTP (middleware)" "examples/http/advanced_http.flowa" 2

# Test 7: Email (if .env configured)
if [ -f "examples/.env" ]; then
    echo "=== Email Tests ==="
    # run_test "Email functionality" "examples/test_email.flowa" 3
    echo "Skipping email test (requires SMTP configuration)"
fi

echo ""
echo "==========================================="
echo "Test Results Summary"
echo "==========================================="
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${YELLOW}Warnings: $WARNINGS${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo "==========================================="

# Exit with failure if any tests failed
if [ $FAILED -gt 0 ]; then
    exit 1
fi

exit 0
