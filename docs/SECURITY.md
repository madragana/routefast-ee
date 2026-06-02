# Security

## Transport
- TLS 1.3 minimum (`tls.VersionTLS13`).
- Mutual TLS: `RequireAndVerifyClientCert`. Clients without a CA-signed cert
  cannot complete the handshake.

## Identity & credentials
- Each node has an ed25519 keypair; the server keeps only the public key.
- Keypairs rotate weekly (`/key/rotate`); history is append-only in
  `key_rotations`.
- Bearer tokens are 256 bits of CSPRNG entropy, valid 7 days, refreshable.
  Refresh revokes the prior token.

## Audit & compliance
- `audit_logs` is append-only — records are inserted, never updated or deleted.
- The audit trail endpoint supports date-range export for SOC 2 / HIPAA
  evidence.

## Secrets handling
- TLS material is read from disk paths supplied via environment variables.
- `tls/` and `.env` are git-ignored.
- The dev CA produced by `gen-certs` is for local use only — issue production
  certs from your own PKI.

## Reporting
Email security@aovidi.com. Please do not open public issues for vulnerabilities.
