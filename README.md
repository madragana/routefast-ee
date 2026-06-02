# RouteFast Enterprise Edition v1.0

[![CI](https://github.com/madragana/routefast-ee/actions/workflows/ci.yml/badge.svg)](https://github.com/madragana/routefast-ee/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-LICENSE--EE-red)](LICENSE-EE)

**Central credential server (`lipd-server`) for distributed network defense.**

RouteFast CE (Community Edition) nodes operate autonomously, detecting network
attacks and making mitigation decisions via distributed quorum voting.
**RouteFast EE** is the SaaS / on-premises control plane that:

- **Issues credentials** — ed25519 keypairs per node (rotated weekly)
- **Manages tokens** — 7-day bearer tokens with automatic refresh
- **Distributes policies** — quorum rules, mitigation templates, evidence schemas
- **Audits decisions** — append-only compliance log (SOC 2 / HIPAA ready)
- **Enforces mTLS** — all node-to-server communication is cryptographically verified

## Architecture

```
┌────────────────────────────────────┐
│   AOvidi control plane (lipd-server)│
│   ├── YugabyteDB (distributed PG)   │
│   ├── mTLS (TLS 1.3+)               │
│   └── 7 REST/JSON endpoints         │
└────────────────────────────────────┘
              ↕ HTTPS / mTLS
    ┌─────────┬─────────┬─────────┐
    │ Node A  │ Node B  │ Node C  │
    │  (CE)   │  (CE)   │  (CE)   │
    └─────────┴─────────┴─────────┘
```

- **Database**: YugabyteDB 2.20+ / 3.0+ (distributed PostgreSQL), via the `pgx/v5` driver
- **Transport**: TLS 1.3 with mutual authentication (mTLS)
- **API**: REST / JSON, 7 endpoints
- **Language**: Go 1.22+

## Quick Start (5 minutes)

```bash
git clone https://github.com/madragana/routefast-ee.git
cd routefast-ee

# One command: YugabyteDB + schema + dev certs
make setup

# Start the server (mTLS on :8443)
make run

# In another terminal, test the health endpoint
curl --cacert tls/ca.crt --cert tls/client.crt --key tls/client.key \
  https://localhost:8443/api/v1/health
```

## API

| # | Method | Path | Auth | Purpose |
|---|--------|------|------|---------|
| 1 | GET  | `/api/v1/health`        | mTLS        | Liveness + DB check |
| 2 | POST | `/api/v1/register`      | mTLS        | Provision a CE node, issue keypair + token |
| 3 | POST | `/api/v1/token/refresh` | mTLS + token| Rotate the 7-day bearer token |
| 4 | POST | `/api/v1/key/rotate`    | mTLS + token| Rotate the node ed25519 keypair |
| 5 | GET  | `/api/v1/policies`      | mTLS + token| Fetch policy bundle for a customer |
| 6 | POST | `/api/v1/audit/log`     | mTLS + token| Append a decision to the audit log |
| 7 | GET  | `/api/v1/audit/trail`   | mTLS + token| Query audit log by date range |

See [docs/API.md](docs/API.md) for full request/response details.

## Documentation

- [docs/QUICKSTART.md](docs/QUICKSTART.md) — local dev setup
- [docs/API.md](docs/API.md) — endpoint reference
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — design decisions
- [docs/OPERATIONS.md](docs/OPERATIONS.md) — deploy + monitor
- [docs/SECURITY.md](docs/SECURITY.md) — mTLS, compliance

## Licensing

- **Community Edition** libraries: Apache 2.0 (imported from `routefast-ce`)
- **Enterprise Edition** code: [LICENSE-EE](LICENSE-EE) (source-available, commercial)

## Development

```bash
make help          # list all targets
make setup         # full dev environment
make test          # tests + coverage
make lint          # go vet + gofmt
make docker-build  # build container
```

## Support

- GitHub: https://github.com/madragana/routefast-ee
- Issues: https://github.com/madragana/routefast-ee/issues
- Email: me@aovidi.com

---

**RouteFast** — Fighting AI with AI.
