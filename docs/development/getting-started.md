# Getting started

This guide is the supported local-development contract for OpenRevenue. It is
designed to be repeatable on macOS and Linux and uses fictional local data only.

## Supported toolchain

| Tool | Supported version |
| --- | --- |
| Go | 1.26.5 |
| Node.js | 22.23.1 |
| pnpm | 10.14.0 |
| Docker Engine/Desktop | Engine 27.0+ with Compose 2.30+ |
| PostgreSQL | 17.10 |
| Redis | 7.4.9 |
| MinIO | RELEASE.2025-09-07T16-13-09Z |

Go, Node, and pnpm are declared in `.tool-versions`; Node is also declared in
`.nvmrc` and `.node-version`. `go.mod` selects the exact Go toolchain and
`package.json` selects the exact package manager. Local service images are
pinned in `.env.example` and have identical fallbacks in `docker-compose.yml`.

You may use mise, asdf, fnm, nvm, or another version manager. Docker Desktop is
the simplest way to obtain Docker and Compose on macOS; on Linux, install Docker
Engine and its Compose plugin.

## Clean-machine setup

```sh
git clone https://github.com/opencorex-org/openrevenue.git
cd openrevenue
mise install                 # or install versions from .tool-versions manually
corepack enable
make bootstrap
make services-up
make quality
```

`make bootstrap` is idempotent. It validates prerequisites, creates `.env` from
the safe example only when `.env` does not exist, downloads Go modules, and
installs the frozen pnpm lockfile. Existing developer configuration is never
overwritten.

If setup fails, run:

```sh
make doctor
```

The doctor reports missing tools, version mismatches, a stopped Docker daemon,
missing Compose support, and missing local configuration with a concrete
remediation for each problem.

## Run the platform

Start the pinned local services and wait for core dependencies to become
healthy:

```sh
make services-up
make api
```

In separate terminals, optional processes can be started with `make worker`,
`make scheduler`, and `make web`.

| Service | Local endpoint |
| --- | --- |
| API health | http://localhost:8080/health |
| API metrics | http://localhost:8080/metrics |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |
| MinIO API / console | http://localhost:9000 / http://localhost:9001 |
| Mailpit SMTP / UI | localhost:1025 / http://localhost:8025 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3001 |
| OTLP gRPC / HTTP | localhost:4317 / localhost:4318 |

Use `make services-status`, `make services-logs`, and `make services-down` to
operate the stack. `services-down` preserves named-volume data.

## Quality commands

`make quality` is the complete pre-push baseline and matches CI:

```sh
make format-check
make lint
make typecheck
make test
make build
```

Run `make format` to apply supported Go and frontend formatting. Run `make help`
to list every supported command.

## Configuration and data safety

`.env.example` contains only fictional localhost credentials marked
`openrevenue_dev_only`. Bootstrap copies it to the ignored `.env` file. Never
commit `.env`, private keys, production credentials, taxpayer exports, database
dumps, or real personal data. Development seeds must remain fictional and must
be reviewed for privacy before merging.

The local stack is not hardened for internet exposure or production use. Real
deployments require a secret manager, TLS, credential rotation, restricted
networking, backups, audit controls, and production observability.

## Database setup

After `make services-up`, install `golang-migrate` and the PostgreSQL client if
you need to apply or inspect migrations:

```sh
set -a
. ./.env
set +a
make migrate
make seed
```

Migrations are forward-only. Seed data is explicitly fictional and intended
only for local development.
