#!/bin/bash
# Performance Benchmark

echo "=== PERFORMANCE BENCHMARK ==="
echo ""

# Simple loop performance
echo "Test 1: Loop Performance (10000 iterations)"
time ./flowa_new -c 'for(i=0; i<10000; i=i+1) { x = i * 2 }' 2>&1 | grep "real\|user\|sys"

echo ""
echo "Test 2: Function Call Performance (1000 calls)"
cat > /tmp/perf_test.flowa << 'EOF'
func add(a, b) {
    return a + b
}
for(i=0; i<1000; i=i+1) {
    result = add(i, i+1)
}
EOF
time ./flowa_new /tmp/perf_test.flowa 2>&1 | grep "real\|user\|sys"

echo ""
echo "Test 3: String Concatenation (1000 ops)"
time ./flowa_new -c 'for(i=0; i<1000; i=i+1) { s = "test" + i }' 2>&1 | grep "real\|user\|sys"

echo ""
echo "✅ Performance tests complete"
