# Changelog

## Unreleased
- Fixed confirmation gating logic so “yes” or “y” responses allow execution and added `--yes` to bypass prompts entirely.
- Sequential dry runs now populate the results map, ensuring summaries include every visited directory.
- Parallel execution logs actual errors, prints captured output separately, and halts other workers when `--continue-on-failure` is not set.
