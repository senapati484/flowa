#!/bin/bash

# Flowa Test Runner
# Runs all example files and reports status

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "🚀 Starting Flowa Tests..."
echo "=========================="

# List of new consolidated example files
TESTS=(
    "01_basics.flowa"
    "02_control_flow.flowa"
    "03_functions.flowa"
    "04_data_structures.flowa"
    "05_modules.flowa"
    "06_http_client.flowa"
    "07_auth.flowa"
    "08_files.flowa"
    "09_advanced_features.flowa"
)

PASS_COUNT=0
FAIL_COUNT=0

# Build the latest version
echo "📦 Building Flowa..."

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Go to project root to build
cd "$PROJECT_ROOT"
go build -o flowa_new ./cmd/flowa

if [ ! -f "flowa_new" ]; then
    echo -e "${RED}❌ Build failed!${NC}"
    exit 1
fi

# Go to examples directory to run tests
cd "$SCRIPT_DIR"

echo ""

for test in "${TESTS[@]}"; do
    echo -n "Testing $test... "
    
    # Run test and capture output/exit code
    OUTPUT=$("$PROJECT_ROOT/flowa_new" "$test" 2>&1)
    EXIT_CODE=$?
    
    if [ $EXIT_CODE -eq 0 ]; then
        echo -e "${GREEN}PASS ✅${NC}"
        ((PASS_COUNT++))
    else
        echo -e "${RED}FAIL ❌${NC}"
        echo "$OUTPUT"
        ((FAIL_COUNT++))
    fi
done

echo ""
echo "=========================="
echo "Results: $PASS_COUNT passed, $FAIL_COUNT failed"

if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "${GREEN}✨ ALL TESTS PASSED! ✨${NC}"
    exit 0
else
    exit 1
fi
