# op-txverify

op-txverify is a command-line utility for verifying [Safe](https://app.safe.global) transactions. It helps users validate Safe multisig transactions by:

1. Parsing and presenting transactions in a clearly understandable manner
2. Identifying common addresses and contracts that users interact with
3. Simplifying access by allowing users to input transactions via QR codes

## QR Code Scanning

The QR scanning functionality provided by `op-txverify` allows you to verify Safe transactions by scanning QR codes displayed on a web interface. This is especially useful for air-gapped verification where transmitting data to the verification device over bluetooth or USB is not desireable.

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

### Option 1: Download from Releases

1. Go to the [Releases](https://github.com/ethereum-optimism/op-txverify/releases) page
2. Download the appropriate binary for your operating system and architecture
3. Make the binary executable: `chmod +x op-txverify_[version]_[os]_[arch]`
4. Rename and move the binary to a location in your PATH:
    ```bash
    mv op-txverify_[version]_[os]_[arch] /usr/local/bin/op-txverify
    ```

#### Optional: Verify Checksum

To verify the integrity of your downloaded binary:

1. Download the `op-txverify_[version]_SHA256SUMS` file from the releases page
1. Run the verification command and compare with the corresponding entry in the SHA256SUMS file:
    ```bash
    sha256sum op-txverify_[version]_[os]_[arch]
    ```
1. Compare the two checksums to ensure they match

### Option 2: Build from Source

Prerequisites:

- Go 1.23 or later
- Git
- [GoReleaser](https://goreleaser.com/install/)

Steps:

1. Clone the repository:
    ```bash
    git clone https://github.com/yourusername/op-txverify.git
    cd op-txverify
    ```
1. Build using GoReleaser:
    ```bash
    goreleaser build --snapshot --clean
    ```
1. The binaries will be available in the `dist` directory with names like:
    ```bash
    dist/op-txverify_[version]_[os]_[arch]/op-txverify
    ```
1. Install the binary for your platform:
    ```bash
    chmod +x dist/op-txverify_[version]_[os]_[arch]/op-txverify
    mv dist/op-txverify_[version]_[os]_[arch]/op-txverify /usr/local/bin/
    ```

#### Compare Checksums (Optional)

To verify a downloaded binary against your local build:

1. Check the SHA256SUMS file generated in the `dist` directory:
    ```bash
    cat dist/op-txverify_[version]_SHA256SUMS
    ```
1. Calculate the checksum of your downloaded binary:
    ```bash
    sha256sum /path/to/downloaded/op-txverify_[version]_[os]_[arch]
    ```
1. Compare the two checksums to ensure they match
