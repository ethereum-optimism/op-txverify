# Releasing

Releases are cut locally with [GoReleaser](https://goreleaser.com). There is no
release CI — whoever cuts the release runs it from their machine.

> **OP Labs employees:** a detailed internal runbook is maintained in Notion
> (access required):
> [Managing Releases of op-txverify](https://www.notion.so/oplabs/Managing-Releases-of-op-txverify-380f153ee16280b69a79c8c769b8177b).
> This file is the condensed, public version.

## Prerequisites

Tools: `go`, `just`, `goreleaser`, `golangci-lint`, `gpg`.

Environment variables:

| Variable | Purpose |
| --- | --- |
| `GITHUB_TOKEN` | GitHub token with `repo` scope (creates the draft release) |
| `GPG_FINGERPRINT` | Fingerprint of the GPG key used to sign the checksums file (key must be in your local keychain) |
| `MACOS_SIGN_P12` | Developer ID Application certificate — `.p12` file path or base64-encoded contents |
| `MACOS_SIGN_PASSWORD` | Password for the `.p12` |
| `MACOS_NOTARY_ISSUER_ID` | App Store Connect API key issuer ID |
| `MACOS_NOTARY_KEY_ID` | App Store Connect API key ID |
| `MACOS_NOTARY_KEY` | App Store Connect API key — `.p8` file path or base64-encoded contents |

macOS signing/notarization is skipped entirely if `MACOS_SIGN_P12` is unset.
The release will still succeed, but darwin binaries will be blocked by
Gatekeeper on users' machines — don't ship a real release without it.

## Cutting a release

1. Make sure you're on an up-to-date `main` with a clean working tree:

    ```bash
    git checkout main && git pull
    ```

2. Tag the release (GoReleaser derives the version from the tag):

    ```bash
    git tag vX.Y.Z
    git push origin vX.Y.Z
    ```

3. Run the release (runs tests and lint first):

    ```bash
    just release
    ```

    This builds linux/darwin × amd64/arm64 binaries, signs and notarizes the
    darwin binaries, GPG-signs the `SHA256SUMS` file, and creates a **draft**
    GitHub release.

4. Review the draft on the [releases page](https://github.com/ethereum-optimism/op-txverify/releases)
   — check the changelog and that all artifacts are attached (8 binaries,
   `SHA256SUMS`, `SHA256SUMS.sig`, source zip) — then publish it.

5. Spot-check a darwin binary passes Gatekeeper:

    ```bash
    spctl -a -vv -t install dist/op-txverify_*_darwin_arm64/op-txverify
    ```

To test the whole pipeline without publishing anything:

```bash
just release-dry-run
```

## Generating signing material

Do this when setting up a new release machine, rotating credentials, or
moving to a new Apple Developer account.

### GPG key

1. Generate a key: `gpg --full-generate-key` (ed25519, no expiry or a long one).
2. Get the fingerprint: `gpg --list-secret-keys --keyid-format long`.
3. Export `GPG_FINGERPRINT=<fingerprint>`.
4. Publish the public key somewhere users can find it (e.g. a keyserver via
   `gpg --send-keys`) so they can verify `SHA256SUMS.sig`.

### Apple Developer ID certificate (`MACOS_SIGN_P12`)

Requires an active [Apple Developer Program](https://developer.apple.com/programs/)
membership. Note: only the **Account Holder** can create Developer ID
certificates.

1. On a Mac, open Keychain Access → Certificate Assistant → Request a
   Certificate From a Certificate Authority. Save the CSR to disk.
2. At [developer.apple.com/account/resources/certificates](https://developer.apple.com/account/resources/certificates),
   create a certificate of type **Developer ID Application**, uploading the CSR.
3. Download the certificate and double-click to import it into your keychain.
4. In Keychain Access, find the "Developer ID Application: …" certificate,
   right-click → Export, and save as a password-protected `.p12`.
5. Export `MACOS_SIGN_P12` (path to the file, or `base64 < cert.p12`) and
   `MACOS_SIGN_PASSWORD`.

### App Store Connect API key (notarization)

1. At [App Store Connect](https://appstoreconnect.apple.com) → Users and
   Access → Integrations → App Store Connect API → Team Keys, generate a new
   key with the **Developer** role (creating keys requires Admin access).
2. Record the **Issuer ID** (`MACOS_NOTARY_ISSUER_ID`) and **Key ID**
   (`MACOS_NOTARY_KEY_ID`) shown on that page.
3. Download the `.p8` private key — Apple only lets you download it once.
   Export `MACOS_NOTARY_KEY` (path to the file, or its base64 contents).

Store all of the above in your team's secret manager (e.g. 1Password), not
just on the release machine.
