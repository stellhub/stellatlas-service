# StellAtlas Service

StellAtlas Service is the Go CMDB service for the Stell platform.

CMDB stands for Configuration Management Database. StellAtlas manages configuration items, asset inventory, topology relationships, ownership metadata, and lifecycle state for middleware and infrastructure resources.

## Naming

`stellatlas-service` follows the Stell service naming system and uses `Atlas` to represent an authoritative map of assets, dependencies, environments, and operational relationships.

## Responsibilities

- Manage configuration items and asset inventory.
- Track relationships between services, middleware, hosts, clusters, environments, and owners.
- Provide lifecycle metadata for discovery, governance, auditing, and platform operations.
- Serve as the backend CMDB service for future Stell platform control planes and consoles.

## API

- `GET /health`
- `GET /api/stellatlas/v1/status`

## Development

Run tests:

```bash
go test ./...
```

Run locally:

```bash
go run ./cmd/stellatlas-service
```

The default HTTP address is `:8080`. Override it with `STELLATLAS_HTTP_ADDR`.
