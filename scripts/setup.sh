#!/usr/bin/env bash
set -euo pipefail

echo "=============================================="
echo " Setting up engage-engine development environment"
echo "=============================================="

# 1. Check Go installation
if ! command -v go >/dev/null 2>&1; then
    echo "[-] Error: Go is not installed. Please install Go 1.22+."
    exit 1
fi
echo "[+] Go detected: $(go version)"

# 2. Check Docker installation (optional for local infra)
if command -v docker >/dev/null 2>&1; then
    echo "[+] Docker detected: $(docker --version)"
else
    echo "[!] Notice: Docker not found. You will need external PostgreSQL, MongoDB, Redis, and Kafka instances."
fi

# 3. Check Task / Make
if command -v task >/dev/null 2>&1; then
    echo "[+] Task detected: $(task --version)"
elif command -v make >/dev/null 2>&1; then
    echo "[+] Make detected: $(make --version | head -n 1)"
fi

# 4. Create .env if it does not exist
if [ ! -f .env ]; then
    echo "[+] Creating .env from .env.example..."
    cp .env.example .env
else
    echo "[+] .env already exists. Skipping creation."
fi

# 5. Verify Go modules
echo "[+] Verifying go.mod..."
go mod verify

echo "=============================================="
echo " Setup complete! Next steps:"
echo "   make build     # (or 'task build') Compile API and worker binaries"
echo "   make test      # (or 'task test') Run test suite"
echo "   make docker-up # (or 'task docker-up') Spin up PostgreSQL, MongoDB, Redis, Kafka"
echo "=============================================="

