#!/usr/bin/env bash
#
# Safe transaction validation script, returning JSON instead of printing details.
# Use this script by calling it with --tx <transaction.json>, so that it outputs
# fully detailed JSON describing the transaction. For example:
#
#   ./op-verify.sh --tx path/to/tx.json
#
# This output can then be piped into a separate script or command for further
# display or verification.
#

set -Eeuo pipefail
IFS=$'\n\t'

###############################################################################
# GLOBAL CONSTANTS & TEXT (not used for printing but for internal logic)
###############################################################################
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

# Known addresses & decimals
readonly MULTICALL3_ADDRESS="0xcA11bde05977b3631167028862bE2a173976CA11"
readonly USDC_MAINNET_ADDRESS="0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
readonly OP_TOKEN_ADDRESS="0x4200000000000000000000000000000000000042"

###############################################################################
# FUNCTIONS
###############################################################################

#------------------------------------------------------------------------------
# Print usage instructions.
#------------------------------------------------------------------------------
usage() {
  cat <<EOF
Usage: $0 --tx <path-to-transaction-file.json>

Validates a Safe transaction JSON file and outputs JSON describing the transaction.

Options:
  --tx <file>   Path to the Safe transaction JSON file.
  -h, --help    Show this help.

Examples:
  $0 --tx my-transaction.json
EOF
  exit 1
}

#------------------------------------------------------------------------------
# Checks that a transaction file exists; exits with error if not.
#------------------------------------------------------------------------------
check_transaction_file() {
  local transaction_file="$1"
  if [[ ! -f "$transaction_file" ]]; then
    echo "Error: Transaction file '$transaction_file' not found." >&2
    exit 1
  fi
}

#------------------------------------------------------------------------------
# Extracts a specified JSON field from the transaction file using jq.
# Returns the value or exits if not found.
#------------------------------------------------------------------------------
get_transaction_field() {
  local transaction_file="$1"
  local field="$2"

  local value
  value=$(jq -r ".$field // empty" "$transaction_file")

  if [[ -z "$value" ]]; then
    echo "Error: Field '$field' not found in transaction file '$transaction_file'." >&2
    exit 1
  fi

  echo "$value"
}

#------------------------------------------------------------------------------
# Generates the message hash for a Safe transaction (keccak over typed data).
# Returns the keccak256 as a hex string.
#------------------------------------------------------------------------------
make_message_hash() {
  local transaction_file="$1"

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

  cast keccak "$message_hash_input"
}

#------------------------------------------------------------------------------
# Generates the domain hash for a Safe transaction (keccak over typed data).
# Returns the keccak256 as a hex string.
#------------------------------------------------------------------------------
make_domain_hash() {
  local transaction_file="$1"

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

  cast keccak "$domain_hash_input"
}

#------------------------------------------------------------------------------
# Converts a raw token amount into a decimal string using the known decimals,
# or returns it plainly if decimals are unknown.
#------------------------------------------------------------------------------
parse_token_amount() {
  local amount_in_wei="$1"
  local decimals="$2"

  if [[ -n "$decimals" ]]; then
    # Using bc for float arithmetic
    echo "$(bc <<< "scale=6; $amount_in_wei / 10^$decimals")"
  else
    echo "$amount_in_wei (UNKNOWN DECIMALS)"
  fi
}

#------------------------------------------------------------------------------
# Attempt to decode the given calldata based on known function selectors.
# Returns JSON describing the function name, addresses, amounts, etc.
#
# If the function is a Multicall3 aggregate call, we recursively parse and
# return an array of subcalls in JSON.
#------------------------------------------------------------------------------
parse_function_data() {
  local target_address="$1"
  local function_data="$2"
  local is_nested="${3:-false}"

  local function_selector="${function_data:0:10}"

  # Identify known token addresses & decimals
  local target_address_name="UNKNOWN CONTRACT"
  local decimals=""
  if [[ "$target_address" == "$USDC_MAINNET_ADDRESS" ]]; then
    target_address_name="USDC"
    decimals="6"
  elif [[ "$target_address" == "$OP_TOKEN_ADDRESS" ]]; then
    target_address_name="OP TOKEN"
    decimals="18"
  elif [[ "$target_address" == "$MULTICALL3_ADDRESS" ]]; then
    target_address_name="MULTICALL3"
  fi

  # Initial fallback structure
  local function_name="UNKNOWN"
  local extra_json='{}'  # appended details about the function call

  # If the calldata is too short, return minimal data
  if [[ "${#function_selector}" -lt 10 ]]; then
    # Return a simple JSON structure for "unknown"
    jq -n \
      --arg targetAddress "$target_address" \
      --arg targetName "$target_address_name" \
      --arg rawData "$function_data" \
      '{
        target: $targetAddress,
        targetName: $targetName,
        functionSelector: "INSUFFICIENT_DATA",
        rawData: $rawData
      }'
    return
  fi

  # Identify known function by signature
  case "$function_selector" in
    "$(cast sig $TRANSFER_SIG)")
      function_name="$TRANSFER_SIG"
      local decoded
      decoded="$(cast decode-calldata "$TRANSFER_SIG" "$function_data")" || true
      # Usually "decoded" has 2 lines: recipient, amount
      local recipient="$(echo "$decoded" | sed -n '1p')"
      local raw_amount="$(echo "$decoded" | sed -n '2p')"
      local amount_cleaned="$(sed -E 's/([0-9]+)( \[[^]]+\])?/\1/' <<< "$raw_amount")"
      local amount_decimals="$(parse_token_amount "$amount_cleaned" "$decimals")"

      extra_json="$(
        jq -n \
          --arg recipient "$recipient" \
          --arg amount "$amount_decimals" \
          '{
             recipient: $recipient,
             amount: $amount
           }'
      )"
      ;;
    "$(cast sig $TRANSFER_FROM_SIG)")
      function_name="$TRANSFER_FROM_SIG"
      local decoded
      decoded="$(cast decode-calldata "$TRANSFER_FROM_SIG" "$function_data")" || true
      # Usually "decoded" has 3 lines: from, to, amount
      local from_addr="$(echo "$decoded" | sed -n '1p')"
      local to_addr="$(echo "$decoded" | sed -n '2p')"
      local raw_amount="$(echo "$decoded" | sed -n '3p')"
      local amount_cleaned="$(sed -E 's/([0-9]+)( \[[^]]+\])?/\1/' <<< "$raw_amount")"
      local amount_decimals="$(parse_token_amount "$amount_cleaned" "$decimals")"

      extra_json="$(
        jq -n \
          --arg from "$from_addr" \
          --arg to "$to_addr" \
          --arg amount "$amount_decimals" \
          '{
             from: $from,
             to: $to,
             amount: $amount
           }'
      )"
      ;;
    "$(cast sig $APPROVE_SIG)")
      function_name="$APPROVE_SIG"
      local decoded
      decoded="$(cast decode-calldata "$APPROVE_SIG" "$function_data")" || true
      # Usually: spender, amount
      local spender="$(echo "$decoded" | sed -n '1p')"
      local raw_amount="$(echo "$decoded" | sed -n '2p')"
      local amount_cleaned="$(sed -E 's/([0-9]+)( \[[^]]+\])?/\1/' <<< "$raw_amount")"
      local amount_decimals="$(parse_token_amount "$amount_cleaned" "$decimals")"

      extra_json="$(
        jq -n \
          --arg spender "$spender" \
          --arg amount "$amount_decimals" \
          '{
             spender: $spender,
             amount: $amount
           }'
      )"
      ;;
    "$(cast sig $INCREASE_ALLOWANCE_SIG)")
      function_name="$INCREASE_ALLOWANCE_SIG"
      local decoded
      decoded="$(cast decode-calldata "$INCREASE_ALLOWANCE_SIG" "$function_data")" || true
      # Usually: spender, addedValue
      local spender="$(echo "$decoded" | sed -n '1p')"
      local raw_amount="$(echo "$decoded" | sed -n '2p')"
      local amount_cleaned="$(sed -E 's/([0-9]+)( \[[^]]+\])?/\1/' <<< "$raw_amount")"
      local amount_decimals="$(parse_token_amount "$amount_cleaned" "$decimals")"

      extra_json="$(
        jq -n \
          --arg spender "$spender" \
          --arg amount "$amount_decimals" \
          '{
             spender: $spender,
             amount: $amount
           }'
      )"
      ;;
    "$(cast sig $DECREASE_ALLOWANCE_SIG)")
      function_name="$DECREASE_ALLOWANCE_SIG"
      local decoded
      decoded="$(cast decode-calldata "$DECREASE_ALLOWANCE_SIG" "$function_data")" || true
      # Usually: spender, subtractedValue
      local spender="$(echo "$decoded" | sed -n '1p')"
      local raw_amount="$(echo "$decoded" | sed -n '2p')"
      local amount_cleaned="$(sed -E 's/([0-9]+)( \[[^]]+\])?/\1/' <<< "$raw_amount")"
      local amount_decimals="$(parse_token_amount "$amount_cleaned" "$decimals")"

      extra_json="$(
        jq -n \
          --arg spender "$spender" \
          --arg amount "$amount_decimals" \
          '{
             spender: $spender,
             amount: $amount
           }'
      )"
      ;;
    "$(cast sig $APPROVE_HASH_SIG)")
      function_name="$APPROVE_HASH_SIG"
      local decoded
      decoded="$(cast decode-calldata "$APPROVE_HASH_SIG" "$function_data")" || true
      # Usually: single line with the hash
      local hash_val="$(echo "$decoded" | sed -n '1p')"

      extra_json="$(
        jq -n \
          --arg hash "$hash_val" \
          '{ hash: $hash }'
      )"
      ;;
    "$(cast sig $AGGREGATE3_SIG)")
      function_name="$AGGREGATE3_SIG"
      # If nested inside a MULTICALL, block further nesting to avoid complexity
      if [[ "$is_nested" == "true" ]]; then
        # Return error JSON or exit. We'll choose to exit for clarity.
        echo "Error: Nested Multicall3 detected, not supported." >&2
        exit 1
      fi

      # If this isn't a call to the multicall3 contract, we'll just return the
      # function name & data
      if [[ "$target_address" != "$MULTICALL3_ADDRESS" ]]; then
        extra_json="$(
          jq -n \
            --arg functionName "$function_name" \
            --arg functionData "$function_data" \
            '{ functionName: $functionName, functionData: $functionData }'
        )"
      else
        # Decode the subcalls
        local decoded_calls
        decoded_calls="$(cast decode-calldata "$AGGREGATE3_SIG" "$function_data")"

        # We'll parse them out with a while loop & store them in a JSON array
        # Layout is: [ (targetAddr, bool, bytes), (targetAddr, bool, bytes), ... ]
        # We can attempt a simpler parse strategy with a custom approach or a robust parser.
        # For demonstration, let's do a basic approach with BASH regex.

        # Remove outer square brackets
        local calls_str="${decoded_calls#\[}"
        calls_str="${calls_str%\]}"

        # We'll build an array string that we feed to jq
        local subcalls_json='[]'

        while [[ "$calls_str" =~ \(([^,]+),\ ([^,]+),\ ([^\)]+)\)(.*) ]]; do
          local call_target="${BASH_REMATCH[1]}"
          local call_allowFailure="${BASH_REMATCH[2]}"
          local call_data="${BASH_REMATCH[3]}"
          calls_str="${BASH_REMATCH[4]}"

          # Remove leading comma and space if present
          if [[ "$calls_str" =~ ^,\ (.*) ]]; then
            calls_str="${BASH_REMATCH[1]}"
          fi

          # Recursively parse each subcall
          local sub_json
          sub_json="$(parse_function_data "$call_target" "$call_data" "true")"

          # Merge into array
          subcalls_json="$(
            jq -n \
              --argjson arr "$subcalls_json" \
              --argjson item "$sub_json" \
              '($arr + [ $item ])'
          )"
        done

        extra_json="$(
          jq -n \
            --argjson nestedCalls "$subcalls_json" \
            '{
              subcalls: $nestedCalls
            }'
        )"
      fi
      ;;
    *)
      # Unknown function - store the rawData for reference
      function_name="UNKNOWN"
      extra_json="$(
        jq -n \
          --arg data "$function_data" \
          '{ rawData: $data }'
      )"
      ;;
  esac

  # Finally, build the JSON for this call
  jq -n \
    --arg targetAddress "$target_address" \
    --arg targetAddressName "$target_address_name" \
    --arg fnName "$function_name" \
    --arg selector "$function_selector" \
    --arg functionData "$function_data" \
    --argjson extra "$extra_json" \
    '{
      target: $targetAddress,
      targetName: $targetAddressName,
      functionName: $fnName,
      functionSelector: $selector,
      functionData: $functionData,
      parsedData: $extra
    }'
}

#------------------------------------------------------------------------------
# Reads the transaction file, extracts top-level fields, and returns JSON
# describing the entire transaction (including any subcalls).
#------------------------------------------------------------------------------
create_transaction_json() {
  local transaction_file="$1"

  local safe chain to value data operation nonce
  safe="$(get_transaction_field "$transaction_file" "safe")"
  chain="$(get_transaction_field "$transaction_file" "chain")"
  to="$(get_transaction_field "$transaction_file" "to")"
  value="$(get_transaction_field "$transaction_file" "value")"
  data="$(get_transaction_field "$transaction_file" "data")"
  operation="$(get_transaction_field "$transaction_file" "operation")"
  nonce="$(get_transaction_field "$transaction_file" "nonce")"

  # Operation ID to string
  local operation_name="Unsupported"
  if [[ "$operation" == "0" ]]; then
    operation_name="Call"
  elif [[ "$operation" == "1" ]]; then
    operation_name="DelegateCall"
  fi

  # build the JSON for the top-level function call
  local top_level_call
  top_level_call="$(parse_function_data "$to" "$data" "false")"

  jq -n \
    --arg safe "$safe" \
    --arg chain "$chain" \
    --arg to "$to" \
    --arg value "$value" \
    --arg operation "$operation_name" \
    --arg nonce "$nonce" \
    --argjson calls "$top_level_call" \
    '{
      safe: $safe,
      chain: $chain,
      to: $to,
      value: $value,
      operation: $operation,
      nonce: $nonce,
      call: $calls
    }'
}

#------------------------------------------------------------------------------
# MAIN: Parse arguments, produce final JSON with domainHash, messageHash, etc.
#------------------------------------------------------------------------------
main() {
  local transaction_file=""

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
    usage
  fi

  check_transaction_file "$transaction_file"

  # Generate necessary details
  local domain_hash
  local message_hash
  domain_hash="$(make_domain_hash "$transaction_file")"
  message_hash="$(make_message_hash "$transaction_file")"

  local tx_json
  tx_json="$(create_transaction_json "$transaction_file")"

  # Build final combined JSON (domainHash + messageHash + transaction)
  jq -n \
    --arg domainHash "$domain_hash" \
    --arg messageHash "$message_hash" \
    --argjson transaction "$tx_json" \
    '{
      domainHash: $domainHash,
      messageHash: $messageHash,
      transaction: $transaction
    }'
}

main "$@"
