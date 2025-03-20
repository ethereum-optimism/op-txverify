# op-verify

Utilities for verifying Safe transactions.

## Installation

### Option 1: Download from Releases

1. Go to the [Releases](https://github.com/yourusername/op-verify/releases) page
2. Download the appropriate binary for your operating system and architecture
3. Make the binary executable: `chmod +x op-verify_[version]_[os]_[arch]`
4. Rename and move the binary to a location in your PATH:
    ```bash
    mv op-verify_[version]_[os]_[arch] /usr/local/bin/op-verify
    ```

#### Optional: Verify Checksum

To verify the integrity of your downloaded binary:

1. Download the `op-verify_[version]_SHA256SUMS` file from the releases page
1. Run the verification command and compare with the corresponding entry in the SHA256SUMS file:
    ```bash
    sha256sum op-verify_[version]_[os]_[arch]
    ```

### Option 2: Build from Source

Prerequisites:

- Go 1.23 or later
- Git
- [GoReleaser](https://goreleaser.com/install/)

Steps:

1. Clone the repository:
    ```bash
    git clone https://github.com/yourusername/op-verify.git
    cd op-verify
    ```
1. Build using GoReleaser:
    ```bash
    goreleaser build --snapshot --clean
    ```
1. The binaries will be available in the `dist` directory with names like:
    ```bash
    dist/op-verify_[version]_[os]_[arch]/op-verify
    ```
1. Install the binary for your platform:
    ```bash
    chmod +x dist/op-verify_[version]_[os]_[arch]/op-verify
    mv dist/op-verify_[version]_[os]_[arch]/op-verify /usr/local/bin/
    ```

#### Compare Checksums (Optional)

To verify a downloaded binary against your local build:

1. Check the SHA256SUMS file generated in the `dist` directory:
    ```bash
    cat dist/op-verify_[version]_SHA256SUMS
    ```
1. Calculate the checksum of your downloaded binary:
    ```bash
    sha256sum /path/to/downloaded/op-verify_[version]_[os]_[arch]
    ```
1. Compare the two checksums to ensure they match
