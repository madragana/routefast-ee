#!/usr/bin/env bash
set -euo pipefail
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
echo "coverage report: coverage.html"
