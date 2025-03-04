#!/usr/bin/env bash
#
# Simplified Safe transaction validation script. Generally inspired by:
# https://github.com/pcaversaccio/safe-tx-hashes-util
#
# Differences from the original approach:
#   - Only uses the cast binary (no other Foundry tools).
#   - Always takes a transaction file as input (no Safe API).
#   - No support for older Safe transaction formats.
#
# Usage:
#   ./op-verify.sh --tx <path-to-tx-file.json>
#

set -Eeuo pipefail
IFS=$'\n\t'

###############################################################################
# GLOBALS
###############################################################################

# Safe-specific hashes and signatures
readonly DOMAIN_SEPARATOR_TYPEHASH="0x47e79534a245952e8b16893a336b85a3d9ea9fa8c573f3d803afb92a79469218"
readonly SAFE_TX_TYPEHASH="0xbb8310d486368db6bd6f849402fdd73ad53d316b5a4b2644ad6efe0f941286d8"
readonly DOMAIN_SEPARATOR_SIG="fn(bytes32,uint256,address)"
readonly SAFE_TX_SIG="fn(bytes32,address,uint256,bytes32,uint8,uint256,uint256,uint256,address,address,uint256)"

# Known function signatures
readonly TRANSFER_SIG="transfer(address,uint256)"
readonly TRANSFER_FROM_SIG="transferFrom(address,address,uint256)"
readonly APPROVE_SIG="approve(address,uint256)"
readonly INCREASE_ALLOWANCE_SIG="increaseAllowance(address,uint256)"
readonly DECREASE_ALLOWANCE_SIG="decreaseAllowance(address,uint256)"
readonly APPROVE_HASH_SIG="approveHash(bytes32)"
readonly AGGREGATE3_SIG="aggregate3((address,bool,bytes)[])"

# Known addresses
readonly MULTICALL3_ADDRESS="0xcA11bde05977b3631167028862bE2a173976CA11"
readonly USDC_MAINNET_ADDRESS="0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
readonly OP_TOKEN_ADDRESS="0x4200000000000000000000000000000000000042"

# Text formatting
BOLD="\e[1m"
DIM="\e[2m"
RESET="\e[0m"
BLUE="\e[34m"
CYAN="\e[36m"
GREEN="\e[32m"
MAGENTA="\e[35m"
YELLOW="\e[33m"

###############################################################################
# FUNCTIONS
###############################################################################

#------------------------------------------------------------------------------
# Prints a heading with a thick dividing line.
# Globals:
#   BOLD, CYAN, RESET
# Arguments:
#   $1 - The heading text.
#------------------------------------------------------------------------------
print_heading() {
  local heading="$1"
  echo -e "${BOLD}${CYAN}${heading}${RESET}"
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
}

#------------------------------------------------------------------------------
# Prints a lighter divider line.
#------------------------------------------------------------------------------
print_divider() {
  echo -e "${DIM}${MAGENTA}──────────────────━━━━━━━───────────────────────────────────────────────────────${RESET}"
}

#------------------------------------------------------------------------------
# Prints usage instructions.
#------------------------------------------------------------------------------
usage() {
  cat <<EOF
Usage: $0 --tx <path-to-transaction-file.json>

Validates a Safe transaction JSON file and shows relevant data/hashes.

Options:
  --tx <file>   Path to the Safe transaction JSON file.

Examples:
  $0 --tx my-transaction.json
EOF
  exit 1
}

#------------------------------------------------------------------------------
# Checks that a transaction file exists.
# Globals:
#   None
# Arguments:
#   $1 - The transaction file path to check.
# Exits:
#   1 - If the file does not exist.
#------------------------------------------------------------------------------
check_transaction_file() {
  local transaction_file="$1"

  if [[ ! -f "$transaction_file" ]]; then
    echo "Error: Transaction file '$transaction_file' not found." >&2
    exit 1
  fi
}

#------------------------------------------------------------------------------
# Gets a field from a transaction JSON file by simple grep & sed.
# Globals:
#   None
# Arguments:
#   $1 - The transaction file path.
#   $2 - The JSON field key to retrieve.
# Returns:
#   The trimmed value of the requested field, or exits if not found.
#------------------------------------------------------------------------------
get_transaction_field() {
  local transaction_file="$1"
  local field="$2"

  check_transaction_file "$transaction_file"

  local value
  value=$(
    grep -o "\"$field\"[[:space:]]*:[[:space:]]*[^,}]*" "$transaction_file" \
      | sed -E "s/\"$field\"[[:space:]]*:[[:space:]]*//" \
      | sed -E 's/^"(.*)"$/\1/' \
      | sed -E 's/^[[:space:]]*|[[:space:]]*$//'
  )

  if [[ -z "$value" ]]; then
    echo "Error: Field '$field' not found in transaction file '$transaction_file'." >&2
    exit 1
  fi

  echo "$value"
}

#------------------------------------------------------------------------------
# Generates the message hash for a Safe transaction.
# Globals:
#   SAFE_TX_TYPEHASH, SAFE_TX_SIG
# Arguments:
#   $1 - The path to the transaction file.
# Returns:
#   The computed keccak256 message hash as a hex string.
#------------------------------------------------------------------------------
make_message_hash() {
  local transaction_file="$1"

  check_transaction_file "$transaction_file"

  local safe chain to value data operation
  local safe_tx_gas base_gas gas_price gas_token refund_receiver nonce

  safe=$(get_transaction_field "$transaction_file" "safe")
  chain=$(get_transaction_field "$transaction_file" "chain")
  to=$(get_transaction_field "$transaction_file" "to")
  value=$(get_transaction_field "$transaction_file" "value")
  data=$(get_transaction_field "$transaction_file" "data")
  operation=$(get_transaction_field "$transaction_file" "operation")
  safe_tx_gas=$(get_transaction_field "$transaction_file" "safe_tx_gas")
  base_gas=$(get_transaction_field "$transaction_file" "base_gas")
  gas_price=$(get_transaction_field "$transaction_file" "gas_price")
  gas_token=$(get_transaction_field "$transaction_file" "gas_token")
  refund_receiver=$(get_transaction_field "$transaction_file" "refund_receiver")
  nonce=$(get_transaction_field "$transaction_file" "nonce")

  local message_hash_input
  message_hash_input=$(
    cast abi-encode "$SAFE_TX_SIG" \
      "$SAFE_TX_TYPEHASH" \
      "$to" \
      "$value" \
      "$(cast keccak256 "$data")" \
      "$operation" \
      "$safe_tx_gas" \
      "$base_gas" \
      "$gas_price" \
      "$gas_token" \
      "$refund_receiver" \
      "$nonce"
  )

  local message_hash
  message_hash=$(cast keccak "$message_hash_input")

  echo "$message_hash"
}

#------------------------------------------------------------------------------
# Generates the domain hash for a Safe transaction.
# Globals:
#   DOMAIN_SEPARATOR_TYPEHASH, DOMAIN_SEPARATOR_SIG
# Arguments:
#   $1 - The path to the transaction file.
# Returns:
#   The computed keccak256 domain hash as a hex string.
#------------------------------------------------------------------------------
make_domain_hash() {
  local transaction_file="$1"

  check_transaction_file "$transaction_file"

  local safe chain
  safe=$(get_transaction_field "$transaction_file" "safe")
  chain=$(get_transaction_field "$transaction_file" "chain")

  local domain_hash_input
  domain_hash_input=$(
    cast abi-encode "$DOMAIN_SEPARATOR_SIG" \
      "$DOMAIN_SEPARATOR_TYPEHASH" \
      "$chain" \
      "$safe"
  )

  local domain_hash
  domain_hash=$(cast keccak "$domain_hash_input")

  echo "$domain_hash"
}

#------------------------------------------------------------------------------
# Parses a token amount given a known or unknown number of decimals.
# Globals:
#   None
# Arguments:
#   $1 - The amount (integer).
#   $2 - The decimals (integer), or an empty string if unknown.
# Returns:
#   A human-readable decimal representation.
#------------------------------------------------------------------------------
parse_token_amount() {
  local amount="$1"
  local decimals="$2"

  if [[ -n "$decimals" ]]; then
    # Using bc for handling decimal arithmetic
    amount=$(echo "scale=6; $amount / 10^$decimals" | bc)
  else
    amount="$amount (UNKNOWN DECIMALS)"
  fi

  echo "$amount"
}

#------------------------------------------------------------------------------
# Parses function data (coupled with known function selectors) to show user-
# friendly details about the transaction.
# Globals:
#   USDC_MAINNET_ADDRESS, OP_TOKEN_ADDRESS, MULTICALL3_ADDRESS, BOLD, RESET, 
#   GREEN, YELLOW, SAFE_TX_SIG, AGGREGATE3_SIG; known function name constants.
# Arguments:
#   $1 - Target contract address.
#   $2 - Calldata (hex string).
#   $3 - (Optional) Transaction index label.
#   $4 - (Optional) Whether this call is nested in a Multicall.
#------------------------------------------------------------------------------
parse_function_data() {
  local target_address="$1"
  local function_data="$2"
  local tx_number="${3:-1}"
  local is_nested="${4:-false}"

  local target_address_name=""
  local target_address_network=""
  local target_decimals=""

  # Identify known token addresses
  if [[ "$target_address" == "$USDC_MAINNET_ADDRESS" ]]; then
    target_address_name="USDC"
    target_address_network="MAINNET ONLY"
    target_decimals="6"
  elif [[ "$target_address" == "$OP_TOKEN_ADDRESS" ]]; then
    target_address_name="OP TOKEN"
    target_address_network="ALL NETWORKS"
    target_decimals="18"
  elif [[ "$target_address" == "$MULTICALL3_ADDRESS" ]]; then
    target_address_name="MULTICALL3"
    target_address_network="ALL NETWORKS"
  else
    target_address_name="UNKNOWN"
    target_address_network="UNKNOWN"
  fi

  local function_selector="${function_data:0:10}"
  local function_name="Unknown"

  # Match known function selectors
  if [[ "${#function_selector}" -lt 10 ]]; then
    function_name="UNKNOWN (insufficient data)"
  elif [[ "$function_selector" == "$(cast sig $TRANSFER_SIG)" ]]; then
    function_name="$TRANSFER_SIG"
  elif [[ "$function_selector" == "$(cast sig $TRANSFER_FROM_SIG)" ]]; then
    function_name="$TRANSFER_FROM_SIG"
  elif [[ "$function_selector" == "$(cast sig $APPROVE_SIG)" ]]; then
    function_name="$APPROVE_SIG"
  elif [[ "$function_selector" == "$(cast sig $INCREASE_ALLOWANCE_SIG)" ]]; then
    function_name="$INCREASE_ALLOWANCE_SIG"
  elif [[ "$function_selector" == "$(cast sig $DECREASE_ALLOWANCE_SIG)" ]]; then
    function_name="$DECREASE_ALLOWANCE_SIG"
  elif [[ "$function_selector" == "$(cast sig $APPROVE_HASH_SIG)" ]]; then
    function_name="$APPROVE_HASH_SIG"
  elif [[ "$function_selector" == "$(cast sig $AGGREGATE3_SIG)" ]]; then
    function_name="$AGGREGATE3_SIG"
  else
    function_name="UNKNOWN (selector: $function_selector)"
  fi

  # Disallow nested Multicall calls
  if [[ "$is_nested" == "true" && "$target_address" == "$MULTICALL3_ADDRESS" && "$function_selector" == "$(cast sig $AGGREGATE3_SIG)" ]]; then
    echo -e "${YELLOW}ERROR: Nested Multicall3 detected. This is not supported.${RESET}" >&2
    exit 1
  fi

  # If top-level or valid nested call to Multicall, parse subcalls
  if [[ "$target_address" == "$MULTICALL3_ADDRESS" && "$function_selector" == "$(cast sig $AGGREGATE3_SIG)" ]]; then
    local decoded_calls
    decoded_calls=$(cast decode-calldata "$AGGREGATE3_SIG" "$function_data")

    local total_calls
    total_calls=$(echo "$decoded_calls" | grep -o "(" | wc -l)

    print_heading "TRANSACTIONS ARE BEING AGGREGATED WITH MULTICALL"
    echo -e "${BOLD}Number of sub-transactions:${RESET} $total_calls"
    echo ""

    # Remove outer brackets [...]
    local calls_str="${decoded_calls#\[}"
    calls_str="${calls_str%\]}"

    local call_count=1
    while [[ "$calls_str" =~ \(([^,]+),\ ([^,]+),\ ([^\)]+)\)(.*) ]]; do
      local call_target="${BASH_REMATCH[1]}"
      local call_allowFailure="${BASH_REMATCH[2]}"
      local call_data="${BASH_REMATCH[3]}"
      calls_str="${BASH_REMATCH[4]}"

      # Remove leading comma and space if present
      if [[ "$calls_str" =~ ^,\ (.*) ]]; then
        calls_str="${BASH_REMATCH[1]}"
      fi

      parse_function_data "$call_target" "$call_data" "$call_count" true
      ((call_count++))
    done

  else
    print_heading "TRANSACTION DETAILS (TX #${tx_number})"
    printf "${GREEN}%-10s${RESET}: %s (%s)\n" "Target"     "$target_address" "$target_address_name"
    printf "${GREEN}%-10s${RESET}: %s\n"      "Function"   "$function_name"
    echo ""

    # If the selector is too short, just display the raw data
    if [[ "${#function_selector}" -lt 10 ]]; then
      echo -e "${DIM}Raw Data:${RESET}"
      echo "$function_data" | fold -w 80 -s
      echo ""
      return
    fi

    # Decode known functions
    if [[ "$function_selector" == "$(cast sig $TRANSFER_SIG)" ]]; then
      local decoded
      decoded=$(cast decode-calldata "$TRANSFER_SIG" "$function_data")
      local to
      to=$(echo "$decoded" | head -1)
      local amount_line
      amount_line=$(echo "$decoded" | tail -1)
      local amount
      amount=$(echo "$amount_line" | sed -E 's/([0-9]+)( \[[^]]+\])?/\1/')
      local formatted_amount
      formatted_amount=$(parse_token_amount "$amount" "$target_decimals")

      printf "  ${BOLD}%-8s${RESET}: %s\n" "To" "$to"
      printf "  ${BOLD}%-8s${RESET}: %s\n" "Amount" "$formatted_amount"
      if [[ "$target_address_name" != "UNKNOWN" ]]; then
        printf "  ${BOLD}%-8s${RESET}: %s\n" "Token" "$target_address_name"
      fi
      echo ""

    elif [[ "$function_selector" == "$(cast sig $TRANSFER_FROM_SIG)" ]]; then
      local decoded
      decoded=$(cast decode-calldata "$TRANSFER_FROM_SIG" "$function_data")
      local from
      from=$(echo "$decoded" | head -1)
      local to
      to=$(echo "$decoded" | sed -n '2p')
      local amount_line
      amount_line=$(echo "$decoded" | tail -1)
      local amount
      amount=$(echo "$amount_line" | sed -E 's/([0-9]+)( \[[^]]+\])?/\1/')
      local formatted_amount
      formatted_amount=$(parse_token_amount "$amount" "$target_decimals")

      printf "  ${BOLD}%-8s${RESET}: %s\n" "From"   "$from"
      printf "  ${BOLD}%-8s${RESET}: %s\n" "To"     "$to"
      printf "  ${BOLD}%-8s${RESET}: %s\n" "Amount" "$formatted_amount"
      if [[ "$target_address_name" != "UNKNOWN" ]]; then
        printf "  ${BOLD}%-8s${RESET}: %s\n" "Token" "$target_address_name"
      fi
      echo ""

    elif [[ "$function_selector" == "$(cast sig $APPROVE_SIG)" ]]; then
      local decoded
      decoded=$(cast decode-calldata "$APPROVE_SIG" "$function_data")
      local spender
      spender=$(echo "$decoded" | head -1)
      local amount_line
      amount_line=$(echo "$decoded" | tail -1)
      local amount
      amount=$(echo "$amount_line" | sed -E 's/([0-9]+)( \[[^]]+\])?/\1/')
      local formatted_amount
      formatted_amount=$(parse_token_amount "$amount" "$target_decimals")

      printf "  ${BOLD}%-8s${RESET}: %s\n" "Spender" "$spender"
      printf "  ${BOLD}%-8s${RESET}: %s\n" "Amount"  "$formatted_amount"
      if [[ "$target_address_name" != "UNKNOWN" ]]; then
        printf "  ${BOLD}%-8s${RESET}: %s\n" "Token" "$target_address_name"
      fi
      echo ""

    elif [[ "$function_selector" == "$(cast sig $INCREASE_ALLOWANCE_SIG)" ]]; then
      local decoded
      decoded=$(cast decode-calldata "$INCREASE_ALLOWANCE_SIG" "$function_data")
      local spender
      spender=$(echo "$decoded" | head -1)
      local amount_line
      amount_line=$(echo "$decoded" | tail -1)
      local amount
      amount=$(echo "$amount_line" | sed -E 's/([0-9]+)( \[[^]]+\])?/\1/')
      local formatted_amount
      formatted_amount=$(parse_token_amount "$amount" "$target_decimals")

      printf "  ${BOLD}%-8s${RESET}: %s\n" "Spender"  "$spender"
      printf "  ${BOLD}%-8s${RESET}: %s\n" "Increase" "$formatted_amount"
      if [[ "$target_address_name" != "UNKNOWN" ]]; then
        printf "  ${BOLD}%-8s${RESET}: %s\n" "Token" "$target_address_name"
      fi
      echo ""

    elif [[ "$function_selector" == "$(cast sig $DECREASE_ALLOWANCE_SIG)" ]]; then
      local decoded
      decoded=$(cast decode-calldata "$DECREASE_ALLOWANCE_SIG" "$function_data")
      local spender
      spender=$(echo "$decoded" | head -1)
      local amount_line
      amount_line=$(echo "$decoded" | tail -1)
      local amount
      amount=$(echo "$amount_line" | sed -E 's/([0-9]+)( \[[^]]+\])?/\1/')
      local formatted_amount
      formatted_amount=$(parse_token_amount "$amount" "$target_decimals")

      printf "  ${BOLD}%-8s${RESET}: %s\n" "Spender"  "$spender"
      printf "  ${BOLD}%-8s${RESET}: %s\n" "Decrease" "$formatted_amount"
      if [[ "$target_address_name" != "UNKNOWN" ]]; then
        printf "  ${BOLD}%-8s${RESET}: %s\n" "Token" "$target_address_name"
      fi
      echo ""

    elif [[ "$function_selector" == "$(cast sig $APPROVE_HASH_SIG)" ]]; then
      local decoded
      decoded=$(cast decode-calldata "$APPROVE_HASH_SIG" "$function_data")
      local hash
      hash=$(echo "$decoded" | head -1)

      printf "  ${BOLD}%-8s${RESET}: %s\n" "Hash" "$hash"
      echo ""

    else
      # Unknown function - just show raw data
      echo -e "${DIM}Raw Data:${RESET}"
      echo "$function_data" | fold -w 80 -s
      echo ""
    fi
  fi
}

#------------------------------------------------------------------------------
# Parses top-level transaction data, calling parse_function_data for the actual
# decoding of the "data" field.
# Globals:
#   BOLD, RESET, YELLOW
# Arguments:
#   $1 - The path to the transaction file.
#------------------------------------------------------------------------------
parse_transaction_data() {
  local transaction_file="$1"

  local safe chain to value data operation nonce
  safe=$(get_transaction_field "$transaction_file" "safe")
  chain=$(get_transaction_field "$transaction_file" "chain")
  to=$(get_transaction_field "$transaction_file" "to")
  value=$(get_transaction_field "$transaction_file" "value")
  data=$(get_transaction_field "$transaction_file" "data")
  operation=$(get_transaction_field "$transaction_file" "operation")
  nonce=$(get_transaction_field "$transaction_file" "nonce")

  local operation_name
  if [[ "$operation" == "0" ]]; then
    operation_name="Call"
  elif [[ "$operation" == "1" ]]; then
    operation_name="DelegateCall"
  else
    echo -e "${YELLOW}Error: Unsupported operation \"${operation}\".${RESET}" >&2
    exit 1
  fi

  echo ""
  print_heading "BASIC TRANSACTION DETAILS"
  printf "${BOLD}%-14s${RESET}: %s\n" "Safe"       "$safe"
  printf "${BOLD}%-14s${RESET}: %s\n" "Chain ID"   "$chain"
  printf "${BOLD}%-14s${RESET}: %s\n" "Target"     "$to"
  printf "${BOLD}%-14s${RESET}: %s\n" "ETH Value"  "$value"
  printf "${BOLD}%-14s${RESET}: %s\n" "Nonce"      "$nonce"
  printf "${BOLD}%-14s${RESET}: %s\n" "Operation"  "$operation_name"
  echo ""

  parse_function_data "$to" "$data" 1 false
}

#------------------------------------------------------------------------------
# MAIN ENTRY POINT
#------------------------------------------------------------------------------
main() {
  local transaction_file=""

  # Parse command line arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --tx)
        transaction_file="$2"
        shift 2
        ;;
      -h|--help)
        usage
        ;;
      *)
        echo "Unknown option: $1" >&2
        usage
        ;;
    esac
  done

  if [[ -z "$transaction_file" ]]; then
    echo "Error: Transaction file not specified. Use --tx <file>."
    echo ""
    usage
  fi

  check_transaction_file "$transaction_file"

  # Generate the domain hash
  local domain_hash
  domain_hash=$(make_domain_hash "$transaction_file")

  # Generate the message hash
  local message_hash
  message_hash=$(make_message_hash "$transaction_file")

  # Parse and display transaction data
  parse_transaction_data "$transaction_file"

  # Print domain and message hashes
  print_heading "HASHES"
  printf "${BOLD}%-12s${RESET}: %s\n" "Domain Hash"  "$domain_hash"
  printf "${BOLD}%-12s${RESET}: %s\n" "Message Hash" "$message_hash"
  echo ""

  # Print verification instructions
  print_heading "VERIFICATION INSTRUCTIONS"
  echo -e "${BOLD}1. Transaction details should EXACTLY MATCH what you expect to see.${RESET}"
  echo -e "${BOLD}2. Domain and message hashes should EXACTLY MATCH other machines.${RESET}"
  echo -e "${BOLD}3. Your hardware wallet should show you the EXACT SAME HASHES.${RESET}"
  echo -e "${BOLD}4. WHEN IN DOUBT, ASK FOR HELP.${RESET}"
  echo ""
}

main "$@"
