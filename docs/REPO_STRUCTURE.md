# Repository Structure

```
routefast-ee/
├── cmd/
│   ├── lipd-server/        # API server entry point (main.go)
│   └── gen-certs/          # dev mTLS certificate generator
├── internal/
│   ├── api/                # HTTP handlers + middleware (7 endpoints)
│   ├── config/             # env-based configuration
│   ├── crypto/             # ed25519 keypairs + bearer tokens
│   └── storage/            # pgx/YugabyteDB data-access layer
├── migrations/             # ordered SQL migrations (001–003)
├── deployments/
│   ├── docker/             # production Dockerfile
│   ├── docker-compose/     # local YugabyteDB + migrate job
│   ├── kubernetes/helm/    # Helm chart (stub)
│   └── terraform/          # aws / gcp / azure IaC (stubs)
├── tests/
│   ├── integration/        # full-workflow tests
│   ├── storage/            # storage-layer tests
│   └── fixtures/           # seed.sql + test data
├── scripts/                # setup-dev / migrate / test
├── docs/                   # API, ARCHITECTURE, SECURITY, OPERATIONS, QUICKSTART
├── examples/               # curl client examples
├── .github/workflows/      # ci.yml, release.yml
├── go.mod
├── Makefile
├── README.md
├── LICENSE                 # Apache 2.0 (CE libs)
└── LICENSE-EE              # commercial source-available
```

## Dependency on CE
`routefast-ee` imports the decision schema, LIP-4D protocol, and quorum logic
from `routefast-ce` (`github.com/madragana/routefast-ce`). Add that require
entry to `go.mod` when wiring the shared types in.
