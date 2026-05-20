#!/usr/bin/env bash
# ----------------------------------------------------------------------------
# scripts/run.sh
#
# Helper for running any go-bybit example with variables loaded from .env.
#
# USAGE:
#   ./scripts/run.sh ./examples/account-info
#   ./scripts/run.sh ./examples/simple-trade
#   ./scripts/run.sh ./examples/inventory-tracker
#
# BEHAVIOUR:
#   - Loads every variable from <repo>/.env (if present).
#   - Exits with a clear message if .env is missing.
#   - Forwards all positional args to `go run`.
# ----------------------------------------------------------------------------

set -euo pipefail

readonly ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly ENV_FILE="${ROOT_DIR}/.env"

if [[ ! -f "${ENV_FILE}" ]]; then
    echo "error: ${ENV_FILE} not found." >&2
    echo "       cp .env.example .env  &&  edit it with your keys" >&2
    exit 1
fi

if [[ $# -lt 1 ]]; then
    echo "usage: $0 <go-package-path>" >&2
    echo "  e.g.: $0 ./examples/simple-trade" >&2
    exit 1
fi

set -a
# shellcheck disable=SC1090
source "${ENV_FILE}"
set +a

cd "${ROOT_DIR}"
exec go run "$@"
