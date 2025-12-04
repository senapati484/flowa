#!/bin/bash
# Comprehensive Final Test Suite for Flowa VM

echo "========================================"
echo "FLOWA VM - FINAL PRODUCTION TEST SUITE"
echo "========================================"
echo ""

PASS=0
FAIL=0

run_test() {
    local name="$1"
    local file="$2"
    echo -n "Testing $name... "
    if timeout 3 ./flowa_new "$file" >/dev/null 2>&1; then
        echo "✅ PASS"
        ((PASS++))
    else
        echo "❌ FAIL"
        ((FAIL++))
    fi
}

# Core Language Features
echo "=== CORE LANGUAGE FEATURES ==="
run_test "Basics" "examples/01_basics.flowa"
run_test "Control Flow" "examples/02_control_flow.flowa"
run_test "Functions" "examples/03_functions.flowa"
run_test "Data Structures" "examples/04_data_structures.flowa"
run_test "String Concatenation" "examples/test_string_concat.flowa"
run_test "Pipeline Operator" "examples/test_pipeline.flowa"
run_test "Postfix Operators" "examples/test_postfix_ops.flowa"
run_test "Classic For Loop" "examples/test_simple_classic_for.flowa"
echo ""

# Module System
echo "=== MODULE SYSTEM ==="
run_test "Modules" "examples/05_modules.flowa"
run_test "Import System" "examples/09_advanced_features.flowa"
run_test "Simple Import" "examples/test_simple_import.flowa"
echo ""

# HTTP & Network
echo "=== HTTP & NETWORK ==="
run_test "HTTP Client" "examples/06_http_client.flowa"  
run_test "HTTP Client Simple" "examples/test_http_client_simple.flowa"
echo ""

# Authentication & Security
echo "=== AUTH & SECURITY ==="
run_test "Auth Module" "examples/07_auth.flowa"
run_test "Auth Test" "examples/test_auth.flowa"
run_test "JWT" "examples/test_jwt.flowa"
echo ""

# File I/O
echo "=== FILE I/O ==="
run_test "File System" "examples/08_files.flowa"
echo ""

# Mail & Config
echo "=== MAIL & CONFIG ==="
run_test "Email" "examples/test_email.flowa"
run_test "Mail Template" "examples/test_mail_template.flowa"
echo ""

# WebSocket (Info)
echo "=== WEBSOCKET ==="
run_test "WebSocket Info" "examples/test_websocket_info.flowa"
echo ""

echo "========================================"
echo "FINAL RESULTS:"
echo "  ✅ PASSED: $PASS"
echo "  ❌ FAILED: $FAIL"
echo "  TOTAL: $((PASS + FAIL))"
if [ $FAIL -eq 0 ]; then
    echo ""
    echo "🎉 ALL TESTS PASSED - PRODUCTION READY!"
else
    echo ""
    echo "⚠️  Some tests failed. Review above."
fi
echo "========================================"
