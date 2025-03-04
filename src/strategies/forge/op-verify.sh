#!/usr/bin/env bash

# Simplified Safe transaction validation script. Generally inspired by
# https://github.com/pcaversaccio/safe-tx-hashes-util with several important
# differences:
#   - Only uses the cast binary (no other foundry tools)
#   - Always takes a transaction file as input (no Safe API)
#   - No support for older Safe transaction formats

# Safe-specific hashes and signatures
readonly DOMAIN_SEPARATOR_TYPEHASH="0x47e79534a245952e8b16893a336b85a3d9ea9fa8c573f3d803afb92a79469218"
readonly SAFE_TX_TYPEHASH="0xbb8310d486368db6bd6f849402fdd73ad53d316b5a4b2644ad6efe0f941286d8"
readonly DOMAIN_SEPARATOR_SIG="fn(bytes32,uint256,address)"
readonly SAFE_TX_SIG="fn(bytes32,address,uint256,bytes32,uint8,uint256,uint256,uint256,address,address,uint256)"

# Known function signatures.
readonly TRANSFER_SIG="transfer(address,uint256)"
readonly TRANSFER_FROM_SIG="transferFrom(address,address,uint256)"
readonly APPROVE_SIG="approve(address,uint256)"
readonly INCREASE_ALLOWANCE_SIG="increaseAllowance(address,uint256)"
readonly DECREASE_ALLOWANCE_SIG="decreaseAllowance(address,uint256)"
readonly APPROVE_HASH_SIG="approveHash(bytes32)"
readonly AGGREGATE3_SIG="aggregate3((address,bool,bytes)[])"

# Known addresses.
readonly MULTICALL3_ADDRESS="0xcA11bde05977b3631167028862bE2a173976CA11"
readonly USDC_MAINNET_ADDRESS="0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
readonly OP_TOKEN_ADDRESS="0x4200000000000000000000000000000000000042"

# Add these color/format definitions near the top of your script:
BOLD="\e[1m"
DIM="\e[2m"
RESET="\e[0m"
BLUE="\e[34m"
CYAN="\e[36m"
GREEN="\e[32m"
MAGENTA="\e[35m"
YELLOW="\e[33m"

# Print a heading with a thick line
print_heading() {
  local heading="$1"
  echo -e "${BOLD}${CYAN}${heading}${RESET}"
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
}

# Lighter divider line
print_divider() {
  echo -e "${DIM}${MAGENTA}──────────────────━━━━━━━───────────────────────────────────────────────────────${RESET}"
}

# Checks that a transaction file exists.
#
# @param $1 The transaction file to check.
check_transaction_file() {
  local transaction_file="$1"

  # Make sure the transaction file actually exists.
  if [[ ! -f "$transaction_file" ]]; then
    echo "Error: Transaction file '$transaction_file' not found"
    exit 1
  fi
}

# Gets a field from a transaction file.
#
# @param $1 The transaction file to get the field from.
# @param $2 The field to get.
get_transaction_field() {
  local transaction_file="$1"
  local field="$2"

  # Make sure the transaction file actually exists.
  check_transaction_file "$transaction_file"

  # Parse the field from the transaction file.
  local value=$(
    grep -o "\"$field\"[[:space:]]*:[[:space:]]*[^,}]*" "$transaction_file" | 
    sed -E 's/"'"$field"'"[[:space:]]*:[[:space:]]*//' | 
    sed -E 's/^"(.*)"$/\1/' |
    sed -E 's/^[[:space:]]*|[[:space:]]*$//'
  )

  # If value is empty, the field might not exist or has a different format
  if [[ -z "$value" ]]; then
    echo "Error: Field '$field' not found in transaction file '$transaction_file'"
    exit 1
  fi

  # Return the value.
  echo "$value"
}

# Generates the message hash for a Safe transaction.
#
# @param $1 The transaction file to generate the message hash for.
make_message_hash() {
  local transaction_file="$1"

  # Make sure the transaction file actually exists.
  check_transaction_file "$transaction_file"

  # Grab relevant transaction fields.
  local safe=$(get_transaction_field "$transaction_file" "safe")
  local chain=$(get_transaction_field "$transaction_file" "chain")
  local to=$(get_transaction_field "$transaction_file" "to")
  local value=$(get_transaction_field "$transaction_file" "value")
  local data=$(get_transaction_field "$transaction_file" "data")
  local operation=$(get_transaction_field "$transaction_file" "operation")
  local safe_tx_gas=$(get_transaction_field "$transaction_file" "safe_tx_gas")
  local base_gas=$(get_transaction_field "$transaction_file" "base_gas")
  local gas_price=$(get_transaction_field "$transaction_file" "gas_price")
  local gas_token=$(get_transaction_field "$transaction_file" "gas_token")
  local refund_receiver=$(get_transaction_field "$transaction_file" "refund_receiver")
  local nonce=$(get_transaction_field "$transaction_file" "nonce")

  # Generate the message hash input.
  local message_hash_input=$(
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

  # Generate the message hash.
  local message_hash=$(cast keccak "$message_hash_input")

  # Return the message hash.
  echo "$message_hash"
}

# Generate the domain hash for a Safe transaction.
#
# @param $1 The transaction file to generate the domain hash for.
make_domain_hash() {
  local transaction_file="$1"

  # Make sure the transaction file actually exists.
  check_transaction_file "$transaction_file"

  # Grab relevant transaction fields.
  local safe=$(get_transaction_field "$transaction_file" "safe")
  local chain=$(get_transaction_field "$transaction_file" "chain")

  # Generate the domain hash input.
  local domain_hash_input=$(
    cast abi-encode "$DOMAIN_SEPARATOR_SIG" \
    "$DOMAIN_SEPARATOR_TYPEHASH" \
    "$chain" \
    "$safe"
  )

  # Generate the domain hash.
  local domain_hash=$(cast keccak "$domain_hash_input")

  # Return the domain hash.
  echo "$domain_hash"
}

parse_token_amount() {
  local amount="$1"
  local decimals="$2"

  if [[ "$decimals" != "" ]]; then
    local amount=$(echo "scale=6; $amount / 10^$decimals" | bc)
  else
    local amount=$(echo "$amount (UNKNOWN DECIMALS)")
  fi

  echo "$amount"
}

parse_function_data() {
  local target_address="$1"
  local function_data="$2"
  local tx_number="${3:-1}"     # Transaction index label
  local is_nested="${4:-false}" # Whether this is nested inside a Multicall

  # Check if this is a known address
  local target_address_name=""
  local target_address_network=""
  local target_decimals=""
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

  # Get the function selector (first 10 hex characters).
  local function_selector="${function_data:0:10}"
  local function_name="Unknown"

  # Determine known function name by comparing against known signatures.
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

  # Disallow nested Multicall3 calls.
  if [[ "$is_nested" == "true" && "$target_address" == "$MULTICALL3_ADDRESS" && "$function_selector" == "$(cast sig $AGGREGATE3_SIG)" ]]; then
    echo -e "${YELLOW}ERROR: Nested Multicall3 detected. This is not supported.${RESET}"
    exit 1
  fi

  if [[ "$target_address" == "$MULTICALL3_ADDRESS" && "$function_selector" == "$(cast sig $AGGREGATE3_SIG)" ]]; then
    # Use cast decode to parse the aggregate3 calls thoroughly
    local decoded_calls
    decoded_calls=$(cast decode-calldata "$AGGREGATE3_SIG" "$function_data")

    # Count subcalls by scanning array items
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
    printf "${GREEN}%-10s${RESET}: %s (%s)\n" "Target"  "$target_address" "$target_address_name"
    printf "${GREEN}%-10s${RESET}: %s\n"      "Function" "$function_name"
    echo ""

    # If the selector is too short, just show the raw data
    if [[ "${#function_selector}" -lt 10 ]]; then
      echo -e "${DIM}Raw Data:${RESET}"
      echo "$function_data" | fold -w 80 -s
      echo ""
      return
    fi

    # Handle known functions with decode
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

      printf "  ${BOLD}%-8s${RESET}: %s\n" "From" "$from"
      printf "  ${BOLD}%-8s${RESET}: %s\n" "To"   "$to"
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

      printf "  ${BOLD}%-8s${RESET}: %s\n" "Spender" "$spender"
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

      printf "  ${BOLD}%-8s${RESET}: %s\n" "Spender" "$spender"
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
      # Unknown function data
      echo -e "${DIM}Raw Data:${RESET}"
      echo "$function_data" | fold -w 80 -s
      echo ""
    fi
  fi
}

parse_transaction_data() {
  local transaction_file="$1"

  # Grab relevant transaction fields.
  local safe=$(get_transaction_field "$transaction_file" "safe")
  local chain=$(get_transaction_field "$transaction_file" "chain")
  local to=$(get_transaction_field "$transaction_file" "to")
  local value=$(get_transaction_field "$transaction_file" "value")
  local data=$(get_transaction_field "$transaction_file" "data")
  local operation=$(get_transaction_field "$transaction_file" "operation")
  local nonce=$(get_transaction_field "$transaction_file" "nonce")

  # Error if operation is unsupported.
  local operation_name
  if [[ "$operation" == "0" ]]; then
    operation_name="Call"
  elif [[ "$operation" == "1" ]]; then
    operation_name="DelegateCall"
  else
    echo -e "${YELLOW}Error: Unsupported operation${RESET}"
    exit 1
  fi

  echo ""
  print_heading "BASIC TRANSACTION DETAILS"
  printf "${BOLD}%-14s${RESET}: %s\n" "Safe"         "$safe"
  printf "${BOLD}%-14s${RESET}: %s\n" "Chain ID"     "$chain"
  printf "${BOLD}%-14s${RESET}: %s\n" "Target"       "$to"
  printf "${BOLD}%-14s${RESET}: %s\n" "ETH Value"    "$value"
  printf "${BOLD}%-14s${RESET}: %s\n" "Nonce"        "$nonce"
  printf "${BOLD}%-14s${RESET}: %s\n" "Operation"    "$operation_name"
  echo ""

  # Parse the function data for the transaction.
  parse_function_data "$to" "$data" 1 false
}

# Main entrypoint.
#
# @param $@ The command line arguments.
main() {
  local transaction_file=""

  # Parse command line arguments.
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --tx)
        transaction_file="$2"
        shift 2
        ;;
      *)
        echo "Unknown option: $1"
        exit 1
        ;;
    esac
  done

  # Make sure the transaction file is specified.
  if [[ -z "$transaction_file" ]]; then
    echo "Error: Transaction file not specified. Use --tx <file>"
    exit 1
  fi

  # Make sure the transaction file actually exists.
  check_transaction_file "$transaction_file"

  # Generate the domain hash.
  local domain_hash=$(make_domain_hash "$transaction_file")

  # Generate the message hash.
  local message_hash=$(make_message_hash "$transaction_file")

  # Parse the transaction data.
  parse_transaction_data "$transaction_file"

  # Print the domain and message hashes.
  print_heading "HASHES"
  printf "${BOLD}%-12s${RESET}: %s\n" "Domain Hash" "$domain_hash"
  printf "${BOLD}%-12s${RESET}: %s\n" "Message Hash" "$message_hash"
  echo ""

  # Print verification instructions.
  print_heading "VERIFICATION INSTRUCTIONS"
  echo -e "${BOLD}1. Transaction details should EXACTLY MATCH what you expect to see${RESET}"
  echo -e "${BOLD}2. Domain and message hashes should EXACTLY MATCH other machines${RESET}"
  echo -e "${BOLD}3. Hardware wallet should show you the EXACT SAME HASHES${RESET}"
  echo -e "${BOLD}4. WHEN IN DOUBT, ASK FOR HELP${RESET}"
  echo ""
}

# Entrypoint.
main "$@"
