#!/usr/bin/env bash
#
# Demonstration script that calls op-verify.sh, receives one JSON object, and
# then prints it in the old-school, color/styled way (headings, lines, subcalls).
#
# Usage:
#   ./verify.sh <path-to-transaction-file.json>
#

set -Eeuo pipefail
IFS=$'\n\t'

###############################################################################
# STYLED PRINTING
###############################################################################
BOLD="\e[1m"
DIM="\e[2m"
RESET="\e[0m"
BLUE="\e[34m"
CYAN="\e[36m"
GREEN="\e[32m"
MAGENTA="\e[35m"
YELLOW="\e[33m"

print_heading() {
  local heading="$1"
  echo -e "${BOLD}${CYAN}${heading}${RESET}"
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
}

print_divider() {
  echo -e "${DIM}${MAGENTA}──────────────────━━━━━━━───────────────────────────────────────────────────────${RESET}"
}

###############################################################################
# RECURSIVE LOGIC: PRINT A FUNCTION CALL + SUBCALLS
###############################################################################
# We'll mimic the older script's approach: "TRANSACTION DETAILS (TX #n)",
# function name, target, etc. Tracking subcalls if present.
###############################################################################

# We'll keep a global transaction index to label subcalls as #2, #3, ...
call_counter=1

print_call() {
  local call_json="$1"
  local is_nested="${2:-false}"

  # Grab high-level keys
  local fn_name="$(echo "$call_json" | jq -r '.functionName')"
  local target_addr="$(echo "$call_json" | jq -r '.target')"
  local target_name="$(echo "$call_json" | jq -r '.targetName')"

  # Show headings differently if is_nested or top-level
  if [[ "$is_nested" == "false" ]]; then
    print_heading "TRANSACTION DETAILS (TX #$call_counter)"
  else
    print_heading "SUBCALL DETAILS (TX #$call_counter)"
  fi

  # Increment the global call counter
  ((call_counter++))

  printf "${GREEN}%-10s${RESET}: %s (%s)\n" "Target" "$target_addr" "$target_name"
  printf "${GREEN}%-10s${RESET}: %s\n\n" "Function" "$fn_name"

  # 1) If rawData is present (no parsedData, or unknown function):
  local raw_data="$(echo "$call_json" | jq -r '.rawData // empty')"
  if [[ -n "$raw_data" ]]; then
    echo -e "${DIM}Raw Data:${RESET}"
    echo "$raw_data" | fold -w 80 -s
    echo ""
    return
  fi

  # 2) Otherwise, check for parsedData
  local parsed_data="$(echo "$call_json" | jq -r '.parsedData // empty')"
  if [[ -n "$parsed_data" && "$parsed_data" != "null" ]]; then
    # Check if there's a subcalls array in parsedData
    local subcalls="$(echo "$parsed_data" | jq -r '.subcalls // empty')"
    if [[ -n "$subcalls" && "$subcalls" != "null" ]]; then
      # This is a multicall array
      local count_subcalls="$(echo "$subcalls" | jq -r 'length')"
      print_heading "TRANSACTIONS ARE BEING NESTED WITH MULTICALL3"
      echo -e "${BOLD}Number of sub-transactions:${RESET} $count_subcalls"
      echo ""

      # Iterate over subcalls array
      for ((i=0; i<"$count_subcalls"; i++)); do
        local subcall_json="$(echo "$subcalls" | jq -r ".[$i]")"
        print_call "$subcall_json" "true"
      done
      return
    else
      # Check if parsedData only contains rawData
      local raw_data_in_parsed="$(echo "$parsed_data" | jq -r '.rawData // empty')"
      local keys_count="$(echo "$parsed_data" | jq -r 'keys | length')"
      
      if [[ -n "$raw_data_in_parsed" && "$keys_count" -eq 1 ]]; then
        # Only contains rawData, print it specially
        echo -e "${DIM}Raw Data:${RESET}"
        echo "$raw_data_in_parsed" | fold -w 80 -s
        echo ""
        return
      fi
      
      # Print each field in parsedData in a readable format (excluding rawData)
      echo -e "${DIM}Parsed Data:${RESET}"
      
      # Get all the keys from parsedData except rawData
      local keys=$(echo "$parsed_data" | jq -r 'keys[] | select(. != "rawData")')
      
      # Iterate through each key and print its value
      for key in $keys; do
        # Get the value for this key
        local value=$(echo "$parsed_data" | jq -r --arg k "$key" '.[$k] | tostring')
        
        # Format the output
        printf "${YELLOW}%-15s${RESET}: %s\n" "$key" "$value"
      done
      
      echo ""
      return
    fi
  fi

  # 3) Fallback if nothing else matched
  echo -e "${DIM}Raw Function Data (fallback):${RESET}"
  echo "$call_json" | jq -r '.functionData' | fold -w 80 -s
  echo ""
}

###############################################################################
# MAIN
###############################################################################

transaction_file="${1:-}"

if [[ -z "$transaction_file" ]]; then
  echo "Usage: $0 <path-to-tx-file.json>"
  exit 1
fi

# 1. Capture JSON from op-verify.sh
json_result=""
if ! json_result="$(
  "$(dirname "$0")/strategies/forge/op-verify.sh" \
    --tx "$transaction_file" \
)"; then
  echo -e "${BOLD}${YELLOW}ERROR:${RESET} Failed to verify transaction using op-verify.sh"
  echo "Please check that the transaction file exists and is valid."
  exit 1
fi

# Check if the result is valid JSON
if ! echo "$json_result" | jq . >/dev/null 2>&1; then
  echo -e "${BOLD}${YELLOW}ERROR:${RESET} op-verify.sh returned invalid JSON:"
  echo "$json_result"
  exit 1
fi

# 2. Parse out domainHash / messageHash
domain_hash="$(echo "$json_result" | jq -r '.domainHash')"
message_hash="$(echo "$json_result" | jq -r '.messageHash')"
tx_obj="$(echo "$json_result" | jq -r '.transaction')"

# 3. Print a "BASIC TRANSACTION DETAILS" block
echo ""
print_heading "BASIC TRANSACTION DETAILS"
safe="$(echo "$tx_obj" | jq -r '.safe')"
chain="$(echo "$tx_obj" | jq -r '.chain')"
nonce_val="$(echo "$tx_obj" | jq -r '.nonce')"
to="$(echo "$tx_obj" | jq -r '.to')"
value="$(echo "$tx_obj" | jq -r '.value')"
operation="$(echo "$tx_obj" | jq -r '.operation')"

printf "${BOLD}%-14s${RESET}: %s\n" "Safe"        "$safe"
printf "${BOLD}%-14s${RESET}: %s\n" "Chain ID"    "$chain"
printf "${BOLD}%-14s${RESET}: %s\n" "Target"      "$to"
printf "${BOLD}%-14s${RESET}: %s\n" "ETH Value"   "$value"
printf "${BOLD}%-14s${RESET}: %s\n" "Nonce"       "$nonce_val"
printf "${BOLD}%-14s${RESET}: %s\n" "Operation"   "$operation"
echo ""

# 4. Print the top-level function call details (and subcalls if it's Multicall)
top_level_call="$(echo "$tx_obj" | jq -r '.call')"
print_call "$top_level_call" "false"

# 5. Print the domain and message hashes
print_heading "HASHES"
printf "${BOLD}%-12s${RESET}: %s\n" "Domain Hash"  "$domain_hash"
printf "${BOLD}%-12s${RESET}: %s\n" "Message Hash" "$message_hash"
echo ""

# 6. Print final verification instructions
print_heading "VERIFICATION INSTRUCTIONS"
echo -e "${BOLD}1. Transaction details should EXACTLY MATCH what you expect to see.${RESET}"
echo -e "${BOLD}2. Domain and message hashes should EXACTLY MATCH other machines.${RESET}"
echo -e "${BOLD}3. Hardware wallet should show you the EXACT SAME HASHES.${RESET}"
echo -e "${BOLD}4. WHEN IN DOUBT, ASK FOR HELP.${RESET}"
echo ""
