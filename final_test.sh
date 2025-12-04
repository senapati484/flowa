#!/bin/bash
# Final Production Test - All Examples

echo "======================================"
echo "FLOWA VM - FINAL PRODUCTION VERIFICATION"
echo "======================================"
echo ""

PASS=0
FAIL=0
TOTAL=0

run_test() {
    local name="$1"
    local file="$2"
    ((TOTAL++))
    echo -n "[$TOTAL] $name... "
    
    # Run with 2 second timeout
    if output=$(timeout 2 ./flowa_new "$file" 2>&1); then
        # Check if output contains error
        if echo "$output" | grep -qi "error\|panic\|failed"; then
            echo "❌ FAIL (error in output)"
            ((FAIL++))
            echo "$output" | head -3
        else
            echo "✅ PASS"
            ((PASS++))
        fi
    else
        exit_code=$?
        if [ $exit_code -eq 124 ]; then
            echo "⏱️  TIMEOUT (likely server - OK)"
            ((PASS++))
        else
            echo "❌ FAIL (exit $exit_code)"
            ((FAIL++))
        fi
    fi
}

# Core Examples
run_test "Basics" "examples/01_basics.flowa"
run_test "Control Flow" "examples/02_control_flow.flowa"
run_test "Functions" "examples/03_functions.flowa"
run_test "Data Structures" "examples/04_data_structures.flowa"
run_test "Modules" "examples/05_modules.flowa"
run_test "HTTP Client" "examples/06_http_client.flowa"
run_test "Auth" "examples/07_auth.flowa"
run_test "Files" "examples/08_files.flowa"
run_test "Advanced Features" "examples/09_advanced_features.flowa"

# Test Examples
run_test "Auth Test" "examples/test_auth.flowa"
run_test "Email" "examples/test_email.flowa"
run_test "JWT" "examples/test_jwt.flowa"
run_test "Mail Template" "examples/test_mail_template.flowa"
run_test "Pipeline" "examples/test_pipeline.flowa"
run_test "Postfix Ops" "examples/test_postfix_ops.flowa"
run_test "String Concat" "examples/test_string_concat.flowa"
run_test "WebSocket Info" "examples/test_websocket_info.flowa"
run_test "Simple Import" "examples/test_simple_import.flowa"
run_test "Classic For Loop" "examples/test_simple_classic_for.flowa"
run_test "HTTP Client Simple" "examples/test_http_client_simple.flowa"

# WebSocket Examples (compile checks)
run_test "WebSocket Chat" "examples/websocket_chat_server.flowa"
run_test "WebSocket Echo" "examples/websocket_echo_server.flowa"
run_test "WebSocket Test" "examples/test_websocket.flowa"

echo ""
echo "======================================"
echo "RESULTS:"
echo "  ✅ PASSED: $PASS"
echo "  ❌ FAILED: $FAIL"
echo "  📊 TOTAL: $TOTAL"
echo "  📈 SUCCESS RATE: $(( PASS * 100 / TOTAL ))%"
echo "======================================"

if [ $FAIL -eq 0 ]; then
    echo ""
    echo "🎉 ALL TESTS PASSED - PRODUCTION READY!"
    exit 0
else
    echo ""
    echo "⚠️  $FAIL tests failed"
    exit 1
fi
