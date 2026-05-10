# Changelog

## Unreleased
- Fixed confirmation gating logic so “yes” or “y” responses allow execution and added `--yes` to bypass prompts entirely.
- Sequential dry runs now populate the results map, ensuring summaries include every visited directory.
- Parallel execution logs actual errors, prints captured output separately, and halts other workers when `--continue-on-failure` is not set.
- Reworked the CI workflow to run tests on Ubuntu and emit zipped binaries plus SHA256 checksums for Linux, macOS, and Windows (amd64/arm64) targets.
- Added OS-aware command execution for Linux, macOS, and Windows, including native Windows shell support and cross-platform SSH command selection.
- Release workflows now attach the built binary archives and SHA256 checksum files to GitHub Releases.
