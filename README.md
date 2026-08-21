# op-txverify

op-txverify verifies [Safe](https://app.safe.global) transactions, as a command-line utility and as
a page at [op-txverify.optimism.io](https://op-txverify.optimism.io). It helps users validate Safe
multisig transactions by:

1. Parsing and presenting transactions in a clearly understandable manner
2. Identifying common addresses and contracts that users interact with
3. Verifying a transaction on a phone, independently of the computer being signed from
4. Simplifying access by allowing users to input transactions via QR codes

## Verifying on a phone

The page at [op-txverify.optimism.io](https://op-txverify.optimism.io) shows the two EIP-712 hashes
a hardware wallet displays, plus the decoded calldata, without involving the computer running the
Safe UI. That matters because `op-txverify` on the signing computer proves nothing if that computer
is compromised — it would report whatever the attacker wants.

1. Open the page on a phone, ideally on cellular rather than the same network as the computer.
2. Enter either the `safeTxHash`, or the Safe address, nonce, and network. Prefer the latter: those
   three values come from your own knowledge of the transaction, whereas a `safeTxHash` is usually
   handed over by the computer being distrusted. The entries are remembered for next time.
3. Compare the domain hash and message hash to your device, and the decoded call to what was
   actually approved.

Two properties are worth understanding:

- **The hashes come from the Safe contract**, read with `encodeTransactionData` over an RPC, not from
  the page's own arithmetic. The WebAssembly module recomputes them independently, so an RPC
  returning a wrong preimage shows up as a visible mismatch.
- **The decode is `core/parsing.go` compiled to WebAssembly**, so there is one decoder
  implementation rather than a JavaScript reimplementation that could drift from the CLI.

Supported chains are Ethereum, OP Mainnet, Base, and Sepolia. Safe 1.5.0 and later removed
`encodeTransactionData`, so the page refuses those rather than failing obscurely.

The page prints a build commit and WebAssembly digest at the bottom. That is a **version indicator,
not an integrity check** — serve-time HTML injection never touches the artifact the digest
describes, so a matching digest does not prove the page you loaded is the page that was built. To
confirm the hashes without trusting the page at all, read `encodeTransactionData` yourself on a
block explorer's Read as Proxy tab, using the parameters the page displays.

## QR code scanning

The QR scanning functionality provided by `op-txverify` allows you to verify Safe transactions by scanning QR codes displayed on a web interface. This is especially useful for air-gapped verification where transmitting data to the verification device over bluetooth or USB is not desirable.

To use the QR code scanner:

1. Open [https://op-txverify.optimism.io](https://op-txverify.optimism.io) on your mobile device.
2. Paste the transaction hash or Safe UI link of the Safe transaction you want to verify.
3. Run the following command on your verification device:
    ```bash
    op-txverify qr
    ```
4. A browser window will automatically open with the QR scanner interface. It is served on
   loopback only, so no other host on the network can submit a scanner result.
5. Display the QR codes on your mobile device to the QR scanner.
6. After successful scanning, op-txverify will verify the transaction and display the results.

## Installation

`op-txverify` is not distributed as a release binary. Build the latest version
from source before each signing ceremony.

Prerequisites:

- Go 1.23 or later
- Git

Install the latest `main` directly:

```bash
go install github.com/ethereum-optimism/op-txverify/cmd/op-txverify@main
```

Make sure your Go binary directory is on your `PATH`. For most local Go
installations, this is:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

Or clone and build explicitly:

1. Clone the repository:
    ```bash
    git clone https://github.com/ethereum-optimism/op-txverify.git
    cd op-txverify
    ```
1. Build the binary:
    ```bash
    go build -trimpath -o ./bin/op-txverify ./cmd/op-txverify
    ```
1. Install it somewhere on your `PATH`:
    ```bash
    sudo mv ./bin/op-txverify /usr/local/bin/op-txverify
    ```

Confirm the installed binary:

```bash
op-txverify --version
```
