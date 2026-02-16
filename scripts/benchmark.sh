#!/bin/bash

# Go-Collider API Benchmark Script
# Based on wrk benchmarking tool

set -e

# Configuration
HOST="${1:-http://localhost:8080}"
BENCH_NAME="${2:-}"
EVENT_TYPE="${3:-page_view}"
THREADS=4
CONNECTIONS=50
DURATION="10s"
USER_ID=1

# Directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
RUNTIME_DIR="$PROJECT_ROOT/runtime/wrk_scripts"
BENCHMARK_DIR="$PROJECT_ROOT/benchmarks"

# Output file with datetime
DATETIME=$(date +"%Y%m%d_%H%M%S")
if [ -n "$BENCH_NAME" ]; then
    OUTPUT_FILE="$BENCHMARK_DIR/${DATETIME}_benchmark_${BENCH_NAME}.md"
else
    OUTPUT_FILE="$BENCHMARK_DIR/${DATETIME}_benchmark_.md"
fi

# Create directories
mkdir -p "$RUNTIME_DIR"
mkdir -p "$BENCHMARK_DIR"

# Create JSON data for POST request
cat > "$RUNTIME_DIR/event_data.json" << EOF
{
  "user_id": $USER_ID,
  "event_type": "$EVENT_TYPE",
  "metadata": {
    "page": "/dashboard",
    "referrer": "https://google.com"
  }
}
EOF

# Initialize output file
if [ -n "$BENCH_NAME" ]; then
    TITLE="Go-Collider Benchmark Report - $BENCH_NAME"
else
    TITLE="Go-Collider Benchmark Report"
fi

cat > "$OUTPUT_FILE" << EOF
# $TITLE

**Date:** $(date)
**Configuration:**
- Host: $HOST
- Benchmark Name: ${BENCH_NAME:-default}
- Threads: $THREADS
- Connections: $CONNECTIONS
- Duration: $DURATION per test
- User ID: $USER_ID
- Event Type: $EVENT_TYPE

---

EOF

echo "=================================="
echo "Go-Collider Benchmark"
echo "=================================="
echo "Host: $HOST"
echo "Benchmark Name: ${BENCH_NAME:-default}"
echo "Threads: $THREADS"
echo "Connections: $CONNECTIONS"
echo "Duration: $DURATION"
echo "User ID: $USER_ID"
echo "Event Type: $EVENT_TYPE"
echo "Output: $OUTPUT_FILE"
echo "=================================="

# Lua script for POST requests
cat > "$RUNTIME_DIR/post_event.lua" << 'EOF'
-- POST request for creating events
wrk.method = "POST"
wrk.headers["Content-Type"] = "application/json"

local file = io.open("runtime/wrk_scripts/event_data.json", "r")
if file then
    wrk.body = file:read("*all")
    file:close()
end

function response(status, headers, body)
    if status ~= 200 and status ~= 201 then
        print("Error response: " .. status)
    end
end
EOF

# Lua script for dynamic GET user events
cat > "$RUNTIME_DIR/get_user_events.lua" << 'EOF'
-- GET request for user events
local user_id = "1"

local params = {
    "?limit=10",
    "?limit=50",
    "?limit=100",
    "?page=1&limit=20",
    "?page=2&limit=20"
}

local counter = 0

function request()
    counter = counter + 1
    local param = params[(counter % #params) + 1]
    local path = "/users/" .. user_id .. "/events" .. param
    return wrk.format("GET", path)
end
EOF

# Lua script for stats requests
cat > "$RUNTIME_DIR/get_stats.lua" << EOF
-- GET request for statistics
local event_type = "$EVENT_TYPE"

local params = {
    "?from=2025-01-01T00:00:00Z&to=2026-12-31T23:59:59Z&type=" .. event_type,
    "?from=2025-06-01T00:00:00Z&to=2025-12-31T23:59:59Z&type=" .. event_type,
    "?type=" .. event_type,
    "?from=2025-01-01T00:00:00Z&to=2026-01-01T00:00:00Z"
}

local counter = 0

function request()
    counter = counter + 1
    local param = params[(counter % #params) + 1]
    local path = "/stats" .. param
    return wrk.format("GET", path)
end
EOF

# Function to run benchmark and save results
run_test() {
    local test_name="$1"
    local wrk_args="$2"

    echo ""
    echo "----------------------------------"
    echo "Test: $test_name"
    echo "----------------------------------"

    RESULT=$(wrk -t$THREADS -c$CONNECTIONS -d$DURATION --latency $wrk_args 2>&1)
    echo "$RESULT"

    # Append to markdown file
    cat >> "$OUTPUT_FILE" << EOF

## $test_name

\`\`\`
$RESULT
\`\`\`

EOF
}

# Test 1: POST /events
run_test "Create Event (POST /events)" "-s $RUNTIME_DIR/post_event.lua $HOST/events"

# Test 2: GET /events
run_test "Events List (GET /events)" "$HOST/events?page=1&limit=100"

# Test 3: GET /users/{uid}/events
run_test "User Events (GET /users/{uid}/events)" "-s $RUNTIME_DIR/get_user_events.lua $HOST"

# Test 4: GET /stats
run_test "Stats (GET /stats)" "-s $RUNTIME_DIR/get_stats.lua $HOST"

# Finalize report
cat >> "$OUTPUT_FILE" << EOF

---

**Benchmark completed at:** $(date)

EOF

echo ""
echo "=================================="
echo "All tests completed!"
echo "=================================="
echo ""
echo "Results saved to: $OUTPUT_FILE"
echo ""

# Cleanup
rm -f "$RUNTIME_DIR/event_data.json"
rm -rf "$RUNTIME_DIR"
