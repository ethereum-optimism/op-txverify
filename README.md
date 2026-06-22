# op-txverify

op-txverify is a command-line utility for verifying [Safe](https://app.safe.global) transactions. It helps users validate Safe multisig transactions by:

1. Parsing and presenting transactions in a clearly understandable manner
2. Identifying common addresses and contracts that users interact with
3. Simplifying access by allowing users to input transactions via QR codes

## QR code scanning

The QR scanning functionality provided by `op-txverify` allows you to verify Safe transactions by scanning QR codes displayed on a web interface. This is especially useful for air-gapped verification where transmitting data to the verification device over bluetooth or USB is not desirable.

To use the QR code scanner:

1. Open [https://op-txverify.optimism.io](https://op-txverify.optimism.io) on your mobile device.
2. Paste the transaction hash or Safe UI link of the Safe transaction you want to verify.
3. Run the following command on your verification device:
    ```bash
    op-txverify qr
    ```
4. A browser window will automatically open with the QR scanner interface.
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
