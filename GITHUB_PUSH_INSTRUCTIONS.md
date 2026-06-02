# Push to GitHub

```bash
# 1. Create an empty repo on GitHub: madragana/routefast-ee

# 2. From the unzipped folder:
cd routefast-ee
git init -b main

# 3. Initialize Go modules / pull deps (needs internet)
go mod tidy        # resolves pgx/v5 and writes go.sum

# 4. First commit
git add .
git commit -m "RouteFast EE v1.0: lipd-server scaffold (mTLS, YugabyteDB, 7 endpoints)"

# 5. Connect remote and push
git remote add origin git@github.com:madragana/routefast-ee.git
git push -u origin main
```

## Notes
- `go.sum` is intentionally absent; `go mod tidy` generates it on first run.
- The `tls/` directory is git-ignored — never commit certificates.
- Helm chart and Terraform dirs are stubs (`.gitkeep`); fill in per environment.
- To wire in shared CE types, add `require github.com/madragana/routefast-ce vX.Y.Z`
  to go.mod and import the decision schema / LIP-4D / quorum packages.
