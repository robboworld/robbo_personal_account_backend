# Licensing keys (local / container)

Generate (from `backend/`):

```bash
go run ./scripts/generate_licensing_keys/main.go ./keys/licensing
# or: openssl genrsa -out keys/licensing/dev-private.pem 2048
```

Files:
- `dev-private.pem` / `dev-public.pem` — RS256 for activation JWT + manifest (**gitignored** `*.pem`)
- `addon/paid-addon.js` — **placeholder** (committed) that registers `premium.auto_update` for smoke tests
- `addon/paid-addon.local.js` — optional override from private **`rs3-paid-addon`** (**gitignored**)

## Sync real paid-addon (local / staging)

```bash
# in rs3-paid-addon/
npm run build

# in backend/
./scripts/sync_paid_addon.sh
# or: RS3_PAID_ADDON_DIST=/abs/path/to/rs3-paid-addon/dist ./scripts/sync_paid_addon.sh

docker compose up -d --force-recreate app   # picks up volume ./keys/licensing
```

Server prefers `paid-addon.local.js` over `paid-addon.js`. Do **not** commit the private bundle.

For production set `LICENSING_PRIVATE_KEY_PATH` and `licensing.jwtKid=rs3-prod-1` (public key must match desktop `prodPublicKeyPem.js`).
