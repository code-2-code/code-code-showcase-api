# code-code-showcase-api

Showcase-facing HTTP API for Code Code.

This repository owns:

- `packages/showcase-api`: showcase handlers, public read projections, HTTP
  server wiring, and telemetry setup.
- `code-code-contracts`: generated shared contracts as a Git submodule.

Useful checks:

```bash
cd packages/showcase-api && go test ./...
```
