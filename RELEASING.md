# Releasing

Releases are local builds published as draft GitHub releases. There is no
release CI: whoever cuts the release runs the build from their machine, checks
the draft, then publishes it.

`op-txverify` does not sign or notarize macOS binaries. This keeps releases
independent from Apple Developer credentials, but macOS may require users to
approve or unquarantine downloaded binaries. The README documents the user-side
install path.

## Prerequisites

Tools:

- `go`
- `just`
- `goreleaser`
- `golangci-lint`
- `gh`

GitHub access:

- You need write access to `ethereum-optimism/op-txverify`.
- You need permission to create release tags.
- Org rules make tags immutable, so verify the tag before pushing it.

Authenticate `gh` before cutting a release:

```bash
gh auth status
```

## Cutting a release

1. Start from an up-to-date `main` with a clean worktree:

    ```bash
    git checkout main
    git pull --ff-only
    git status --short
    ```

2. Pick the release version and create the tag locally:

    ```bash
    git tag vX.Y.Z
    ```

3. Push the tag:

    ```bash
    git push origin vX.Y.Z
    ```

    Do NOT push the tag until the version is final. Org rules make tags
    immutable after creation.

4. Build artifacts and create the draft GitHub release:

    ```bash
    just release vX.Y.Z
    ```

    This checks that the current commit is tagged `vX.Y.Z`, runs tests and
    lint, builds linux/darwin binaries for amd64/arm64, copies them into the
    filenames referenced by `SHA256SUMS`, uploads the artifacts with
    `gh release create`, and leaves the release as a draft.

5. Review the draft on the [releases page](https://github.com/ethereum-optimism/op-txverify/releases):

    - Check that the generated notes are reasonable.
    - Check that the assets include four binaries, `SHA256SUMS`, and the source
      zip.
    - Download one asset and verify it against `SHA256SUMS`.

6. Publish the draft release from GitHub.

To test the build without creating a release:

```bash
just release-dry-run
```

## Checksums

Releases include `SHA256SUMS`, but do not include a GPG signature for that file.
The checksum protects users from accidental corruption or partial downloads. It
does not prove publisher identity if someone can modify both the binary and the
checksum on GitHub.

If stronger release provenance is needed later, add it with a real trust model,
such as CI-built artifacts and GitHub/Sigstore attestations. Do not reintroduce
local GPG key management unless users have a documented, trusted public key and
are expected to verify it.
