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

### Download the latest release

Use the GitHub CLI to download the latest release. Pick the pattern that
matches your machine:

```bash
mkdir -p /tmp/op-txverify-download
cd /tmp/op-txverify-download

gh release download \
  --repo ethereum-optimism/op-txverify \
  --pattern 'op-txverify_*_darwin_arm64' \
  --pattern 'op-txverify_*_SHA256SUMS'

shasum -a 256 --check --ignore-missing op-txverify_*_SHA256SUMS

chmod +x op-txverify_*_darwin_arm64
sudo mv op-txverify_*_darwin_arm64 /usr/local/bin/op-txverify
```

Other common asset patterns:

- Apple silicon macOS: `op-txverify_*_darwin_arm64`
- Intel macOS: `op-txverify_*_darwin_amd64`
- Linux amd64: `op-txverify_*_linux_amd64`
- Linux arm64: `op-txverify_*_linux_arm64`

The release binaries are not signed or notarized by Apple. Apple documents that
macOS checks downloaded software from outside the App Store, including whether
it is signed by an identified developer and notarized. Browser downloads are
more likely to carry quarantine metadata that triggers the Privacy & Security
override flow for unsigned software. Prefer `gh release download`, verify the
checksum, and install the binary from Terminal.

If macOS still blocks the binary and you have verified the checksum, remove the
quarantine attribute:

```bash
xattr -d com.apple.quarantine /usr/local/bin/op-txverify
```

### Build from source

If there is no usable release yet, build from source with Go:

Prerequisites:

- Go 1.23 or later
- Git

Steps:

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

You can also install directly from GitHub:

```bash
go install github.com/ethereum-optimism/op-txverify/cmd/op-txverify@main
```

Maintainer release instructions live in [RELEASING.md](RELEASING.md).
