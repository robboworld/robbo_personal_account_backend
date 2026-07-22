#!/usr/bin/env bash
# Sync private rs3-paid-addon dist into local licensing addon dir (not committed).
# Usage (from backend/):
#   ./scripts/sync_paid_addon.sh
#   RS3_PAID_ADDON_DIST=/path/to/rs3-paid-addon/dist ./scripts/sync_paid_addon.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${ROOT}/keys/licensing/addon"
OUT_FILE="${OUT_DIR}/paid-addon.local.js"

DEFAULT_DIST=""
if [[ -d "${ROOT}/../../RS/rs3-paid-addon/dist" ]]; then
  DEFAULT_DIST="$(cd "${ROOT}/../../RS/rs3-paid-addon/dist" && pwd)"
elif [[ -d "${ROOT}/../../../RS/rs3-paid-addon/dist" ]]; then
  DEFAULT_DIST="$(cd "${ROOT}/../../../RS/rs3-paid-addon/dist" && pwd)"
elif [[ -d "${HOME}/Projects/RS/rs3-paid-addon/dist" ]]; then
  DEFAULT_DIST="${HOME}/Projects/RS/rs3-paid-addon/dist"
fi

DIST="${RS3_PAID_ADDON_DIST:-$DEFAULT_DIST}"
if [[ -z "$DIST" || ! -f "${DIST}/paid-addon.js" ]]; then
  echo "paid-addon.js not found. Build rs3-paid-addon (npm run build) and set RS3_PAID_ADDON_DIST." >&2
  exit 1
fi

mkdir -p "$OUT_DIR"
cp -f "${DIST}/paid-addon.js" "$OUT_FILE"
echo "synced ${DIST}/paid-addon.js -> ${OUT_FILE}"
echo "Restart/recreate backend app so LICENSING_ADDON_STATIC_DIR picks it up (volume mount keys/licensing)."
