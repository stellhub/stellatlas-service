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

## Architecture

The service is built on [`github.com/stellhub/stellar`](https://github.com/stellhub/stellar). StellAtlas uses Stellar configuration loading, HTTP routing, the Chi HTTP adapter, Redis client wiring, PostgreSQL client wiring, health checks, and observability hooks.

StellAtlas keeps PostgreSQL as the CMDB source of truth and uses Redis as a read-through cache for high-frequency read models:

- application list
- application owner relation

Instance data uses the current snapshot model from PostgreSQL because instance state changes frequently and must stay tied to change events.

## API

- `GET /health`
- `GET /stellar/status`
- `GET /api/stellatlas/v1/status`
- `GET /api/stellatlas/v1/apps?env=prod&status=active&search=order`
- `GET /api/stellatlas/v1/apps/detail?app_id=stellaxis.payment.risk.antifraud.api`
- `POST /api/stellatlas/v1/apps`
- `PUT /api/stellatlas/v1/apps`
- `DELETE /api/stellatlas/v1/apps?app_id=stellaxis.payment.risk.antifraud.api`
- `GET /api/stellatlas/v1/app-owners?app_id=<uuid>`
- `GET /api/stellatlas/v1/app-instances?app_id=<uuid>&env=prod`

Application create/update requests validate the standard five-segment app ID from `docs/Standard.md`:

```text
organization.businessDomain.capabilityDomain.application.role
```

Example create request:

```json
{
  "app_id": "stellaxis.payment.risk.antifraud.api",
  "app_name": "Payment Risk Antifraud API",
  "environment": "prod",
  "status": "active",
  "lifecycle": "managed",
  "owner_team_code": "payment-platform",
  "owner_team_name": "Payment Platform",
  "language": "go",
  "repository_url": "https://git.example.com/stellaxis/payment/risk/antifraud/api"
}
```

## Storage

- `schema.sql` defines the PostgreSQL schema for CI core data, attributes, relations, application-person ownership, instance snapshots, change events, source records, baselines, and read models.
- `application.yml` enables the Stellar HTTP server, Redis client, and PostgreSQL client. PostgreSQL startup ping is disabled by default so the service can boot before local dependencies are created.

## Development

Run tests:

```bash
go test ./...
```

Run locally:

```bash
go run ./cmd/stellatlas-service
```

The default HTTP address is `:8080`. Change it in `application.yml`.

Load 100 temporary enterprise-style application IDs into PostgreSQL:

```bash
psql "postgres://user:password@host:5432/stellatlas-service?sslmode=disable" -f enterprise_apps.sql
```

Run the custom router example:

```bash
go run ./examples/http-custom-router-example
```
