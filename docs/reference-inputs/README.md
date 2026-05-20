# Reference Inputs

This directory stores repo-local reference material used by the product requirements, implementation plan, design system, and migration notes.

Reference inputs should be checked in when they are needed to make the repository portable and when they do not contain secrets or unsafe production data. Do not place credentials, tokens, auth headers, private keys, passwords, client secrets, raw service-account JSON, or unmasked sensitive source-system exports in this directory.

## Current Checked-In Assets

- `branding/Firefly.png`: mascot mark used by the DEV UI and design mocks.
- `branding/google-g.png`: Google sign-in mark used by the DEV login surface.
- `branding/Wordmarks/Gold W black outline.png`: district wordmark reference asset.

## Inventory

Use [VENDORED_INVENTORY.md](VENDORED_INVENTORY.md) as the durable inventory for checked-in reference inputs and any intentionally deferred snapshots.
