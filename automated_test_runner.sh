#!/bin/bash

# STRIGOI AUTOMATED TEST RUNNER
# Nela Park Standard - Zero Human Interaction
# Runs complete test suite and generates comprehensive report

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_DIR="./test_results/$(date +%Y%m%d_%H%M%S)"
STRIGOI_BIN="./strigoi"
TEST_TIMEOUT=30
PARALLEL_TESTS=4
LOG_LEVEL="INFO"

# Statistics
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Create test directory
mkdir -p "$TEST_DIR"
mkdir -p "$TEST_DIR/logs"
mkdir -p "$TEST_DIR/artifacts"
mkdir -p "$TEST_DIR/performance"

# Logging function
log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $1" | tee -a "$TEST_DIR/test_run.log"
}

# Test execution function
run_test() {
    local test_id=$1
    local test_cmd=$2
    local expected_pattern=$3
    local max_duration_ms=${4:-5000}
    local test_category=${5:-FUNCTIONAL}
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    echo -n "[$TOTAL_TESTS] Running $test_id... "
    
    # Prepare test environment
    local test_log="$TEST_DIR/logs/${test_id}.log"
    local test_result="$TEST_DIR/${test_id}.yaml"
    
    # Execute with timeout and capture everything
    local start_time=$(date +%s%N)
    local exit_code=0
    
    # Run test with timeout
    if timeout "$TEST_TIMEOUT" bash -c "$test_cmd" > "$test_log" 2>&1; then
        exit_code=0
    else
        exit_code=$?
    fi
    
    local end_time=$(date +%s%N)
    local duration_ms=$(( (end_time - start_time) / 1000000 ))
    
    # Check if output matches expected pattern
    local status="FAIL"
    if [[ $exit_code -eq 124 ]]; then
        status="TIMEOUT"
        echo -e "${YELLOW}TIMEOUT${NC}"
    elif grep -q "$expected_pattern" "$test_log" 2>/dev/null; then
        if [[ $duration_ms -le $max_duration_ms ]]; then
            status="PASS"
            PASSED_TESTS=$((PASSED_TESTS + 1))
            echo -e "${GREEN}PASS${NC} (${duration_ms}ms)"
        else
            status="SLOW"
            echo -e "${YELLOW}SLOW${NC} (${duration_ms}ms > ${max_duration_ms}ms)"
        fi
    else
        status="FAIL"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        echo -e "${RED}FAIL${NC}"
    fi
    
    # Generate YAML result
    cat > "$test_result" <<EOF
test_id: $test_id
category: $test_category
command: "$test_cmd"
status: $status
exit_code: $exit_code
duration_ms: $duration_ms
max_duration_ms: $max_duration_ms
expected_pattern: "$expected_pattern"
timestamp: $(date -Iseconds)
log_file: logs/${test_id}.log
EOF

    # Capture system metrics during test
    if command -v vmstat &> /dev/null; then
        vmstat 1 2 | tail -1 >> "$TEST_DIR/performance/${test_id}.perf"
    fi
    
    return 0
}

# Performance test function
perf_test() {
    local test_id=$1
    local test_cmd=$2
    local iterations=${3:-10}
    
    echo -n "Performance testing $test_id ($iterations iterations)... "
    
    local total_time=0
    local min_time=999999
    local max_time=0
    
    for i in $(seq 1 $iterations); do
        local start=$(date +%s%N)
        timeout 5 bash -c "$test_cmd" > /dev/null 2>&1
        local end=$(date +%s%N)
        local duration=$(( (end - start) / 1000000 ))
        
        total_time=$((total_time + duration))
        [[ $duration -lt $min_time ]] && min_time=$duration
        [[ $duration -gt $max_time ]] && max_time=$duration
    done
    
    local avg_time=$((total_time / iterations))
    
    cat > "$TEST_DIR/performance/${test_id}_perf.yaml" <<EOF
test_id: $test_id
iterations: $iterations
avg_ms: $avg_time
min_ms: $min_time
max_ms: $max_time
timestamp: $(date -Iseconds)
EOF
    
    echo -e "${GREEN}Done${NC} (avg: ${avg_time}ms)"
}

# Stress test function
stress_test() {
    local test_id=$1
    local test_cmd=$2
    local concurrent=${3:-10}
    
    echo -n "Stress testing $test_id ($concurrent concurrent)... "
    
    local pids=()
    local start=$(date +%s)
    
    for i in $(seq 1 $concurrent); do
        timeout 10 bash -c "$test_cmd" > /dev/null 2>&1 &
        pids+=($!)
    done
    
    # Wait for all to complete
    local failed=0
    for pid in "${pids[@]}"; do
        if ! wait $pid; then
            failed=$((failed + 1))
        fi
    done
    
    local end=$(date +%s)
    local duration=$((end - start))
    
    if [[ $failed -eq 0 ]]; then
        echo -e "${GREEN}PASS${NC} (${duration}s, all succeeded)"
    else
        echo -e "${YELLOW}PARTIAL${NC} (${duration}s, $failed/$concurrent failed)"
    fi
}

# Build Strigoi if needed
build_strigoi() {
    if [[ ! -f "$STRIGOI_BIN" ]]; then
        log "Building Strigoi..."
        if go build -o "$STRIGOI_BIN" ./cmd/strigoi; then
            log "Build successful"
        else
            log "Build failed - some tests may fail"
        fi
    fi
}

# Main test execution
main() {
    log "=== STRIGOI AUTOMATED TEST SUITE ==="
    log "Test directory: $TEST_DIR"
    log "Binary: $STRIGOI_BIN"
    
    # Build if needed
    build_strigoi
    
    # CORE TESTS
    log "\n${BLUE}=== CORE COMMAND TESTS ===${NC}"
    run_test "TEST-CORE-0001" "$STRIGOI_BIN" "Strigoi" 500 "FUNCTIONAL"
    run_test "TEST-CORE-0002" "$STRIGOI_BIN --help" "Usage:" 100 "FUNCTIONAL"
    run_test "TEST-CORE-0003" "$STRIGOI_BIN --version" "v0.5" 100 "FUNCTIONAL"
    run_test "TEST-CORE-0004" "$STRIGOI_BIN --brief" "Strigoi" 100 "FUNCTIONAL"
    run_test "TEST-CORE-0005" "$STRIGOI_BIN --examples" "Strigoi" 100 "FUNCTIONAL"
    
    # PROBE TESTS
    log "\n${BLUE}=== PROBE COMMAND TESTS ===${NC}"
    run_test "TEST-PROBE-0001" "$STRIGOI_BIN probe --help" "probe" 100 "FUNCTIONAL"
    run_test "TEST-PROBE-0002" "$STRIGOI_BIN probe north --help" "API endpoints" 100 "FUNCTIONAL"
    run_test "TEST-PROBE-0003" "$STRIGOI_BIN probe south --help" "Analyze" 100 "FUNCTIONAL"
    run_test "TEST-PROBE-0004" "$STRIGOI_BIN probe east --help" "Trace" 100 "FUNCTIONAL"
    run_test "TEST-PROBE-0005" "$STRIGOI_BIN probe west --help" "Examine" 100 "FUNCTIONAL"
    
    # Test with mock targets (won't actually probe external services)
    # DISABLED: dry-run flag not implemented yet
    # run_test "TEST-PROBE-0010" "$STRIGOI_BIN probe north --target localhost --dry-run" "would scan" 1000 "FUNCTIONAL"
    # run_test "TEST-PROBE-0011" "$STRIGOI_BIN probe south --scan-mcp . --dry-run" "would scan" 2000 "FUNCTIONAL"
    
    # STREAM TESTS
    log "\n${BLUE}=== STREAM COMMAND TESTS ===${NC}"
    run_test "TEST-STREAM-0001" "$STRIGOI_BIN stream --help" "stream" 100 "FUNCTIONAL"
    run_test "TEST-STREAM-0002" "$STRIGOI_BIN stream tap --help" "tap" 100 "FUNCTIONAL"
    run_test "TEST-STREAM-0003" "$STRIGOI_BIN stream analyze --help" "analyze" 100 "FUNCTIONAL"
    
    # SESSION TESTS
    log "\n${BLUE}=== SESSION COMMAND TESTS ===${NC}"
    run_test "TEST-SESSION-0001" "$STRIGOI_BIN session --help" "session" 100 "FUNCTIONAL"
    run_test "TEST-SESSION-0002" "$STRIGOI_BIN session list" "session" 500 "FUNCTIONAL"
    
    # MODULE TESTS
    log "\n${BLUE}=== MODULE COMMAND TESTS ===${NC}"
    run_test "TEST-MODULE-0001" "$STRIGOI_BIN module --help" "module" 100 "FUNCTIONAL"
    run_test "TEST-MODULE-0002" "$STRIGOI_BIN module list" "Available modules" 500 "FUNCTIONAL"
    
    # PERFORMANCE TESTS
    log "\n${BLUE}=== PERFORMANCE TESTS ===${NC}"
    perf_test "PERF-STARTUP" "$STRIGOI_BIN --version" 20
    perf_test "PERF-HELP" "$STRIGOI_BIN --help" 20
    
    # STRESS TESTS
    log "\n${BLUE}=== STRESS TESTS ===${NC}"
    stress_test "STRESS-CONCURRENT" "$STRIGOI_BIN --version" 50
    
    # INTEGRATION TESTS
    log "\n${BLUE}=== INTEGRATION TESTS ===${NC}"
    
    # Test pipeline
    if [[ -f "./test_data/sample.pcap" ]]; then
        run_test "TEST-INT-0001" "$STRIGOI_BIN stream analyze --input ./test_data/sample.pcap" "Protocol" 5000 "INTEGRATION"
    else
        log "Skipping PCAP test - no test data"
        SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
    fi
    
    # VSM FEEDBACK LOOP TESTS
    log "\n${BLUE}=== VSM FEEDBACK LOOP VALIDATION ===${NC}"
    
    # Check for self-regulation
    run_test "TEST-VSM-0001" "$STRIGOI_BIN module list" "Available modules" 1000 "VSM"
    
    # Generate test report
    generate_report
}

# Report generation
generate_report() {
    local pass_rate=0
    if [[ $TOTAL_TESTS -gt 0 ]]; then
        pass_rate=$(( (PASSED_TESTS * 100) / TOTAL_TESTS ))
    fi
    
    local report_file="$TEST_DIR/TEST_REPORT.md"
    
    cat > "$report_file" <<EOF
# STRIGOI AUTOMATED TEST REPORT
## Generated: $(date)

### SUMMARY
- **Total Tests**: $TOTAL_TESTS
- **Passed**: $PASSED_TESTS ✅
- **Failed**: $FAILED_TESTS ❌
- **Skipped**: $SKIPPED_TESTS ⏭️
- **Pass Rate**: ${pass_rate}%

### TEST CATEGORIES
\`\`\`
FUNCTIONAL: Core functionality tests
PERFORMANCE: Speed and resource tests
INTEGRATION: Multi-component tests
VSM: Feedback loop validation
\`\`\`

### DETAILED RESULTS
EOF

    # Add individual test results
    echo -e "\n#### Test Details\n" >> "$report_file"
    for result in "$TEST_DIR"/*.yaml; do
        if [[ -f "$result" ]]; then
            local test_id=$(grep "test_id:" "$result" | cut -d' ' -f2)
            local status=$(grep "status:" "$result" | cut -d' ' -f2)
            local duration=$(grep "duration_ms:" "$result" | cut -d' ' -f2)
            
            local icon="❓"
            case $status in
                PASS) icon="✅" ;;
                FAIL) icon="❌" ;;
                SLOW) icon="🐌" ;;
                TIMEOUT) icon="⏱️" ;;
            esac
            
            echo "- $test_id: $icon $status (${duration}ms)" >> "$report_file"
        fi
    done
    
    # Add performance summary if available
    if ls "$TEST_DIR/performance"/*_perf.yaml &> /dev/null; then
        echo -e "\n### PERFORMANCE METRICS\n" >> "$report_file"
        for perf in "$TEST_DIR/performance"/*_perf.yaml; do
            local test_id=$(grep "test_id:" "$perf" | cut -d' ' -f2)
            local avg_ms=$(grep "avg_ms:" "$perf" | cut -d' ' -f2)
            echo "- $test_id: avg ${avg_ms}ms" >> "$report_file"
        done
    fi
    
    # VSM Compliance
    echo -e "\n### VSM COMPLIANCE\n" >> "$report_file"
    echo "- Feedback Loops Tested: 0/51 (0%)" >> "$report_file"
    echo "- To be implemented in Phase 2" >> "$report_file"
    
    # Recommendations
    echo -e "\n### RECOMMENDATIONS\n" >> "$report_file"
    if [[ $pass_rate -ge 97 ]]; then
        echo "✅ **READY FOR RELEASE** - Pass rate exceeds 97% target" >> "$report_file"
    elif [[ $pass_rate -ge 90 ]]; then
        echo "⚠️ **NEARLY READY** - Address remaining failures" >> "$report_file"
    else
        echo "❌ **NOT READY** - Significant issues need resolution" >> "$report_file"
    fi
    
    # Output summary
    log "\n${GREEN}=== TEST SUITE COMPLETE ===${NC}"
    log "Total: $TOTAL_TESTS | Passed: $PASSED_TESTS | Failed: $FAILED_TESTS | Pass Rate: ${pass_rate}%"
    log "Full report: $report_file"
    log "Test logs: $TEST_DIR/logs/"
    
    # Exit code based on pass rate
    if [[ $pass_rate -ge 97 ]]; then
        exit 0
    else
        exit 1
    fi
}

# Run main test suite
main "$@"