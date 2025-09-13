#!/bin/bash

# Demo script for Jackadi - showcases all tasks and specs from the demo plugin
# This script runs each task and spec collector to demonstrate Jackadi's capabilities

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
AGENT_ID="agent1"
COLLECTION="demo"
JACK_CMD="/usr/bin/jack"

# Check if jack binary exists
if ! which jack >/dev/null 2>&1; then
    echo -e "${RED}Error: jack binary not found. Please install JACK or run 'make build' first.${NC}"
    exit 1
fi

# Helper function to print section headers
print_header() {
    echo
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo
}

# Helper function to run a task with description
run_task() {
    local task_name="$1"
    local description="$2"
    local display_override="$3"

    if [[ "$display_override" == "--display" ]]; then
        shift 3
        local cmd_display="$1"
        shift 1
    else
        shift 2
        local cmd_display="$JACK_CMD run $AGENT_ID $COLLECTION:$task_name"
        for arg in "$@"; do
            cmd_display="$cmd_display $arg"
        done
    fi

    echo -e "${YELLOW}Running: $description${NC}"
    echo -e "${GREEN}Command: $cmd_display${NC}"
    echo

    if $JACK_CMD run $AGENT_ID $COLLECTION:$task_name "$@"; then
        echo -e "${GREEN}✓ Task completed successfully${NC}"
    else
        echo -e "${RED}✗ Task failed${NC}"
    fi

    echo
    echo "---"
    echo -e "${BLUE}Press Enter to continue to the next task...${NC}"
    read
}

# Helper function to run spec commands
run_spec() {
    local spec_name="$1"
    local description="$2"

    echo -e "${YELLOW}Spec Collector: $description${NC}"
    echo -e "${GREEN}Command: $JACK_CMD run $AGENT_ID specs:get $COLLECTION.$spec_name${NC}"
    echo

    if $JACK_CMD run $AGENT_ID specs:get $COLLECTION.$spec_name; then
        echo -e "${GREEN}✓ Spec retrieved successfully${NC}"
    else
        echo -e "${RED}✗ Spec retrieval failed${NC}"
    fi

    echo
    echo "---"
    echo -e "${BLUE}Press Enter to continue to the next spec...${NC}"
    read
}

print_header "JACKADI DEMO - TASK EXECUTION SHOWCASE"

echo "This demo showcases Jackadi's distributed task execution capabilities."
echo "All tasks will be executed on agent: $AGENT_ID"
echo "Plugin collection: $COLLECTION"
echo

# Wait for user confirmation
echo -e "${YELLOW}Press Enter to start the demo, or Ctrl+C to cancel...${NC}"
read

echo "Syncing plugins"
$JACK_CMD run $AGENT_ID plugins:sync

# ============================================================================
# BASIC TASKS
# ============================================================================

print_header "1. BASIC TASKS"

run_task "hello" "Simple hello world task (no arguments)"

run_task "configure_service" "Configure service with options" \
    nginx-pro Verbose=true Region=eu-west-1 Timeout=60

run_task "monitor_health" "System health monitoring (context-aware)"

# ============================================================================
# COMPLEX INPUT TYPES
# ============================================================================

print_header "2. COMPLEX INPUT TYPES"

run_task "create_user" "Create user with multiple argument types" --display \
    "$JACK_CMD run $AGENT_ID $COLLECTION:create_user 12345 johndoe john@jackadi.io true '[\"read\",\"write\",\"admin\"]' '{\"department\":\"engineering\",\"team\":\"backend\"}' '{\"hostname\":\"web-01\",\"cpu_cores\":8,\"memory_gb\":16,\"is_production\":true}' '[100,200,300]'" \
    12345 johndoe john@jackadi.io true \
    '["read","write","admin"]' \
    '{"department":"engineering","team":"backend"}' \
    '{"hostname":"web-01","cpu_cores":8,"memory_gb":16,"is_production":true}' \
    '[100,200,300]'

run_task "find_user" "Find user by email (pointer return type)" \
    admin@jackadi.io

run_task "find_user" "Find non-existent user (returns nil)" \
    nonexistent@jackadi.io

# ============================================================================
# OUTPUT TYPE DEMONSTRATIONS
# ============================================================================

print_header "3. OUTPUT TYPE DEMONSTRATIONS"

run_task "get_system_version" "String output - OS version"

run_task "get_connection_count" "Integer output - connection count"

run_task "is_maintenance_mode" "Boolean output - maintenance mode status"

run_task "get_cpu_usage" "Float output - CPU usage percentage"

run_task "list_services" "String slice output - active services"

run_task "get_users" "Struct slice output - user list"

run_task "get_env_vars" "String map output - environment variables"

run_task "get_metrics" "Mixed interface{} map output - system metrics"

run_task "get_server_info" "Complex struct output - server information"

run_task "get_reboot_history" "Fixed array output - reboot timestamps"

run_task "get_db_stats" "Complex nested output - database statistics"

# ============================================================================
# SYSTEM OPERATIONS
# ============================================================================

print_header "4. SYSTEM OPERATIONS (WITH LOCKS)"

run_task "upgrade_system" "OS upgrade (dry run)" \
    DryRun=true SecurityOnly=true BackupBefore=true

run_task "upgrade_system" "OS upgrade (exclude packages)" --display \
    "$JACK_CMD run $AGENT_ID $COLLECTION:upgrade_system ExcludePackages='[\"kernel-default\",\"systemd\"]' RebootRequired=true" \
    ExcludePackages='["kernel-default","systemd"]' RebootRequired=true

# ============================================================================
# SPEC COLLECTION
# ============================================================================

print_header "5. SPEC COLLECTORS (SYSTEM INVENTORY)"

echo "Spec collectors gather system information automatically when agents connect."
echo "They enable intelligent task targeting and system inventory management."
echo "The demo plugin includes the following spec collectors:"
echo

run_spec "os" "operating system information"
run_spec "hardware" "hardware specifications"
run_spec "hardware.CPUCores" "CPUCores in hardware specifications struct"

print_header "ALL SPECS SUMMARY"

echo -e "${YELLOW}Retrieving all specs for agent: $AGENT_ID${NC}"
echo -e "${GREEN}Command: $JACK_CMD run $AGENT_ID specs:all${NC}"
echo

if $JACK_CMD spec:all $AGENT_ID; then
    echo -e "${GREEN}✓ All specs retrieved successfully${NC}"
else
    echo -e "${RED}✗ Failed to retrieve all specs${NC}"
fi

echo
echo -e "${BLUE}Press Enter to continue to demo completion...${NC}"
read

# ============================================================================
# DEMO COMPLETE
# ============================================================================

print_header "DEMO COMPLETE"

echo -e "${GREEN}✓ All tasks and specs have been executed successfully!${NC}"
echo
echo "Key demonstrations completed:"
echo "• Simple tasks (hello, configure, monitor)"
echo "• Complex input types (user creation with mixed arguments)"
echo "• Various output types (strings, numbers, arrays, structs, maps)"
echo "• System operations with write locks (OS upgrades)"
echo "• Spec collection for system inventory"
echo
echo "You can now:"
echo "• Check task results: $JACK_CMD results list"
echo "• View agent information and specs: $JACK_CMD agent list"
echo "• Target agents by specs: $JACK_CMD run -q 'specs.os.distribution == \"opensuse-tumbleweed\"' demo:hello"
echo
echo -e "${BLUE}Thank you for trying Jackadi!${NC}"